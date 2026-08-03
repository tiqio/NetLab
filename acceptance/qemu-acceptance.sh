#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${NETLAB_BASE_URL:-http://127.0.0.1:8088}
LAB_NAME=${NETLAB_ACCEPTANCE_LAB:-qemu-acceptance-$(date -u +%Y%m%dT%H%M%SZ)}
ALLOW_REBOOT=${NETLAB_ACCEPTANCE_ALLOW_REBOOT:-0}
POLL_SECONDS=${NETLAB_ACCEPTANCE_POLL_SECONDS:-180}
LAB_ID=""

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

cleanup() {
  [[ -n $LAB_ID ]] || return 0
  local lab revision
  lab=$(api GET "/labs/$LAB_ID" 2>/dev/null) || return 0
  revision=$(jq -r '.laboratory.revision' <<<"$lab")
  api DELETE "/labs/$LAB_ID" '' "$revision" >/dev/null || true
}
trap cleanup EXIT

require_tools() {
  for tool in curl jq python3; do command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 2; }; done
}

free_tcp_port() {
  python3 - <<'PY'
import socket

with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
    listener.bind(("127.0.0.1", 0))
    print(listener.getsockname()[1])
PY
}

wait_task() {
  local id=$1 deadline=$((SECONDS + POLL_SECONDS)) task state
  while (( SECONDS < deadline )); do
    task=$(api GET "/tasks/$id")
    state=$(jq -r '.state' <<<"$task")
    case $state in succeeded) printf '%s\n' "$task"; return 0;; failed|cancelled) printf '%s\n' "$task" >&2; return 1;; esac
    sleep 1
  done
  echo "task $id timed out" >&2
  return 1
}

wait_node() {
  local id=$1 expected=$2 deadline=$((SECONDS + POLL_SECONDS)) node
  while (( SECONDS < deadline )); do
    node=$(api GET "/nodes/$id")
    [[ $(jq -r '.observed_state' <<<"$node") == "$expected" ]] && { printf '%s\n' "$node"; return 0; }
    [[ $(jq -r '.observed_state' <<<"$node") == failed ]] && { printf '%s\n' "$node" >&2; return 1; }
    sleep 1
  done
  echo "node $id did not reach $expected" >&2
  return 1
}

wait_capability() {
  local id=$1 capability=$2 deadline=$((SECONDS + POLL_SECONDS)) observations state
  while (( SECONDS < deadline )); do
    observations=$(api GET "/nodes/$id/capabilities")
    state=$(jq -r --arg capability "$capability" 'first(.observations[] | select(.capability==$capability)) | .state // "unknown"' <<<"$observations")
    case $state in
      ready) printf '%s\n' "$observations"; return 0 ;;
      unavailable|failed) printf '%s\n' "$observations" >&2; return 1 ;;
    esac
    sleep 1
  done
  echo "capability $capability did not become ready for node $id" >&2
  return 1
}

registered_qemu_versions() {
  local images templates
  images=$(api GET /images)
  templates=$(api GET /templates)
  jq -r --argjson images "$images" '
    [ $images[] | select(.runtime_kind == "qemu" and .availability == "available") | .id ] as $legal |
    .[] | .versions[] | select(.enabled == true and (.image_version_id as $id | $legal | index($id))) | .id
  ' <<<"$templates"
}

registered_ubuntu_version() {
  local images templates
  images=$(api GET /images)
  templates=$(api GET /templates)
  jq -r --argjson images "$images" '
    [ $images[] | select(.runtime_kind == "qemu" and .availability == "available") | .id ] as $legal |
    first(.[] | select(.template_key == "ubuntu-qemu") | .versions[] | select(.enabled == true and (.image_version_id as $id | $legal | index($id)))) | .id // empty
  ' <<<"$templates"
}

create_node() {
  local version=$1 index=$2 response
  response=$(api POST "/labs/$LAB_ID/nodes" "$(jq -nc --arg name "qemu-$index" --arg version "$version" '{name:$name,template_version_id:$version,cpu_count:2,cpu_quota_micros:100000,memory_mib:1024,storage_gib:8,interface_limit:8,interface_count:2,process_limit:256}')")
  jq -r '.node.id' <<<"$response"
}

start_node() {
  local id=$1 node revision
  node=$(api GET "/nodes/$id"); revision=$(jq -r '.revision' <<<"$node")
  api PUT "/nodes/$id/state" '{"desired_state":"running"}' "$revision" >/dev/null
  wait_node "$id" running >/dev/null
}

verify_node() {
  local id=$1 consoles task task_result interface response mapping capture host_port
  consoles=$(api GET "/nodes/$id/consoles")
  jq -e 'map(.mode) | index("telnet") and index("vnc")' <<<"$consoles" >/dev/null

  wait_capability "$id" qmp >/dev/null
  wait_capability "$id" qga >/dev/null
  wait_capability "$id" guest_exec >/dev/null
  response=$(api POST "/nodes/$id/guest-exec" '{"argv":["/bin/sh","-lc","printf netlab-qga"],"timeout_seconds":30,"output_limit":4096}')
  task=$(jq -r '.task.id // .id' <<<"$response")
  task_result=$(wait_task "$task")
  jq -e '.result.exit_code == 0 and (.result.stdout_base64 | @base64d) == "netlab-qga"' <<<"$task_result" >/dev/null

  response=$(api POST "/nodes/$id/interfaces" '{"driver":"virtio-net-pci"}')
  interface=$(jq -r '.interface.id // .id' <<<"$response")
  task=$(jq -r '.task.id // empty' <<<"$response"); [[ -z $task ]] || wait_task "$task" >/dev/null
  local iface_revision; iface_revision=$(jq -r '.interface.revision // .revision' <<<"$response")
  response=$(api DELETE "/interfaces/$interface" '' "$iface_revision")
  task=$(jq -r '.task.id // .id // empty' <<<"$response"); [[ -z $task ]] || wait_task "$task" >/dev/null

  host_port=$(free_tcp_port)
  response=$(api POST "/nodes/$id/port-mappings" "$(jq -nc --argjson port "$host_port" '{protocol:"tcp",host_address:"127.0.0.1",host_port:$port,guest_address:"127.0.0.1",guest_port:22}')")
  mapping=$(jq -r '.port_mapping.id' <<<"$response"); task=$(jq -r '.task.id' <<<"$response"); wait_task "$task" >/dev/null
  api DELETE "/port-mappings/$mapping" >/dev/null

  api GET "/nodes/$id/resources" | jq -e '.configured.cpu_count == 2 and .configured.cpu_quota_micros == 100000 and .observed.runtime_kind == "qemu"' >/dev/null

  local first_interface; first_interface=$(api GET "/labs/$LAB_ID" | jq -r --arg id "$id" '.interfaces[] | select(.node_id == $id) | .id' | head -n1)
  response=$(api POST /captures "$(jq -nc --arg lab "$LAB_ID" --arg source "$first_interface" '{laboratory_id:$lab,source_type:"interface",source_id:$source,format:"pcap",retain:true,duration_seconds:5,max_bytes:1048576}')")
  capture=$(jq -r '.capture.id' <<<"$response"); sleep 1
  curl -fsS --max-time 3 "$BASE_URL/api/v1/captures/$capture/stream" -o /dev/null || true
  api DELETE "/captures/$capture" >/dev/null
}

verify_automatic_networking() {
  local id=$1 snapshot interface response object task diagnostics task_result output deadline
  snapshot=$(api GET "/labs/$LAB_ID")
  interface=$(jq -r --arg id "$id" 'first(.interfaces[] | select(.node_id == $id)) | .id // empty' <<<"$snapshot")
  [[ -n $interface ]] || { echo "Ubuntu node $id has no interface for NAT acceptance" >&2; return 1; }

  response=$(api POST "/labs/$LAB_ID/network-objects" '{"name":"automatic-networking","kind":"nat_bridge","config":{"ipv4_prefix":"10.244.86.0/24","ipv6_prefix":"fd86::/64","uplink":"eth0","dhcpv4":{"start":"10.244.86.20","end":"10.244.86.80","lease_time":"10m"},"dhcpv6":{"start":"fd86::20","end":"fd86::80","lease_time":"10m"},"router_advertisements":true,"dns_servers":["1.1.1.1","2606:4700:4700::1111"],"domain":"acceptance.netlab"}}')
  object=$(jq -r '.network_object.id' <<<"$response")
  task=$(jq -r '.task.id' <<<"$response")
  wait_task "$task" >/dev/null
  api POST "/network-objects/$object/attachments" "$(jq -nc --arg interface "$interface" '{interface_id:$interface,port_name:"lan0",config:{}}')" >/dev/null

  diagnostics=$(api GET "/network-objects/$object/diagnostics")
  jq -e '.allocation_status == "gateway_assigned" and .translation_status.owned_rule == true and .helper.state == "active"' <<<"$diagnostics" >/dev/null

  deadline=$((SECONDS + POLL_SECONDS))
  while (( SECONDS < deadline )); do
    response=$(api POST "/nodes/$id/guest-exec" '{"argv":["/bin/sh","-lc","ip -o address show scope global"],"timeout_seconds":30,"output_limit":16384}')
    task=$(jq -r '.task.id' <<<"$response")
    if task_result=$(wait_task "$task" 2>/dev/null); then
      output=$(jq -r '.result.stdout_base64 | @base64d' <<<"$task_result")
      if grep -Eq 'inet 10\.244\.86\.[0-9]+/' <<<"$output" && grep -Eq 'inet6 (fd86:|[23][0-9a-f]{3}:)' <<<"$output"; then
        return 0
      fi
    fi
    sleep 2
  done
  echo "Ubuntu node $id did not obtain automatic IPv4 and IPv6 addresses" >&2
  return 1
}

require_tools
api GET /capabilities >/dev/null
mapfile -t versions < <(registered_qemu_versions)
ubuntu_version=$(registered_ubuntu_version)
if (( ${#versions[@]} < 1 )); then
  echo "SKIP: one operator-registered available QEMU template version is required; found none." >&2
  exit 77
fi
if [[ -z $ubuntu_version ]]; then
  echo "SKIP: an available Ubuntu QEMU template version is required for automatic-networking acceptance." >&2
  exit 77
fi

lab=$(api POST /labs "$(jq -nc --arg name "$LAB_NAME" '{name:$name,recovery_policy:"auto_restore"}')")
LAB_ID=$(jq -r '.id' <<<"$lab")
nodes=("$(create_node "$ubuntu_version" 1)")
for index in 1 2 3; do nodes+=("$(create_node "${versions[$((index % ${#versions[@]}))]}" "$((index + 1))")"); done
for id in "${nodes[@]}"; do start_node "$id"; done
for id in "${nodes[@]}"; do verify_node "$id"; done
verify_automatic_networking "${nodes[0]}"

if [[ $ALLOW_REBOOT == 1 ]]; then
  marker=${NETLAB_ACCEPTANCE_REBOOT_MARKER:-/var/lib/netlab/acceptance-reboot.json}
  jq -n --arg lab "$LAB_ID" --argjson nodes "$(printf '%s\n' "${nodes[@]}" | jq -R . | jq -s .)" '{laboratory_id:$lab,nodes:$nodes,created_at:now|todate}' >"$marker"
  trap - EXIT
  echo "PREPARED_REBOOT: $marker"
  exit 75
fi

echo "PASS: four-QEMU acceptance completed for laboratory $LAB_ID"
