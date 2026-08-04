#!/usr/bin/env bash
set -euo pipefail

input=${1:?input config required}
release_json=${2:?release json required}
output=${3:?output config required}

version=$(jq -er '.version | select(length>0)' "$release_json")
candidate=$(jq -er '.candidate_id | select(length>0)' "$release_json")
binary_digest=$(jq -er '.binary_digest | select(test("^sha256:[0-9a-f]{64}$"))' "$release_json")
contract_digest=$(jq -er '.contract_digest | select(test("^sha256:[0-9a-f]{64}$"))' "$release_json")
built_at=$(jq -er '.built_at | select(length>0)' "$release_json")

quote_yaml() { sed "s/'/''/g" <<<"$1"; }
block=$(cat <<EOF
release:
  version: '$(quote_yaml "$version")'
  candidate_id: '$(quote_yaml "$candidate")'
  binary_digest: '$(quote_yaml "$binary_digest")'
  contract_digest: '$(quote_yaml "$contract_digest")'
  built_at: '$(quote_yaml "$built_at")'
EOF
)

awk -v block="$block" '
  BEGIN { in_release=0; emitted=0 }
  /^release:[[:space:]]*$/ { print block; in_release=1; emitted=1; next }
  in_release && /^[^[:space:]]/ { in_release=0 }
  !emitted && /^deployment:[[:space:]]*$/ { print block; emitted=1 }
  !in_release { print }
  END { if (!emitted) print block }
' "$input" >"$output"
chmod 0600 "$output"
