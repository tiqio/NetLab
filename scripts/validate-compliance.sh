#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

for schema in specs/002-constitution-gap-closure/contracts/*.json; do
  jq empty "$schema"
done

go run ./cmd/netlab-compliance validate \
  --ledger compliance/constitution-ledger.json \
  --deployment compliance/deployment-authority.json \
  --templates compliance/template-readiness.json \
  --evidence-dir compliance/evidence

go run ./cmd/netlab-compliance report --ledger compliance/constitution-ledger.json
