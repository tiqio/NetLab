#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${NETLAB_BASE_URL:-http://127.0.0.1:8088}
BASE="$BASE_URL/api/v1"
POLL_SECONDS=${NETLAB_ACCEPTANCE_POLL_SECONDS:-300}
LAB_ID=""

api() {
  local method=$1 path=$2 body=${3:-} revision=${4:-}
  local args=(-fsS -X "$method" -H 'Accept: application/json')
  [[ $method == GET ]] || args+=(-H "Idempotency-Key: $(cat /proc/sys/kernel/random/uuid)")
  [[ -z $revision ]] || args+=(-H "If-Match: $revision")
  [[ -z $body ]] || args+=(-H 'Content-Type: application/json' --data "$body")
  curl "${args[@]}" "$BASE$path"
}

wait_task() {
  local id=$1 deadline=$((SECONDS + POLL_SECONDS)) value state
  while ((SECONDS < deadline)); do
    value=$(api GET "/tasks/$id")
    state=$(jq -r .state <<<"$value")
    case $state in
      succeeded) return 0 ;;
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

set_state() {
  local id=$1 desired=$2 value revision task expected
  value=$(api GET "/nodes/$id")
  revision=$(jq -r .revision <<<"$value")
  task=$(api PUT "/nodes/$id/state" "{\"desired_state\":\"$desired\"}" "$revision" | jq -r .task.id)
  wait_task "$task"
  expected=stopped; [[ $desired == running ]] && expected=running
  wait_node "$id" "$expected"
}

cleanup() {
  [[ ${NETLAB_KEEP_FAILED:-0} != 1 ]] || return 0
  [[ -n $LAB_ID ]] || return 0
  local snapshot revision task
  snapshot=$(api GET "/labs/$LAB_ID" 2>/dev/null) || return 0
  revision=$(jq -r .laboratory.revision <<<"$snapshot")
  task=$(api DELETE "/labs/$LAB_ID" '' "$revision" 2>/dev/null | jq -r '.task.id // empty') || true
  [[ -z $task ]] || wait_task "$task" || true
}
trap cleanup EXIT

for tool in curl jq; do command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 2; }; done

templates=$(api GET /templates)
images=$(api GET /images)

template_version() {
  jq -r --arg key "$1" 'first(.[] | select(.template_key==$key) | .versions[] | select(.enabled)) | .id // empty' <<<"$templates"
}

image_id() {
  local explicit=$1 pattern=$2
  if [[ -n $explicit ]]; then
    jq -er --arg id "$explicit" 'first(.[] | select(.id==$id and .runtime_kind=="qemu" and .availability=="available" and .license_status=="reviewed")) | .id' <<<"$images"
    return
  fi
  jq -er --arg pattern "$pattern" 'first(.[] | select(.runtime_kind=="qemu" and .availability=="available" and .license_status=="reviewed" and ((.name+" "+.version)|ascii_downcase|test($pattern)))) | .id' <<<"$images"
}

ubuntu_version=$(template_version ubuntu-qemu)
vyos_version=$(template_version vyos)
fancywan_version=$(template_version fancywan)
ubuntu_image=$(image_id "${NETLAB_UBUNTU_IMAGE_ID:-}" 'ubuntu') || true
vyos_image=$(image_id "${NETLAB_VYOS_IMAGE_ID:-}" 'vyos') || true
fancywan_image=$(image_id "${NETLAB_FANCYWAN_IMAGE_ID:-}" 'fancywan|bulbel') || true

if [[ -z $ubuntu_version || -z $vyos_version || -z $fancywan_version || -z $ubuntu_image || -z $vyos_image || -z $fancywan_image ]]; then
  echo "SKIP: genuine Ubuntu, VyOS, and FancyWAN template versions and reviewed operator media are all required." >&2
  exit 77
fi

if [[ -n ${NETLAB_FORTIGATE_IMAGE_ID:-} && ${NETLAB_FORTIGATE_LICENSE_REVIEWED:-0} != 1 ]]; then
  echo "FortiGate media was supplied without explicit license review attestation." >&2
  exit 6
fi

LAB_ID=$(api POST /labs "$(jq -nc --arg name "genuine-images-$(date -u +%Y%m%dT%H%M%SZ)" '{name:$name,recovery_policy:"auto_restore"}')" | jq -r .id)

default_ubuntu='#cloud-config
write_files:
  - path: /var/tmp/netlab-template-family
    content: ubuntu-qemu
'

read_user_data() {
  local family=$1 file=$2
  if [[ -n $file ]]; then cat "$file"; return; fi
  if [[ $family == ubuntu-qemu ]]; then printf '%s' "$default_ubuntu"; return; fi
  echo "missing operator bootstrap file for $family" >&2
  return 1
}

declare -a results=()
for spec in \
  "ubuntu-qemu|$ubuntu_version|$ubuntu_image|${NETLAB_UBUNTU_USER_DATA_FILE:-}" \
  "vyos|$vyos_version|$vyos_image|${NETLAB_VYOS_USER_DATA_FILE:-}" \
  "fancywan|$fancywan_version|$fancywan_image|${NETLAB_FANCYWAN_USER_DATA_FILE:-}"; do
  IFS='|' read -r family version image bootstrap_file <<<"$spec"
  user_data=$(read_user_data "$family" "$bootstrap_file") || exit 77
  response=$(api POST "/labs/$LAB_ID/nodes" "$(jq -nc --arg name "$family" --arg version "$version" --arg image "$image" --arg user_data "$user_data" '{name:$name,template_version_id:$version,image_version_id:$image,bootstrap:{user_data:$user_data,meta_data:"instance-id: netlab\n"}}')")
  node=$(jq -r .node.id <<<"$response")
  set_state "$node" running
  capabilities=$(api GET "/nodes/$node/capabilities")
  jq -e '.observations[] | select(.capability=="qmp" and .state=="ready")' <<<"$capabilities" >/dev/null
  api GET "/nodes/$node/consoles" | jq -e 'length > 0' >/dev/null
  set_state "$node" stopped
  digest=$(jq -r --arg id "$image" '.[] | select(.id==$id) | .digest' <<<"$images")
  results+=("$(jq -nc --arg family "$family" --arg node "$node" --arg digest "$digest" '{template_key:$family,node_id:$node,image_digest:$digest,genuine_workload:true,status:"passed"}')")
done

printf '%s\n' "${results[@]}" | jq -s '{schema_version:"1.0",generated_at:(now|todate),templates:.}'
