#!/usr/bin/env bash
set -euo pipefail

required=(qemu-system-x86_64 docker ip bridge tc nft tcpdump xorriso ss)
for command_name in "${required[@]}"; do
  command -v "$command_name" >/dev/null || { echo "missing required command: $command_name" >&2; exit 1; }
done
test -e /dev/kvm || { echo "/dev/kvm is unavailable" >&2; exit 1; }

case "${1:-install}" in
  install)
    install -Dm0755 deploy/scripts/check-authority.sh /usr/local/libexec/netlab/check-authority.sh
    NETLAB_RETIRE_LEGACY=${NETLAB_RETIRE_LEGACY:-0} /usr/local/libexec/netlab/check-authority.sh preflight
    if [[ -n "${NETLAB_PREBUILT_BINARY:-}" ]]; then
      test -x "$NETLAB_PREBUILT_BINARY" || { echo "prebuilt binary is not executable: $NETLAB_PREBUILT_BINARY" >&2; exit 1; }
      install -Dm0755 "$NETLAB_PREBUILT_BINARY" /usr/local/bin/netlabd
      install -Dm0644 deploy/systemd/netlab.service /etc/systemd/system/netlab.service
      install -d -m0755 /etc/netlab /usr/local/share/netlab/templates/qemu /usr/local/share/netlab/templates/docker
      if [[ ! -f /etc/netlab/netlab.yaml ]]; then
        install -m0644 deploy/config/netlab.example.yaml /etc/netlab/netlab.yaml
      fi
      install -m0644 templates/qemu/manifest.yaml /usr/local/share/netlab/templates/qemu/manifest.yaml
      install -m0644 templates/docker/manifest.yaml /usr/local/share/netlab/templates/docker/manifest.yaml
    else
      make install
    fi
    systemctl daemon-reload
    systemctl enable netlab
    systemctl restart netlab
    /usr/local/libexec/netlab/check-authority.sh verify
    ;;
  uninstall) systemctl disable --now netlab || true; rm -f /etc/systemd/system/netlab.service /usr/local/bin/netlabd; systemctl daemon-reload ;;
  check) echo "host prerequisites satisfied" ;;
  *) echo "usage: $0 [install|uninstall|check]" >&2; exit 2 ;;
esac
