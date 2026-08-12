#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${NETLAB_BASE_URL:-http://127.0.0.1:18082}
MARKER=${NETLAB_NETWORK_PATH_RESTART_MARKER:-/var/lib/netlab/network-path-host-restart.json}
ALLOW_REBOOT=${NETLAB_ACCEPTANCE_ALLOW_REBOOT:-0}
LAB_ID=${NETLAB_NETWORK_PATH_LAB_ID:-019feee704ee-c5b2bd57159bebe86bed}
UBUNTU_NODE_ID=${NETLAB_NETWORK_PATH_UBUNTU_NODE_ID:-019feee72dd8-8ef0f258c957f37d64b4}
PHASE=${1:-prepare}

require() { command -v "$1" >/dev/null || { echo "$1 is required" >&2; exit 2; }; }
api() { curl -fsS "$BASE_URL/api/v1$1"; }
require curl
require jq
require base64

probe() {
  local family=$1 destination=$2 body response task state value stdout
  if [[ $family == ipv6 ]]; then
    body=$(jq -nc --arg destination "$destination" '{argv:["ping","-6","-c","5","-W","2",$destination],timeout_seconds:20,output_limit:8192}')
  else
    body=$(jq -nc --arg destination "$destination" '{argv:["ping","-c","5","-W","2",$destination],timeout_seconds:20,output_limit:8192}')
  fi
  response=$(curl -fsS -X POST "$BASE_URL/api/v1/nodes/$UBUNTU_NODE_ID/guest-exec" -H 'Content-Type: application/json' --data "$body")
  task=$(jq -r '.task.id' <<<"$response")
  for _ in $(seq 1 100); do
    value=$(api "/tasks/$task")
    state=$(jq -r '.state' <<<"$value")
    case "$state" in
      succeeded)
        [[ $(jq -r '.result.exit_code' <<<"$value") == 0 ]] || return 1
        stdout=$(jq -r '.result.stdout_base64' <<<"$value" | base64 -d)
        grep -Eq '5 (packets transmitted|transmitted), 5 (packets received|received)' <<<"$stdout"
        jq -nc --arg family "$family" --arg destination "$destination" --arg task_id "$task" '{family:$family,destination:$destination,task_id:$task_id,result:"passed"}'
        return
        ;;
      failed|cancelled)
        jq -c . <<<"$value" >&2
        return 1
        ;;
    esac
    sleep .2
  done
  echo "guest-exec task timed out: $task" >&2
  return 1
}

path_snapshot() {
  local results=() family destination
  for destination in 172.16.0.1 10.20.20.1 10.20.20.10 10.30.30.1 10.30.30.10; do
    results+=("$(probe ipv4 "$destination")")
  done
  for destination in fd16::1 fd20::1 fd20::10 fd30::1 fd30::10; do
    results+=("$(probe ipv6 "$destination")")
  done
  printf '%s\n' "${results[@]}" | jq -s .
}

case "$PHASE" in
  prepare)
    [[ $ALLOW_REBOOT == 1 ]] || { echo "SKIP: set NETLAB_ACCEPTANCE_ALLOW_REBOOT=1 for operator-controlled host restart" >&2; exit 77; }
    candidate=$(api /capabilities | jq -r '.release.candidate_id // empty')
    [[ -n $candidate ]] || { echo "candidate identity unavailable" >&2; exit 1; }
    topology=$(api "/labs/$LAB_ID" | jq '{revision:.laboratory.revision,nodes:[.nodes[]|{id,desired_state,observed_state}],links:[.links[]|{id,observed_state}],attachments:[.network_attachments[]|{id,observed_state}],object_links:[.network_object_links[]|{id,observed_state}]}')
    paths=$(path_snapshot)
    install -d -m0700 "$(dirname "$MARKER")"
    jq -n --arg candidate "$candidate" --arg lab_id "$LAB_ID" --arg ubuntu_node_id "$UBUNTU_NODE_ID" --argjson topology "$topology" --argjson paths "$paths" '{schema_version:"1.0",candidate_id:$candidate,laboratory_id:$lab_id,ubuntu_node_id:$ubuntu_node_id,prepared_at:(now|todate),before:{topology:$topology,paths:$paths},phase:"prepared"}' >"$MARKER"
    chmod 0600 "$MARKER"
    echo "PREPARED_REBOOT: reboot the host, then execute $0 verify"
    ;;
  verify)
    [[ -f $MARKER ]] || { echo "restart marker not found: $MARKER" >&2; exit 2; }
    candidate=$(api /capabilities | jq -r '.release.candidate_id // empty')
    expected=$(jq -r .candidate_id "$MARKER")
    [[ $candidate == "$expected" ]] || { echo "candidate changed across restart" >&2; exit 1; }
    recovery=$(api '/tasks?limit=200' | jq -c 'map(select(.kind=="system.recovery")) | max_by(.created_at)')
    [[ $(jq -r '.state // empty' <<<"$recovery") == succeeded ]] || { echo "$recovery" >&2; exit 1; }
    topology=$(api "/labs/$LAB_ID" | jq '{revision:.laboratory.revision,nodes:[.nodes[]|{id,desired_state,observed_state}],links:[.links[]|{id,observed_state}],attachments:[.network_attachments[]|{id,observed_state}],object_links:[.network_object_links[]|{id,observed_state}]}')
    jq -e 'all(.nodes[]; .desired_state==.observed_state) and all(.links[]; .observed_state=="connected") and all(.attachments[]; .observed_state=="active") and all(.object_links[]; .observed_state=="connected")' <<<"$topology" >/dev/null
    paths=$(path_snapshot)
    jq --argjson topology "$topology" --argjson paths "$paths" --argjson recovery "$recovery" '.phase="verified" | .verified_at=(now|todate) | .after={topology:$topology,paths:$paths,recovery_task:$recovery}' "$MARKER" >"$MARKER.tmp"
    mv "$MARKER.tmp" "$MARKER"
    echo "PASS: host restart preserved Ubuntu/VyOS dual-stack paths for candidate $candidate"
    ;;
  *)
    echo "usage: $0 prepare|verify" >&2
    exit 2
    ;;
esac
