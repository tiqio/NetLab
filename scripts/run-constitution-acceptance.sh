#!/usr/bin/env bash
set -uo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"
CANDIDATE_ID=${CANDIDATE_ID:-candidate-$(date -u +%Y%m%dT%H%M%SZ)}
OUTPUT=${NETLAB_ACCEPTANCE_OUTPUT:-compliance/evidence/current-candidate.json}
BASE_URL=${NETLAB_BASE_URL:-}
STARTED=$(date -u +%Y-%m-%dT%H:%M:%SZ)
BROWSER_RUN_ID=${NETLAB_ACCEPTANCE_RUN_ID:-${CANDIDATE_ID}-browser}
BROWSER_OUTPUT=${NETLAB_ACCEPTANCE_OUTPUT_DIR:-$ROOT/web/test-results/acceptance/$BROWSER_RUN_ID}
mkdir -p "$(dirname "$OUTPUT")" "$BROWSER_OUTPUT"
GATES=$(mktemp); trap 'rm -f "$GATES"' EXIT
printf '{}\n' >"$GATES"
record() { local name=$1 status=$2 reason=${3:-} details=${4:-}; jq --arg n "$name" --arg s "$status" --arg r "$reason" --arg d "$details" '.[$n]={status:$s} | if $r!="" then .[$n].reason=$r else . end | if $d!="" then .[$n].details=$d else . end' "$GATES" >"$GATES.tmp" && mv "$GATES.tmp" "$GATES"; }
run_gate() { local name=$1; shift; echo "==> $name: $*"; if "$@"; then record "$name" passed; else local code=$?; if [[ $code == 77 || $code == 75 ]]; then record "$name" skipped "command exited $code; rerun: $*"; else record "$name" failed "command exited $code: $*"; fi; fi; }
count_lines() { "$@" 2>/dev/null | sed '/^[[:space:]]*$/d' | wc -l; }
resource_snapshot() {
  local state=${NETLAB_STATE_DIR:-/var/lib/netlab} ownership='[]'
  if [[ -n $BASE_URL ]]; then ownership=$(curl -fsS "$BASE_URL/api/v1/runtime-ownership" 2>/dev/null | jq -c 'if type=="array" then . else [] end' || printf '[]'); fi
  jq -n \
    --argjson qemu "$(find "$state/runtime/qemu" -mindepth 1 -maxdepth 1 2>/dev/null | wc -l)" \
    --argjson nat "$(find "$state/runtime/nat" -mindepth 1 -maxdepth 1 2>/dev/null | wc -l)" \
    --argjson captures "$(find "$state/captures" -type f 2>/dev/null | wc -l)" \
    --argjson namespaces "$(count_lines bash -c "ip netns list | awk '\$1 ~ /^(nlpc|nlsw|nlr)-/ {print \$1}'")" \
    --argjson links "$(count_lines bash -c "ip -j -d link show | jq -r '.[] | select((.ifalias // \"\") | startswith(\"netlab:\")) | .ifname'")" \
    --argjson ownership "$ownership" \
    '{qemu_runtime_directories:$qemu,nat_runtime_directories:$nat,capture_files:$captures,network_namespaces:$namespaces,owned_links:$links,runtime_ownership_records:($ownership|length),unknown_ownership_records:([$ownership[]|select(.cleanup_state=="unknown_observed")]|length)}'
}
baseline_digest() { printf '%s' "$1" | jq -S -c . | sha256sum | awk '{print "sha256:"$1}'; }
BASELINE_RESOURCES=$(resource_snapshot); BASELINE=$(baseline_digest "$BASELINE_RESOURCES")
run_gate format bash -c 'test -z "$(gofmt -l cmd internal tests)"'
run_gate static_analysis go vet ./...
run_gate unit go test ./internal/...
run_gate contract go test ./tests/contract/... -count=1
run_gate frontend bash -c 'cd web && npm test -- --run && npm run build'
if [[ ${NETLAB_PRIVILEGED:-0} == 1 ]]; then run_gate privileged_integration env NETLAB_PRIVILEGED=1 go test ./tests/integration/... -count=1; else record privileged_integration skipped 'set NETLAB_PRIVILEGED=1 on the authoritative host'; fi
run_gate recovery env NETLAB_PRIVILEGED=${NETLAB_PRIVILEGED:-0} go test ./tests/recovery/... -count=1
run_gate security go test ./tests/security/... -count=1
run_gate leak env NETLAB_PRIVILEGED=${NETLAB_PRIVILEGED:-0} CYCLES=${CYCLES:-100} go test ./tests/integration/... -run Leak -count=1
if [[ ${NETLAB_RUN_BROWSER_ACCEPTANCE:-0} == 1 ]]; then
  if [[ -z $BASE_URL ]]; then record browser failed 'NETLAB_BASE_URL is required for target browser acceptance';
  else
    export NETLAB_ACCEPTANCE_BASE_URL=$BASE_URL NETLAB_ACCEPTANCE_RUN_ID=$BROWSER_RUN_ID NETLAB_ACCEPTANCE_OUTPUT_DIR=$BROWSER_OUTPUT
    run_gate browser make test-e2e-target
    if [[ -f $BROWSER_OUTPUT/evidence.json ]]; then record browser "$(jq -r '.status' "$BROWSER_OUTPUT/evidence.json")" "$(jq -r '.cleanup.remediation // [] | join("; ")' "$BROWSER_OUTPUT/evidence.json")" "evidence=$BROWSER_OUTPUT/evidence.json"; fi
  fi
else record browser skipped 'set NETLAB_RUN_BROWSER_ACCEPTANCE=1 and NETLAB_BASE_URL for target browser acceptance'; fi
FINAL_RESOURCES=$(resource_snapshot); FINAL=$(baseline_digest "$FINAL_RESOURCES")
FINISHED=$(date -u +%Y-%m-%dT%H:%M:%SZ)
SCENARIOS='[]'; [[ -f $BROWSER_OUTPUT/evidence.json ]] && SCENARIOS=$(jq -c '[.run_id]' "$BROWSER_OUTPUT/evidence.json")
CONCLUSION=$(jq -r 'if any(.[]; .status=="failed" or .status=="cancelled") then "failed" elif any(.[]; .status=="skipped" or .status=="blocked") then "blocked" else "passed" end' "$GATES")
[[ $BASELINE == "$FINAL" ]] || CONCLUSION=failed
write_output() {
  local prohibited=$1
  jq -n --arg id "acceptance-$CANDIDATE_ID" --arg candidate "$CANDIDATE_ID" --arg status "$CONCLUSION" --argjson gates "$(cat "$GATES")" --argjson scenarios "$SCENARIOS" --arg before "$BASELINE" --arg after "$FINAL" --argjson before_resources "$BASELINE_RESOURCES" --argjson after_resources "$FINAL_RESOURCES" --arg started "$STARTED" --arg finished "$FINISHED" --arg conclusion "$CONCLUSION" --argjson prohibited "$prohibited" '{schema_version:"1.0",id:$id,candidate_id:$candidate,status:$status,gate_results:$gates,scenario_evidence_ids:$scenarios,exceptions:[],cleanup_baseline:{digest:$before,resources:$before_resources},cleanup_final:{digest:$after,resources:$after_resources},redaction_result:{passed:($prohibited==0),prohibited_content_count:$prohibited},conclusion:$conclusion,started_at:$started,finished_at:$finished}' >"$OUTPUT"
}
write_output 0
PROHIBITED=$(go run ./cmd/netlab-compliance scan-evidence --directory "$(dirname "$OUTPUT")" | jq -r .prohibited_content_count)
if ((PROHIBITED > 0)); then CONCLUSION=failed; fi
write_output "$PROHIBITED"
echo "$CONCLUSION: $OUTPUT"
[[ $CONCLUSION == passed ]]
