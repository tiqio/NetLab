#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${NETLAB_BASE_URL:-http://127.0.0.1:8088}"
BASE_URL="${BASE_URL%/}"
BASE="$BASE_URL/api/v1"
DB="${NETLAB_DB:-/var/lib/netlab/netlab.db}"
STATE="${NETLAB_STATE_DIR:-/var/lib/netlab}"
LAB_ID=""
CONSOLE_PID=""
read -r EVENT_HOST EVENT_PORT < <(python3 - "$BASE_URL" <<'PY'
import sys
from urllib.parse import urlsplit

url = urlsplit(sys.argv[1])
if url.scheme != "http" or not url.hostname:
    raise SystemExit("NETLAB_BASE_URL must be an http URL for raw WebSocket recovery checks")
print(url.hostname, url.port or 80)
PY
)

api() {
  local method=$1 path=$2 body=${3:-} revision=${4:-}
  local args=(-fsS -X "$method" -H 'Accept: application/json')
  [[ $method == GET ]] || args+=(-H "Idempotency-Key: $(cat /proc/sys/kernel/random/uuid)")
  [[ -z $revision ]] || args+=(-H "If-Match: $revision")
  [[ -z $body ]] || args+=(-H 'Content-Type: application/json' --data "$body")
  curl "${args[@]}" "$BASE$path"
}

netlab_mount_exec() {
  local pid
  pid=$(systemctl show --property MainPID --value netlab.service)
  [[ $pid =~ ^[1-9][0-9]*$ ]] || { echo "netlab.service has no main PID" >&2; return 1; }
  nsenter --target "$pid" --mount -- "$@"
}

wait_task() {
  local id=$1 deadline=$((SECONDS + 240)) value state
  while ((SECONDS < deadline)); do
    value=$(api GET "/tasks/$id")
    state=$(jq -r .state <<<"$value")
    case $state in
      succeeded) echo "$value"; return ;;
      failed|cancelled) echo "$value" >&2; return 1 ;;
    esac
    sleep 1
  done
  return 1
}

wait_node() {
  local id=$1 expected=$2 deadline=$((SECONDS + 240)) value state retryable
  while ((SECONDS < deadline)); do
    value=$(api GET "/nodes/$id")
    state=$(jq -r .observed_state <<<"$value")
    [[ $state == "$expected" ]] && return
    if [[ $state == failed ]]; then
      retryable=$(jq -r '.last_error.retryable // false' <<<"$value")
      [[ $retryable == true ]] || { echo "$value" >&2; return 1; }
    fi
    sleep 1
  done
  echo "$value" >&2
  return 1
}

set_state() {
  local id=$1 desired=$2 value revision task
  value=$(api GET "/nodes/$id")
  revision=$(jq -r .revision <<<"$value")
  task=$(api PUT "/nodes/$id/state" "{\"desired_state\":\"$desired\"}" "$revision" | jq -r .task.id)
  wait_task "$task" >/dev/null
}

cleanup() {
  [[ -z $CONSOLE_PID ]] || kill "$CONSOLE_PID" 2>/dev/null || true
  [[ -n $LAB_ID ]] || return
  local value revision task deadline
  value=$(api GET "/labs/$LAB_ID" 2>/dev/null) || return 0
  revision=$(jq -r .laboratory.revision <<<"$value")
  task=$(api DELETE "/labs/$LAB_ID" '' "$revision" 2>/dev/null | jq -r '.task.id // empty') || true
  [[ -z $task ]] || wait_task "$task" >/dev/null 2>&1 || true
  deadline=$((SECONDS + 240))
  while ((SECONDS < deadline)); do
    api GET "/labs/$LAB_ID" >/dev/null 2>&1 || return 0
    sleep 1
  done
}

trap cleanup EXIT

name="t225-restart-$(date -u +%Y%m%dT%H%M%SZ)"
LAB_ID=$(api POST /labs "$(jq -nc --arg name "$name" '{name:$name,recovery_policy:"auto_restore"}')" | jq -r .id)
qemu_version=$(api GET /templates | jq -r 'first(.[] | select(.template_key=="ubuntu-qemu") | .versions[] | select(.enabled) | .id)')
qemu_image=$(api GET /images | jq -r 'first(.[] | select(.runtime_kind=="qemu" and .availability=="available" and (.name | ascii_downcase)=="ubuntu") | .id)')
[[ -n $qemu_version && $qemu_version != null ]] || { echo 'no enabled ubuntu-qemu template version' >&2; exit 77; }
[[ -n $qemu_image && $qemu_image != null ]] || { echo 'no available Ubuntu QEMU image' >&2; exit 77; }

LAB_REVISION=$(api GET "/labs/$LAB_ID" | jq -r .laboratory.revision)
qemu=$(api POST "/labs/$LAB_ID/nodes" "$(jq -nc --arg version "$qemu_version" --arg image "$qemu_image" '{name:"qemu",template_version_id:$version,image_version_id:$image,cpu_count:2,memory_mib:1024,storage_gib:8,interface_count:2,interface_limit:4}')" "$LAB_REVISION")
QEMU_NODE=$(jq -r .node.id <<<"$qemu")
QEMU_IF0=$(jq -r '.interfaces[0].id' <<<"$qemu")
QEMU_IF1=$(jq -r '.interfaces[1].id' <<<"$qemu")

LAB_REVISION=$(api GET "/labs/$LAB_ID" | jq -r .laboratory.revision)
docker=$(api POST "/labs/$LAB_ID/nodes" '{"name":"docker","kind":"docker","cpu_count":1,"memory_mib":128,"interface_count":1,"config":{"image":"busybox:latest","command":["sleep","86400"]}}' "$LAB_REVISION")
DOCKER_NODE=$(jq -r .node.id <<<"$docker")
DOCKER_IF=$(jq -r '.interfaces[0].id' <<<"$docker")

LAB_REVISION=$(api GET "/labs/$LAB_ID" | jq -r .laboratory.revision)
namespace=$(api POST "/labs/$LAB_ID/nodes" '{"name":"namespace","kind":"pc","cpu_count":1,"memory_mib":32,"interface_count":1}' "$LAB_REVISION")
NAMESPACE_NODE=$(jq -r .node.id <<<"$namespace")
NAMESPACE_IF=$(jq -r '.interfaces[0].id' <<<"$namespace")

set_state "$QEMU_NODE" running
set_state "$DOCKER_NODE" running
set_state "$NAMESPACE_NODE" running

link=$(api POST "/labs/$LAB_ID/links" "$(jq -nc --arg a "$QEMU_IF0" --arg b "$DOCKER_IF" '{endpoint_a_id:$a,endpoint_b_id:$b}')")
LINK_ID=$(jq -r .link.id <<<"$link")
wait_task "$(jq -r .task.id <<<"$link")" >/dev/null

LAB_REVISION=$(api GET "/labs/$LAB_ID" | jq -r .laboratory.revision)
network_object=$(api POST "/labs/$LAB_ID/network-objects" '{"name":"pc-object","kind":"pc","config":{"interfaces":[{"name":"eth0","addresses":["192.0.2.2/24"],"modes":[]}]}}' "$LAB_REVISION")
NETWORK_OBJECT_ID=$(jq -r .network_object.id <<<"$network_object")
wait_task "$(jq -r .task.id <<<"$network_object")" >/dev/null
api POST "/network-objects/$NETWORK_OBJECT_ID/attachments" "$(jq -nc --arg interface "$QEMU_IF1" '{interface_id:$interface,port_name:"eth0",config:{}}')" >/dev/null
ATTACHMENT_ID=$(sqlite3 "$DB" "select id from network_attachments where network_object_id='$NETWORK_OBJECT_ID' order by rowid desc limit 1;")
[[ -n $ATTACHMENT_ID ]]
for _ in $(seq 1 30); do
  [[ $(sqlite3 "$DB" "select observed_state from network_attachments where id='$ATTACHMENT_ID';") == active ]] && break
  sleep 1
done
[[ $(sqlite3 "$DB" "select observed_state from network_attachments where id='$ATTACHMENT_ID';") == active ]]

for switch_name in switch-a switch-b; do
  LAB_REVISION=$(api GET "/labs/$LAB_ID" | jq -r .laboratory.revision)
  switch_value=$(api POST "/labs/$LAB_ID/network-objects" "$(jq -nc --arg name "$switch_name" '{name:$name,kind:"switch_l2",config:{}}')" "$LAB_REVISION")
  wait_task "$(jq -r .task.id <<<"$switch_value")" >/dev/null
  if [[ $switch_name == switch-a ]]; then
    SWITCH_A_ID=$(jq -r .network_object.id <<<"$switch_value")
  else
    SWITCH_B_ID=$(jq -r .network_object.id <<<"$switch_value")
  fi
done
object_link=$(api POST "/labs/$LAB_ID/network-object-links" "$(jq -nc --arg a "$SWITCH_A_ID" --arg b "$SWITCH_B_ID" '{object_a_id:$a,port_a_name:"eth0",object_b_id:$b,port_b_name:"eth0"}')")
OBJECT_LINK_ID=$(jq -r .network_object_link.id <<<"$object_link")
wait_task "$(jq -r .task.id <<<"$object_link")" >/dev/null

FAILED_CONNECTION_TASK=$(python3 - <<'PY'
import secrets, time
print(f"{int(time.time() * 1000):012x}-{secrets.token_hex(10)}")
PY
)
NOW=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
sqlite3 "$DB" "insert into operation_tasks(id,kind,resource_type,resource_id,state,progress_current,progress_total,input_json,created_at) values('$FAILED_CONNECTION_TASK','topology_connection.create','topology_connection','orphan-connection','failed',1,2,'{\"laboratory_id\":\"$LAB_ID\"}','$NOW');"
sqlite3 "$DB" "insert into topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id,operation_id,state,created_at) values('$LAB_ID','node_interface','$NAMESPACE_IF','eth0','link','orphan-connection','$FAILED_CONNECTION_TASK','occupied','$NOW');"

capture=$(api POST /captures "$(jq -nc --arg laboratory "$LAB_ID" --arg source "$LINK_ID" '{laboratory_id:$laboratory,source_type:"link",source_id:$source,format:"pcap",retain:true,duration_seconds:180,max_bytes:1048576}')")
CAPTURE_ID=$(jq -r .capture.id <<<"$capture")

python3 - "$QEMU_NODE" "$EVENT_HOST" "$EVENT_PORT" <<'PY' &
import base64, os, socket, sys, time
node = sys.argv[1]
host = sys.argv[2]
port = int(sys.argv[3])
key = base64.b64encode(os.urandom(16)).decode()
sock = socket.create_connection((host, port))
request = f"GET /api/v1/nodes/{node}/consoles/telnet/stream HTTP/1.1\r\nHost: {host}:{port}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
sock.sendall(request.encode())
response = sock.recv(4096)
if b" 101 " not in response:
    raise SystemExit(response.decode(errors="replace"))
time.sleep(120)
PY
CONSOLE_PID=$!
sleep 4

BEFORE_SNAPSHOT=$(api GET "/labs/$LAB_ID")
BEFORE_SEQUENCE=$(jq -r .event_sequence <<<"$BEFORE_SNAPSHOT")
BEFORE_PLACEMENTS=$(jq -c '.placements | sort_by(.resource_id) | map({resource_id,resource_type,x,y,revision})' <<<"$BEFORE_SNAPSHOT")
BEFORE_CONNECTIONS=$(jq -c '{links:(.links|sort_by(.id)|map({id,endpoint_a_id,endpoint_b_id,desired_state,observed_state})),attachments:(.network_attachments|sort_by(.id)|map({id,network_object_id,interface_id,port_name,desired_state,observed_state})),object_links:(.network_object_links|sort_by(.id)|map({id,object_a_id,port_a_name,object_b_id,port_b_name,desired_state,observed_state}))}' <<<"$BEFORE_SNAPSHOT")
BEFORE_RESERVATIONS=$(sqlite3 -json "$DB" "select owner_type,owner_id,port_name,resource_type,resource_id,state from topology_endpoint_reservations where laboratory_id='$LAB_ID' and resource_id in ('$LINK_ID','$ATTACHMENT_ID','$OBJECT_LINK_ID') order by resource_type,resource_id,owner_type,owner_id,port_name;")
[[ $(jq 'length' <<<"$BEFORE_RESERVATIONS") == 6 ]]
EXPECTED_PLACEMENTS=$(( $(jq '.nodes | length' <<<"$BEFORE_SNAPSHOT") + $(jq '.network_objects | length' <<<"$BEFORE_SNAPSHOT") ))
[[ $(jq '.placements | length' <<<"$BEFORE_SNAPSHOT") == "$EXPECTED_PLACEMENTS" ]]
ORPHAN_PLACEMENTS=$(sqlite3 "$DB" "select count(*) from topology_placements p left join nodes n on n.id=p.resource_id and p.resource_type='node' left join network_objects o on o.id=p.resource_id and p.resource_type='network_object' where p.laboratory_id='$LAB_ID' and n.id is null and o.id is null;")
[[ $ORPHAN_PLACEMENTS == 0 ]]
QEMU_PID=$(sqlite3 "$DB" "select object_name from runtime_ownership where resource_type='node' and resource_id='$QEMU_NODE' and object_kind='qemu_process' and cleanup_state='active' order by rowid desc limit 1;")
DOCKER_SHORT_ID=$(docker ps -q --filter "label=io.netlab.node_id=$DOCKER_NODE")
DOCKER_ID=$(docker inspect --format '{{.Id}}' "$DOCKER_SHORT_ID")
NAMESPACE_NAME="nl-$(printf %s "$NAMESPACE_NODE" | sha256sum | cut -c1-12)"
[[ -n $QEMU_PID && -n $DOCKER_ID && -n $NAMESPACE_NAME ]]
netlab_mount_exec ip -n "$NAMESPACE_NAME" link show lo >/dev/null

RESUME_TASK=$(python3 - <<'PY'
import secrets, time
print(f"{int(time.time() * 1000):012x}-{secrets.token_hex(10)}")
PY
)
QEMU_REVISION=$(api GET "/nodes/$QEMU_NODE" | jq -r .revision)
NOW=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
sqlite3 "$DB" "insert into operation_tasks(id,kind,resource_type,resource_id,requested_revision,state,progress_current,progress_total,input_json,created_at) values('$RESUME_TASK','node.set_state','node','$QEMU_NODE',$QEMU_REVISION,'running',1,3,'{\"desired_state\":\"running\",\"previous_desired_state\":\"running\",\"revision\":$QEMU_REVISION}','$NOW');"

systemctl restart netlab
for _ in $(seq 1 60); do
  curl -fsS "$BASE/capabilities" >/dev/null 2>&1 && break
  sleep 1
done
wait_node "$QEMU_NODE" running
wait_node "$DOCKER_NODE" running
wait_node "$NAMESPACE_NODE" running
wait_task "$RESUME_TASK" >/dev/null
sleep 5

AFTER_SNAPSHOT=$(api GET "/labs/$LAB_ID")
AFTER_PLACEMENTS=$(jq -c '.placements | sort_by(.resource_id) | map({resource_id,resource_type,x,y,revision})' <<<"$AFTER_SNAPSHOT")
AFTER_CONNECTIONS=$(jq -c '{links:(.links|sort_by(.id)|map({id,endpoint_a_id,endpoint_b_id,desired_state,observed_state})),attachments:(.network_attachments|sort_by(.id)|map({id,network_object_id,interface_id,port_name,desired_state,observed_state})),object_links:(.network_object_links|sort_by(.id)|map({id,object_a_id,port_a_name,object_b_id,port_b_name,desired_state,observed_state}))}' <<<"$AFTER_SNAPSHOT")
[[ $AFTER_PLACEMENTS == "$BEFORE_PLACEMENTS" ]]
[[ $AFTER_CONNECTIONS == "$BEFORE_CONNECTIONS" ]]
[[ $(jq '.placements | length' <<<"$AFTER_SNAPSHOT") == "$EXPECTED_PLACEMENTS" ]]
AFTER_RESERVATIONS=$(sqlite3 -json "$DB" "select owner_type,owner_id,port_name,resource_type,resource_id,state from topology_endpoint_reservations where laboratory_id='$LAB_ID' and resource_id in ('$LINK_ID','$ATTACHMENT_ID','$OBJECT_LINK_ID') order by resource_type,resource_id,owner_type,owner_id,port_name;")
[[ $AFTER_RESERVATIONS == "$BEFORE_RESERVATIONS" ]]
[[ $(sqlite3 "$DB" "select count(*) from topology_endpoint_reservations where resource_id='orphan-connection';") == 0 ]]

QEMU_PID_AFTER=$(sqlite3 "$DB" "select object_name from runtime_ownership where resource_type='node' and resource_id='$QEMU_NODE' and object_kind='qemu_process' and cleanup_state='active' order by rowid desc limit 1;")
DOCKER_SHORT_ID_AFTER=$(docker ps -q --filter "label=io.netlab.node_id=$DOCKER_NODE")
DOCKER_ID_AFTER=$(docker inspect --format '{{.Id}}' "$DOCKER_SHORT_ID_AFTER")
NAMESPACE_NAME_AFTER="nl-$(printf %s "$NAMESPACE_NODE" | sha256sum | cut -c1-12)"
[[ $QEMU_PID == "$QEMU_PID_AFTER" && $DOCKER_ID == "$DOCKER_ID_AFTER" && $NAMESPACE_NAME == "$NAMESPACE_NAME_AFTER" ]]

LINK_NAME="nll-$(printf %s "$LINK_ID" | sha256sum | cut -c1-11)"
ATTACHMENT_NAME="nla-$(printf %s "$ATTACHMENT_ID" | sha256sum | cut -c1-11)"
ip link show "$LINK_NAME" >/dev/null
ip link show "$ATTACHMENT_NAME" >/dev/null

CAPTURE_STATE=$(api GET "/captures/$CAPTURE_ID" | jq -r '.state+":"+.completion_reason')
[[ $CAPTURE_STATE == failed:service_restart ]]
CONSOLE_STATE=$(sqlite3 "$DB" "select cleanup_state from runtime_ownership where resource_type='node' and resource_id='$QEMU_NODE' and object_kind='console_proxy' order by rowid desc limit 1;")
[[ $CONSOLE_STATE == missing_validation_required ]]

RECOVERY=$(api GET /tasks | jq -c 'map(select(.kind=="system.recovery" and .resource_id=="service_restart")) | max_by(.created_at)')
[[ $(jq -r .state <<<"$RECOVERY") == succeeded ]]
jq -e --arg id "$QEMU_NODE" --arg runtime "$QEMU_PID" '.result.resource_outcomes | any(.resource_type=="node" and .resource_id==$id and .runtime_id==$runtime)' <<<"$RECOVERY" >/dev/null
jq -e --arg id "$DOCKER_NODE" --arg runtime "$DOCKER_ID" '.result.resource_outcomes | any(.resource_type=="node" and .resource_id==$id and .runtime_id==$runtime)' <<<"$RECOVERY" >/dev/null
jq -e --arg id "$NAMESPACE_NODE" --arg runtime "$NAMESPACE_NAME" '.result.resource_outcomes | any(.resource_type=="node" and .resource_id==$id and .runtime_id==$runtime)' <<<"$RECOVERY" >/dev/null
jq -e '.result.resource_outcomes | any(.resource_type=="link" and .resource_id=="orphan-connection" and .details.action=="orphan_reservation_removed")' <<<"$RECOVERY" >/dev/null

python3 - "$BEFORE_SEQUENCE" "$EVENT_HOST" "$EVENT_PORT" <<'PY'
import base64, json, os, socket, struct, sys
sequence = int(sys.argv[1])
host = sys.argv[2]
port = int(sys.argv[3])
key = base64.b64encode(os.urandom(16)).decode()
sock = socket.create_connection((host, port))
request = f"GET /api/v1/events?after={sequence} HTTP/1.1\r\nHost: {host}:{port}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
sock.sendall(request.encode())
response = b""
while b"\r\n\r\n" not in response:
    response += sock.recv(4096)
if b" 101 " not in response.split(b"\r\n", 1)[0]:
    raise SystemExit(response.decode(errors="replace"))
header = sock.recv(2)
length = header[1] & 127
if length == 126:
    length = struct.unpack("!H", sock.recv(2))[0]
elif length == 127:
    length = struct.unpack("!Q", sock.recv(8))[0]
payload = b""
while len(payload) < length:
    payload += sock.recv(length - len(payload))
event = json.loads(payload)
assert event["sequence"] > sequence, event
PY

QEMU_DIRECTORY="$STATE/runtime/qemu/$QEMU_NODE"
DELETED_LAB_ID="$LAB_ID"
cleanup
LAB_ID=""
[[ ! -e $QEMU_DIRECTORY ]]
[[ -z $(docker ps -aq --filter "id=$DOCKER_ID") ]]
! netlab_mount_exec ip -n "$NAMESPACE_NAME" link show lo >/dev/null 2>&1
! ip link show "$LINK_NAME" >/dev/null 2>&1
! ip link show "$ATTACHMENT_NAME" >/dev/null 2>&1
OWNERSHIP_COUNT=$(sqlite3 "$DB" "select count(*) from runtime_ownership where resource_id in ('$QEMU_NODE','$DOCKER_NODE','$NAMESPACE_NODE','$LINK_ID','$ATTACHMENT_ID','$OBJECT_LINK_ID','$NETWORK_OBJECT_ID','$SWITCH_A_ID','$SWITCH_B_ID','$CAPTURE_ID');")
[[ $OWNERSHIP_COUNT == 0 ]]
[[ $(sqlite3 "$DB" "select count(*) from topology_placements where laboratory_id='$DELETED_LAB_ID';") == 0 ]]
[[ $(sqlite3 "$DB" "select count(*) from links where laboratory_id='$DELETED_LAB_ID';") == 0 ]]
[[ $(sqlite3 "$DB" "select count(*) from network_object_links where laboratory_id='$DELETED_LAB_ID';") == 0 ]]
[[ $(sqlite3 "$DB" "select count(*) from network_attachments a join network_objects o on o.id=a.network_object_id where o.laboratory_id='$DELETED_LAB_ID';") == 0 ]]
[[ $(sqlite3 "$DB" "select count(*) from topology_endpoint_reservations where laboratory_id='$DELETED_LAB_ID';") == 0 ]]
[[ $(sqlite3 "$DB" "select count(*) from operation_tasks where id='$FAILED_CONNECTION_TASK' or resource_id in ('$LINK_ID','$ATTACHMENT_ID','$OBJECT_LINK_ID');") == 0 ]]

echo "PASS T225 lab=$name placements=$EXPECTED_PLACEMENTS qemu_pid=$QEMU_PID docker_id=$DOCKER_ID namespace=$NAMESPACE_NAME task=$RESUME_TASK capture=$CAPTURE_STATE console=$CONSOLE_STATE"
