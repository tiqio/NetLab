#!/usr/bin/env bash
set -euo pipefail

mode=${1:-preflight}
expected_port=${NETLAB_AUTHORITY_PORT:-18082}
fixture=${NETLAB_SS_FIXTURE:-}
proc_root=${NETLAB_PROC_ROOT:-/proc}

if [[ ${NETLAB_RETIRE_LEGACY:-0} == 1 && -z "${NETLAB_SS_FIXTURE:-}" ]] && systemctl is-active --quiet netlab-preview.service; then
  systemctl disable --now netlab-preview.service
fi

listeners() {
  if [[ -n "$fixture" ]]; then
    cat "$fixture"
  else
    ss -H -lntp
  fi
}

is_external() {
  case "$1" in
    127.*:*|localhost:*|\[::1\]:*) return 1 ;;
    *) return 0 ;;
  esac
}

external_count=0
expected_count=0
conflicts=()
while read -r state recvq sendq address peer process; do
  [[ "$state" == "LISTEN" && "$process" == *netlabd* ]] || continue
  is_external "$address" || continue
  ((external_count += 1))
  port=${address##*:}
  if [[ "$port" == "$expected_port" ]]; then
    ((expected_count += 1))
    continue
  fi
  pid=$(sed -n 's/.*pid=\([0-9][0-9]*\).*/\1/p' <<<"$process" | head -1)
  executable=""
  [[ -n "$pid" ]] && executable=$(readlink "$proc_root/$pid/exe" 2>/dev/null || true)
  if [[ ${NETLAB_RETIRE_LEGACY:-0} == 1 && "$executable" == /opt/netlab/netlabd ]]; then
    if [[ ${NETLAB_AUTHORITY_DRY_RUN:-0} == 1 ]]; then
      echo "would retire legacy netlabd pid=$pid address=$address" >&2
    else
      kill "$pid"
      for _ in {1..50}; do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.1
      done
      kill -0 "$pid" 2>/dev/null && kill -KILL "$pid"
    fi
    ((external_count -= 1))
    continue
  fi
  conflicts+=("$address pid=${pid:-unknown} exe=${executable:-unknown}")
done < <(listeners)

if ((${#conflicts[@]})); then
  printf 'conflicting externally reachable NetLab control plane: %s\n' "${conflicts[*]}" >&2
  exit 1
fi
if [[ "$mode" == verify ]]; then
  if ((external_count != 1 || expected_count != 1)); then
    echo "expected exactly one external NetLab authority on port $expected_port; found external=$external_count expected=$expected_count" >&2
    exit 1
  fi
elif ((external_count > 1)); then
  echo "multiple externally reachable NetLab control planes detected" >&2
  exit 1
fi
