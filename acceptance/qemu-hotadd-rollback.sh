#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${NETLAB_BASE_URL:-http://127.0.0.1:8088}
STATE_DIR=${NETLAB_STATE_DIR:-/var/lib/netlab}
DATABASE=${NETLAB_DATABASE:-$STATE_DIR/netlab.db}
LAB_NAME=${NETLAB_ACCEPTANCE_LAB:-qemu-hotadd-rollback-$(date -u +%Y%m%dT%H%M%SZ)}
POLL_SECONDS=${NETLAB_ACCEPTANCE_POLL_SECONDS:-180}
LAB_ID=""
NODE_ID=""

api() {
  local method=$1 path=$2 body=${3:-} revision=${4:-}
  local args=(-fsS -X "$method" -H 'Accept: application/json')
  if [[ $method != GET && $method != HEAD ]]; then
    args+=(-H "Idempotency-Key: $(cat /proc/sys/kernel/random/uuid)")
  fi
  [[ -z $revision ]] || args+=(-H "If-Match: $revision")
  [[ -z $body ]] || args+=(-H 'Content-Type: application/json' --data "$body")
  curl "${args[@]}" "$BASE_URL/api/v1$path"
}

qmp() {
  local command=$1 arguments=${2:-'{}'} socket="$STATE_DIR/runtime/qemu/$NODE_ID/qmp.sock"
  python3 - "$socket" "$command" "$arguments" <<'PY'
import json
import socket
import sys

path, command, encoded_arguments = sys.argv[1:]
request_id = "acceptance"

with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as client:
    client.settimeout(10)
    client.connect(path)
    stream = client.makefile("rwb", buffering=0)

    def receive():
        while True:
            line = stream.readline()
            if not line:
                raise RuntimeError("QMP socket closed")
            value = json.loads(line)
            if "event" not in value:
                return value

    banner = receive()
    if "QMP" not in banner:
        raise RuntimeError(f"invalid QMP banner: {banner}")
    stream.write((json.dumps({"execute": "qmp_capabilities", "id": "capabilities"}) + "\r\n").encode())
    response = receive()
    if "error" in response:
        raise RuntimeError(json.dumps(response["error"], sort_keys=True))
    request = {"execute": command, "arguments": json.loads(encoded_arguments), "id": request_id}
    stream.write((json.dumps(request) + "\r\n").encode())
    while True:
        response = receive()
        if response.get("id") == request_id:
            print(json.dumps(response, sort_keys=True))
            break
PY
}

wait_task_state() {
  local id=$1 expected=$2 deadline=$((SECONDS + POLL_SECONDS)) task state
  while (( SECONDS < deadline )); do
    task=$(api GET "/tasks/$id")
    state=$(jq -r '.state' <<<"$task")
    if [[ $state == "$expected" ]]; then
      printf '%s\n' "$task"
      return 0
    fi
    case $state in failed|cancelled|succeeded)
      echo "task $id reached $state instead of $expected" >&2
      printf '%s\n' "$task" >&2
      return 1
      ;;
    esac
    sleep 1
  done
  echo "task $id timed out waiting for $expected" >&2
  return 1
}

wait_node_state() {
  local expected=$1 deadline=$((SECONDS + POLL_SECONDS)) node state
  while (( SECONDS < deadline )); do
    node=$(api GET "/nodes/$NODE_ID")
    state=$(jq -r '.observed_state' <<<"$node")
    [[ $state == "$expected" ]] && return 0
    [[ $state == failed ]] && { printf '%s\n' "$node" >&2; return 1; }
    sleep 1
  done
  echo "node $NODE_ID did not reach $expected" >&2
  return 1
}

set_node_state() {
  local desired=$1 node revision
  node=$(api GET "/nodes/$NODE_ID")
  revision=$(jq -r '.revision' <<<"$node")
  api PUT "/nodes/$NODE_ID/state" "$(jq -nc --arg state "$desired" '{desired_state:$state}')" "$revision" >/dev/null
  wait_node_state "$desired"
}

cleanup() {
  [[ -n $LAB_ID ]] || return 0
  local lab revision deadline
  lab=$(api GET "/labs/$LAB_ID" 2>/dev/null) || return 0
  revision=$(jq -r '.laboratory.revision' <<<"$lab")
  api DELETE "/labs/$LAB_ID" '' "$revision" >/dev/null 2>&1 || true
  deadline=$((SECONDS + POLL_SECONDS))
  while (( SECONDS < deadline )); do
    api GET "/labs/$LAB_ID" >/dev/null 2>&1 || break
    sleep 1
  done
}
trap cleanup EXIT

for tool in curl jq python3 sqlite3 ip; do
  command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 2; }
done
[[ -r $DATABASE ]] || { echo "database is not readable: $DATABASE" >&2; exit 2; }

images=$(api GET /images)
templates=$(api GET /templates)
version=$(jq -r --argjson images "$images" '
  [ $images[] | select(.runtime_kind == "qemu" and .availability == "available") | .id ] as $legal |
  first(.[] | .versions[] | select(.enabled == true and (.image_version_id as $id | $legal | index($id))) | .id) // empty
' <<<"$templates")
if [[ -z $version ]]; then
  echo "SKIP: one operator-registered available QEMU template version is required." >&2
  exit 77
fi

lab=$(api POST /labs "$(jq -nc --arg name "$LAB_NAME" '{name:$name,recovery_policy:"auto_restore"}')")
LAB_ID=$(jq -r '.id' <<<"$lab")
created=$(api POST "/labs/$LAB_ID/nodes" "$(jq -nc --arg version "$version" '{name:"rollback-qemu",template_version_id:$version,cpu_count:2,cpu_quota_micros:100000,memory_mib:1024,storage_gib:8,interface_limit:4,interface_count:1,process_limit:256}')")
NODE_ID=$(jq -r '.node.id' <<<"$created")
node=$(api GET "/nodes/$NODE_ID")
api PUT "/nodes/$NODE_ID/state" '{"desired_state":"running"}' "$(jq -r '.revision' <<<"$node")" >/dev/null
wait_node_state running

filled=0
bus_full=0
for slot in $(seq 0 63); do
  blocker_response=$(qmp device_add "$(jq -nc --arg id "netlab-failure-blocker-$slot" '{driver:"virtio-rng-pci",id:$id,bus:"netlab-rp-1"}')")
  if jq -e 'has("return")' <<<"$blocker_response" >/dev/null; then
    filled=$((filled + 1))
    continue
  fi
  if jq -e 'has("error") and (.error.desc | test("slot|space|address|function"; "i"))' <<<"$blocker_response" >/dev/null; then
    bus_full=1
  fi
  break
done
(( filled > 0 )) || { echo "unable to occupy the target QEMU PCI bus" >&2; exit 1; }
(( bus_full == 1 )) || { echo "target QEMU PCI bus did not reach capacity after $filled devices" >&2; exit 1; }

failed_response=$(api POST "/nodes/$NODE_ID/interfaces" '{"driver":"virtio-net-pci"}')
FAILED_INTERFACE_ID=$(jq -r '.interface.id' <<<"$failed_response")
FAILED_TASK_ID=$(jq -r '.task.id' <<<"$failed_response")
FAILED_TAP="nlt${FAILED_INTERFACE_ID:0:12}"
failed_task=$(wait_task_state "$FAILED_TASK_ID" failed)
jq -e '.error.code == "interface_hot_add_failed" and .error.phase == "hot_add" and .error.cleanup == "interface row and TAP removed"' <<<"$failed_task" >/dev/null

snapshot=$(api GET "/labs/$LAB_ID")
if jq -e --arg id "$FAILED_INTERFACE_ID" '.interfaces[] | select(.id == $id)' <<<"$snapshot" >/dev/null; then
  echo "failed interface row still exists: $FAILED_INTERFACE_ID" >&2
  exit 1
fi
if ip link show "$FAILED_TAP" >/dev/null 2>&1; then
  echo "failed TAP still exists: $FAILED_TAP" >&2
  exit 1
fi
if (( $(sqlite3 "$DATABASE" "SELECT count(*) FROM runtime_ownership WHERE resource_id='$FAILED_INTERFACE_ID';") != 0 )); then
  echo "failed interface ownership rows still exist: $FAILED_INTERFACE_ID" >&2
  exit 1
fi
netdev_probe=$(qmp netdev_del "{\"id\":\"net-$FAILED_INTERFACE_ID\"}")
if jq -e 'has("return")' <<<"$netdev_probe" >/dev/null; then
  echo "failed QMP netdev leaked and was removed by the probe: net-$FAILED_INTERFACE_ID" >&2
  exit 1
fi
jq -e 'has("error") and (.error.desc | test("not found"; "i"))' <<<"$netdev_probe" >/dev/null

set_node_state stopped
set_node_state running

retry_response=$(api POST "/nodes/$NODE_ID/interfaces" '{"driver":"virtio-net-pci"}')
RETRY_INTERFACE_ID=$(jq -r '.interface.id' <<<"$retry_response")
RETRY_TASK_ID=$(jq -r '.task.id' <<<"$retry_response")
wait_task_state "$RETRY_TASK_ID" succeeded >/dev/null
snapshot=$(api GET "/labs/$LAB_ID")
jq -e --arg id "$RETRY_INTERFACE_ID" '.interfaces[] | select(.id == $id and .slot == 1)' <<<"$snapshot" >/dev/null

echo "PASS: hot-add rollback compensated interface=$FAILED_INTERFACE_ID and reused slot=1 with interface=$RETRY_INTERFACE_ID"
