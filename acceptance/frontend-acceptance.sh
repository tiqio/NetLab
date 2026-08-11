#!/usr/bin/env bash
set -Eeuo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
PROFILE=${NETLAB_ACCEPTANCE_PROFILE:-local}
OUTPUT=${NETLAB_ACCEPTANCE_OUTPUT_DIR:-"$ROOT/web/test-results/acceptance"}
RUN_ID=${NETLAB_ACCEPTANCE_RUN_ID:-"accept-$(date -u +%Y%m%dT%H%M%SZ)-$$"}
SCOPE=${NETLAB_ACCEPTANCE_SCOPE:-full}
[[ "$OUTPUT" == /* ]] || OUTPUT="$ROOT/$OUTPUT"
export NETLAB_ACCEPTANCE_PROFILE="$PROFILE"
export NETLAB_ACCEPTANCE_RUN_ID="$RUN_ID"
export NETLAB_ACCEPTANCE_OUTPUT_DIR="$OUTPUT/$RUN_ID"

case "$PROFILE" in
  local|target-host) ;;
  *) echo "NETLAB_ACCEPTANCE_PROFILE must be local or target-host" >&2; exit 2 ;;
esac
case "$SCOPE" in
  full|topology-unification) ;;
  *) echo "NETLAB_ACCEPTANCE_SCOPE must be full or topology-unification" >&2; exit 2 ;;
esac
if ! [[ ${NETLAB_ACCEPTANCE_TIMEOUT_SCALE:-1} =~ ^[0-9]+([.][0-9]+)?$ ]] || [[ ${NETLAB_ACCEPTANCE_TIMEOUT_SCALE:-1} == 0 ]]; then
  echo "NETLAB_ACCEPTANCE_TIMEOUT_SCALE must be positive" >&2
  exit 2
fi

mkdir -p "$NETLAB_ACCEPTANCE_OUTPUT_DIR"
chmod 700 "$NETLAB_ACCEPTANCE_OUTPUT_DIR"

interrupted=0
child=""
on_signal() {
  interrupted=1
  [[ -n "$child" ]] && kill -INT "$child" 2>/dev/null || true
}
trap on_signal INT TERM HUP

cd "$ROOT/web"
npm run test:acceptance-unit
cd "$ROOT"
./scripts/check-ui-localization.sh
make test-acceptance-schema
cd "$ROOT/web"
playwright_specs=()
if [[ "$SCOPE" == topology-unification ]]; then
  playwright_specs=(
    ../tests/e2e/journeys/topologyVisualRecognition.spec.ts
    ../tests/e2e/journeys/topologyAuthoritativePlacement.spec.ts
    ../tests/e2e/journeys/concurrentClients.spec.ts
    ../tests/e2e/journeys/topologyPlacementRecovery.spec.ts
  )
  export NETLAB_ACCEPTANCE_LAB_PREFIX="topology-${RUN_ID}"
fi
if [[ "$PROFILE" == "target-host" ]]; then
  : "${NETLAB_ACCEPTANCE_BASE_URL:?NETLAB_ACCEPTANCE_BASE_URL is required for target-host acceptance}"
  if [[ "$NETLAB_ACCEPTANCE_BASE_URL" == *"@"* ]]; then
    echo "NETLAB_ACCEPTANCE_BASE_URL must not contain credentials" >&2
    exit 2
  fi
  NODE_PATH="$PWD/node_modules" npx playwright test "${playwright_specs[@]}" --config playwright.target.config.ts &
else
  NODE_PATH="$PWD/node_modules" npx playwright test "${playwright_specs[@]}" --config playwright.config.ts &
fi
child=$!
wait "$child" || status=$?

status=${status:-0}
if (( interrupted )); then status=130; fi
"$ROOT/scripts/check-frontend-artifacts.sh"
if [[ -f "$NETLAB_ACCEPTANCE_OUTPUT_DIR/run-summary.json" ]] &&
   ! grep -q '"status": "passed"' "$NETLAB_ACCEPTANCE_OUTPUT_DIR/run-summary.json"; then
  status=1
fi
exit "$status"
