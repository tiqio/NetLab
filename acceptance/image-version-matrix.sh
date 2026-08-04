#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${NETLAB_BASE_URL:-http://127.0.0.1:18082}
POLL_SECONDS=${NETLAB_ACCEPTANCE_POLL_SECONDS:-900}
KEEP_LAB=${NETLAB_KEEP_LAB:-0}
LAB_ID=""
RESULTS=""

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

wait_task() {
  local id=$1 deadline=$((SECONDS + POLL_SECONDS)) value state
  while ((SECONDS < deadline)); do
    value=$(api GET "/tasks/$id")
    state=$(jq -r .state <<<"$value")
    case $state in
      succeeded) printf '%s\n' "$value"; return 0 ;;
      failed|cancelled) printf '%s\n' "$value" >&2; return 1 ;;
    esac
    sleep 1
  done
  echo "task $id timed out" >&2
  return 1
}

wait_node() {
  local id=$1 expected=$2 deadline=$((SECONDS + POLL_SECONDS)) value state
  while ((SECONDS < deadline)); do
    value=$(api GET "/nodes/$id")
    state=$(jq -r .observed_state <<<"$value")
    [[ $state == "$expected" ]] && return 0
    [[ $state == failed ]] && { printf '%s\n' "$value" >&2; return 1; }
    sleep 1
  done
  echo "node $id did not reach $expected" >&2
  return 1
}

wait_capability() {
  local id=$1 capability=$2 deadline=$((SECONDS + POLL_SECONDS)) value state
  while ((SECONDS < deadline)); do
    value=$(api GET "/nodes/$id/capabilities")
    state=$(jq -r --arg capability "$capability" 'first(.observations[] | select(.capability==$capability)) | .state // "unknown"' <<<"$value")
    [[ $state == ready ]] && return 0
    sleep 2
  done
  api GET "/nodes/$id/capabilities" >&2
  return 1
}

set_state() {
  local id=$1 desired=$2 value revision response task
  value=$(api GET "/nodes/$id")
  revision=$(jq -r .revision <<<"$value")
  response=$(api PUT "/nodes/$id/state" "$(jq -nc --arg desired "$desired" '{desired_state:$desired}')" "$revision")
  task=$(jq -r '.task.id // empty' <<<"$response")
  [[ -z $task ]] || wait_task "$task" >/dev/null
  wait_node "$id" "$desired"
}

guest_exec() {
  local id=$1 command=$2 response
  response=$(api POST "/nodes/$id/guest-exec" "$(jq -nc --arg command "$command" '{argv:["/bin/bash","-lc",$command],timeout_seconds:600,output_limit:65536}')")
  wait_task "$(jq -r .task.id <<<"$response")"
}

cleanup() {
  if [[ -n $LAB_ID && $KEEP_LAB != 1 ]]; then
    local value revision
    value=$(api GET "/labs/$LAB_ID" 2>/dev/null) || true
    if [[ -n ${value:-} ]]; then
      revision=$(jq -r .laboratory.revision <<<"$value")
      api DELETE "/labs/$LAB_ID" '' "$revision" >/dev/null || true
    fi
  fi
  [[ -z $RESULTS ]] || rm -f "$RESULTS"
}
trap cleanup EXIT

for command_name in curl jq docker python3; do
  command -v "$command_name" >/dev/null || { echo "$command_name is required" >&2; exit 2; }
done

RESULTS=$(mktemp)
catalog=$(api GET /templates)
LAB_ID=$(api POST /labs "$(jq -nc --arg name "image-version-matrix-$(date -u +%Y%m%dT%H%M%SZ)" '{name:$name,recovery_policy:"auto_restore"}')" | jq -r .id)
response=$(api POST "/labs/$LAB_ID/network-objects" '{"name":"Image Matrix NAT","kind":"nat_bridge","config":{"ipv4_prefix":"10.244.93.0/24","uplink":"auto","dhcpv4":{"start":"10.244.93.20","end":"10.244.93.120","lease_time":"30m"},"dns_servers":["1.1.1.1"],"domain":"images.netlab"}}')
nat_id=$(jq -r .network_object.id <<<"$response")
wait_task "$(jq -r .task.id <<<"$response")" >/dev/null

create_node() {
  local template_key=$1 version=$2 name=$3 bootstrap=${4:-} row template image response node interface
  row=$(jq -r --arg key "$template_key" --arg version "$version" '.[] | select(.template_key==$key) | .versions[] | select(.version==$version and .enabled==true) | [.id,.image_version_id] | @tsv' <<<"$catalog")
  [[ -n $row ]] || { echo "missing enabled template $template_key $version" >&2; return 1; }
  IFS=$'\t' read -r template image <<<"$row"
  [[ -n $image && $image != null ]] || { echo "missing recommended image for $template_key $version" >&2; return 1; }
  response=$(api POST "/labs/$LAB_ID/nodes" "$(jq -nc --arg name "$name" --arg template "$template" --arg image "$image" --arg bootstrap "$bootstrap" '{name:$name,template_version_id:$template,image_version_id:$image,cpu_count:2,cpu_quota_micros:100000,memory_mib:2048,storage_gib:12,interface_count:1,interface_limit:8,process_limit:512} + (if $bootstrap=="" then {} else {bootstrap:{user_data:$bootstrap,meta_data:"instance-id: image-matrix\nlocal-hostname: image-matrix\n"}} end)')")
  node=$(jq -r .node.id <<<"$response")
  interface=$(jq -r '.interfaces[0].id' <<<"$response")
  api POST "/network-objects/$nat_id/attachments" "$(jq -nc --arg interface "$interface" --arg port "$name" '{interface_id:$interface,port_name:$port,config:{}}')" >/dev/null
  printf '%s\t%s\n' "$node" "$interface"
}

for family in ubuntu-container busybox-container; do
  if [[ $family == ubuntu-container ]]; then
    versions=(22.04 24.04 26.04)
  else
    versions=(1.36.1 1.37.0 1.38.0)
  fi
  for version in "${versions[@]}"; do
    IFS=$'\t' read -r node interface < <(create_node "$family" "$version" "$family-$version")
    set_state "$node" running
    container=$(docker ps -q --filter "label=io.netlab.node_id=$node")
    [[ -n $container ]]
    if [[ $family == ubuntu-container ]]; then
      actual=$(docker exec "$container" sh -lc '. /etc/os-release; printf %s "$VERSION_ID"')
      [[ $actual == "$version" ]]
    else
      actual=$(docker exec "$container" busybox | head -1)
      [[ $actual == "BusyBox v$version"* ]]
    fi
    docker exec "$container" test -e /sys/class/net/eth0
    api GET "/nodes/$node/consoles" | jq -e 'map(.mode) | index("telnet")' >/dev/null
    set_state "$node" stopped
    jq -nc --arg kind docker --arg family "$family" --arg version "$version" --arg node "$node" --arg actual "$actual" '{runtime_kind:$kind,template_key:$family,version:$version,node_id:$node,actual_version:$actual,checks:["start","image_version","eth0","telnet","stop"],status:"passed"}' >>"$RESULTS"
  done
done

for version in 22.04 24.04 26.04; do
  bootstrap=$(cat <<EOF
#cloud-config
hostname: netlab-ubuntu-${version//./-}
manage_etc_hosts: true
package_update: true
packages: [qemu-guest-agent]
runcmd:
  - [systemctl, enable, --now, qemu-guest-agent]
  - [sh, -c, "echo IMAGE_MATRIX_$version >/var/tmp/netlab-image-matrix"]
EOF
)
  IFS=$'\t' read -r node interface < <(create_node ubuntu-qemu "$version" "Ubuntu-QEMU-$version" "$bootstrap")
  set_state "$node" running
  wait_capability "$node" qmp
  wait_capability "$node" qga
  consoles=$(api GET "/nodes/$node/consoles")
  jq -e 'map(.mode) as $m | ($m|index("telnet")) and ($m|index("vnc"))' <<<"$consoles" >/dev/null
  rfb=$(python3 - "$node" <<'PY'
import socket
import sys

path = f"/var/lib/netlab/runtime/qemu/{sys.argv[1]}/vnc.sock"
client = socket.socket(socket.AF_UNIX)
client.settimeout(5)
client.connect(path)
print(client.recv(12).decode("ascii", "replace").strip())
client.close()
PY
)
  [[ $rfb == RFB* ]]
  result=$(guest_exec "$node" 'cloud-init status --wait --long >/dev/null; . /etc/os-release; printf "%s|%s|%s" "$VERSION_ID" "$(cat /var/tmp/netlab-image-matrix)" "$(ip -o -4 addr show scope global | awk '\''{print $4}'\'' | head -1)"')
  output=$(jq -r '.result.stdout_base64 | @base64d' <<<"$result")
  [[ $output == "$version|IMAGE_MATRIX_$version|10.244.93."* ]]
  snapshot=$(api GET "/labs/$LAB_ID")
  jq -e --arg interface "$interface" '.interfaces[] | select(.id==$interface and .operational_state=="up")' <<<"$snapshot" >/dev/null
  set_state "$node" stopped
  jq -nc --arg kind qemu --arg family ubuntu-qemu --arg version "$version" --arg node "$node" --arg output "$output" --arg rfb "$rfb" '{runtime_kind:$kind,template_key:$family,version:$version,node_id:$node,guest:$output,vnc_protocol:$rfb,checks:["start","cloud_init","dhcp","qmp","qga","guest_exec","telnet","vnc","interface_state","stop"],status:"passed"}' >>"$RESULTS"
done

for specification in \
  'fancywan|v1-stable' \
  'fancywan|2025.02.20-165226' \
  'fortigate|7.2.0-build1157' \
  'ruijie-router|V1.06' \
  'ruijie-switch|V1.06'; do
  IFS='|' read -r template_key version <<<"$specification"
  row=$(jq -r --arg key "$template_key" --arg version "$version" '.[] | select(.template_key==$key) | .versions[] | select(.version==$version and .enabled==true and (.image_version_id // "")!="") | [.id,.image_version_id] | @tsv' <<<"$catalog")
  if [[ -z $row ]]; then
    jq -nc --arg kind qemu --arg family "$template_key" --arg version "$version" '{runtime_kind:$kind,template_key:$family,version:$version,checks:[],status:"skipped",reason:"no recommended reviewed image is registered"}' >>"$RESULTS"
    continue
  fi
  IFS=$'\t' read -r node interface < <(create_node "$template_key" "$version" "$template_key-$version")
  set_state "$node" running
  wait_capability "$node" qmp
  consoles=$(api GET "/nodes/$node/consoles")
  jq -e 'length > 0 and (map(.mode) | index("telnet"))' <<<"$consoles" >/dev/null
  checks='["start","qmp","telnet","stop"]'
  rfb=""
  if jq -e 'map(.mode) | index("vnc")' <<<"$consoles" >/dev/null; then
    rfb=$(python3 - "$node" <<'PY'
import socket
import sys

path = f"/var/lib/netlab/runtime/qemu/{sys.argv[1]}/vnc.sock"
client = socket.socket(socket.AF_UNIX)
client.settimeout(5)
client.connect(path)
print(client.recv(12).decode("ascii", "replace").strip())
client.close()
PY
)
    [[ $rfb == RFB* ]]
    checks='["start","qmp","telnet","vnc","stop"]'
  fi
  sleep 10
  [[ $(api GET "/nodes/$node" | jq -r .observed_state) == running ]]
  set_state "$node" stopped
  jq -nc --arg kind qemu --arg family "$template_key" --arg version "$version" --arg node "$node" --arg rfb "$rfb" --argjson checks "$checks" '{runtime_kind:$kind,template_key:$family,version:$version,node_id:$node,vnc_protocol:(if $rfb=="" then null else $rfb end),checks:$checks,status:"passed"}' >>"$RESULTS"
done

jq -s --arg lab "$LAB_ID" '{schema_version:"1.0",laboratory_id:$lab,generated_at:(now|todate),results:.}' "$RESULTS"
