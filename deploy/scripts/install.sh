#!/usr/bin/env bash
set -euo pipefail

required=(qemu-system-x86_64 docker ip bridge tc nft tcpdump xorriso)
for command_name in "${required[@]}"; do
  command -v "$command_name" >/dev/null || { echo "missing required command: $command_name" >&2; exit 1; }
done
test -e /dev/kvm || { echo "/dev/kvm is unavailable" >&2; exit 1; }

case "${1:-install}" in
  install) make install; systemctl daemon-reload; systemctl enable --now netlab ;;
  uninstall) systemctl disable --now netlab || true; rm -f /etc/systemd/system/netlab.service /usr/local/bin/netlabd; systemctl daemon-reload ;;
  check) echo "host prerequisites satisfied" ;;
  *) echo "usage: $0 [install|uninstall|check]" >&2; exit 2 ;;
esac
