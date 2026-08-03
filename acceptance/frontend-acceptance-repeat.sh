#!/usr/bin/env bash
set -Eeuo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
OUTPUT=${NETLAB_ACCEPTANCE_OUTPUT_DIR:-"$ROOT/web/test-results/acceptance-repeat"}
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
FIRST_ID="repeat-${STAMP}-1"
SECOND_ID="repeat-${STAMP}-2"

NETLAB_ACCEPTANCE_OUTPUT_DIR="$OUTPUT" NETLAB_ACCEPTANCE_RUN_ID="$FIRST_ID" \
  "$ROOT/acceptance/frontend-acceptance.sh"
NETLAB_ACCEPTANCE_OUTPUT_DIR="$OUTPUT" NETLAB_ACCEPTANCE_RUN_ID="$SECOND_ID" \
  "$ROOT/acceptance/frontend-acceptance.sh"

node "$ROOT/scripts/compare-frontend-acceptance-runs.mjs" \
  "$OUTPUT/$FIRST_ID/evidence.json" \
  "$OUTPUT/$SECOND_ID/evidence.json"
