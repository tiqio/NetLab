#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

forbidden_files=$(find . \
  -path './web/node_modules' -prune -o \
  -path './web/dist' -prune -o \
  -path './test-results' -prune -o \
  -path './web/test-results' -prune -o \
  -path './web/playwright-report' -prune -o \
  -type f \( -name '*.pcap' -o -name '*.pcapng' -o -name '*.qcow2' -o -name '*.raw' -o -name '*.iso' -o -name '*.key' \) -print)

if [[ -n "$forbidden_files" ]]; then
  printf 'forbidden frontend/runtime artifacts found:\n%s\n' "$forbidden_files" >&2
  exit 1
fi

runtime_evidence=$(find . \
  -path './test-results' -prune -o \
  -path './web/test-results' -prune -o \
  -path './acceptance/.runs' -prune -o \
  -type f \( -name '*.ledger.json' -o -name '*.evidence.json' \) -print)

if [[ -n "$runtime_evidence" ]]; then
  printf 'acceptance runtime evidence found outside ignored output directories:\n%s\n' "$runtime_evidence" >&2
  exit 1
fi

if rg -n --hidden \
  -g '!web/node_modules/**' -g '!web/dist/**' -g '!test-results/**' \
  -g '!web/test-results/**' \
  -g '!web/playwright-report/**' -g '!internal/api/http/webdist/**' \
  -e '-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----' \
  -e 'https?://[^/@[:space:]]+:[^/@[:space:]]+@' \
  .; then
  echo 'credential-like material found in frontend artifacts' >&2
  exit 1
fi

topology_assets='web/src/assets/topology'
unexpected_topology_assets=$(find "$topology_assets" -maxdepth 1 -type f \
  ! -name '*.svg' ! -name 'NOTICE.md' -print)
if [[ -n "$unexpected_topology_assets" ]]; then
  printf 'unsupported topology assets found:\n%s\n' "$unexpected_topology_assets" >&2
  exit 1
fi

if rg -n -i -P -e 'https?://(?!www\.w3\.org/2000/svg)' -e 'eve-ng|unetlab' \
  "$topology_assets"/*.svg web/src/features/topology/topologySymbols.ts; then
  echo 'remote or EVE-NG-derived topology artwork reference found' >&2
  exit 1
fi

for symbol in "$topology_assets"/*.svg; do
  if ! rg -q "$(basename "$symbol")" "$topology_assets/NOTICE.md"; then
    echo "topology symbol is missing from NOTICE.md: $symbol" >&2
    exit 1
  fi
done

payload_roots=()
for path in acceptance/.runs test-results web/test-results web/playwright-report; do
  [[ -d "$path" ]] && payload_roots+=("$path")
done
if ((${#payload_roots[@]})) && rg -n -i \
  -e 'console[_ -]?output' -e 'guest[_ -]?output' \
  -e 'packet[_ -]?payload' -e 'payload_(hex|base64)' \
  "${payload_roots[@]}"; then
  echo 'unsafe console, guest, or packet payload content marker found in retained runtime output' >&2
  exit 1
fi

for evidence in $(find acceptance/.runs test-results web/test-results -type f -name 'visual-audit.json' 2>/dev/null); do
  node -e 'const fs=require("fs"); const value=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); if(!value.summary || !Array.isArray(value.results)) process.exit(1)' "$evidence" || {
    echo "invalid visual audit evidence: $evidence" >&2
    exit 1
  }
done

echo 'frontend artifact hygiene check passed'
