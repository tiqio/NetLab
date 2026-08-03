#!/usr/bin/env bash
set -euo pipefail

BASE_URL=${NETLAB_BASE_URL:-http://127.0.0.1:8088}
MARKER=${NETLAB_HOST_RESTART_MARKER:-/var/lib/netlab/acceptance-host-restart.json}
ALLOW_REBOOT=${NETLAB_ACCEPTANCE_ALLOW_REBOOT:-0}
PHASE=${1:-prepare}

require() { command -v "$1" >/dev/null || { echo "$1 is required" >&2; exit 2; }; }
api() { curl -fsS "$BASE_URL/api/v1$1"; }
require curl; require jq

case "$PHASE" in
  prepare)
    [[ $ALLOW_REBOOT == 1 ]] || { echo "SKIP: set NETLAB_ACCEPTANCE_ALLOW_REBOOT=1 for operator-controlled host restart" >&2; exit 77; }
    capabilities=$(api /capabilities)
    candidate=$(jq -r '.release.candidate_id // empty' <<<"$capabilities")
    [[ -n $candidate ]] || { echo "candidate identity unavailable" >&2; exit 1; }
    baseline=$(api /runtime-ownership | jq -S 'map({resource_type,resource_id,object_kind,cleanup_state})')
    install -d -m0700 "$(dirname "$MARKER")"
    jq -n --arg candidate "$candidate" --argjson baseline "$baseline" '{schema_version:"1.0",candidate_id:$candidate,prepared_at:(now|todate),baseline:$baseline,phase:"prepared"}' >"$MARKER"
    chmod 0600 "$MARKER"
    echo "PREPARED_REBOOT: run the host reboot, then execute $0 verify"
    exit 75
    ;;
  verify)
    [[ -f $MARKER ]] || { echo "restart marker not found: $MARKER" >&2; exit 2; }
    capabilities=$(api /capabilities)
    candidate=$(jq -r '.release.candidate_id // empty' <<<"$capabilities")
    expected=$(jq -r .candidate_id "$MARKER")
    [[ $candidate == "$expected" ]] || { echo "candidate changed across restart" >&2; exit 1; }
    recovery=$(api /tasks | jq -c 'map(select(.kind=="system.recovery")) | max_by(.created_at)')
    [[ $(jq -r '.state // empty' <<<"$recovery") == succeeded ]] || { echo "$recovery" >&2; exit 1; }
    final=$(api /runtime-ownership | jq -S 'map({resource_type,resource_id,object_kind,cleanup_state})')
    jq --argjson final "$final" --argjson recovery "$recovery" '.phase="verified" | .verified_at=(now|todate) | .final=$final | .recovery_task=$recovery' "$MARKER" >"$MARKER.tmp"
    mv "$MARKER.tmp" "$MARKER"
    echo "PASS: host restart recovery verified for candidate $candidate"
    ;;
  *) echo "usage: $0 prepare|verify" >&2; exit 2 ;;
esac
