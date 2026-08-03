#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
inventory=${DEPLOYMENT_AUTHORITY:-$root/compliance/deployment-authority.json}
target=${TARGET_HOST:-}

jq -e '
  ([.instances[] | select(.role == "authoritative" and .externally_reachable == true)] | length) == 1 and
  ([.instances[] | select(.role != "authoritative" and .externally_reachable == true)] | length) == 0
' "$inventory" >/dev/null

authoritative_address=$(jq -r '.instances[] | select(.role == "authoritative") | .listen_address' "$inventory")
candidate_id=$(jq -r '.instances[] | select(.role == "authoritative") | .candidate_id' "$inventory")

if [[ -n "$target" ]]; then
  listeners=$(ssh -o BatchMode=yes "$target" 'ss -ltnp')
  grep -F "$authoritative_address" <<<"$listeners" >/dev/null
  if grep -E '(^|:)18080([[:space:]]|$)' <<<"$listeners" | grep -v '127.0.0.1:18080' >/dev/null; then
    echo "preview port 18080 is externally bound" >&2
    exit 5
  fi
fi

printf 'authoritative=%s candidate=%s\n' "$authoritative_address" "$candidate_id"
