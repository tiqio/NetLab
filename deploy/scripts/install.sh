#!/usr/bin/env bash
set -euo pipefail

required=(qemu-system-x86_64 docker ip bridge tc nft tcpdump xorriso ss jq)
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
      work=$(mktemp -d /var/tmp/netlab-install.XXXXXX)
      trap 'rm -rf "$work"' EXIT
      install -m0755 "$NETLAB_PREBUILT_BINARY" "$work/netlabd"
      "$work/netlabd" release >"$work/release.json"
      actual_digest="sha256:$(sha256sum "$work/netlabd" | awk '{print $1}')"
      embedded_digest=$(jq -r '.binary_digest // ""' "$work/release.json")
      [[ -z "$embedded_digest" || "$embedded_digest" == "$actual_digest" ]] || { echo "embedded binary digest does not match candidate artifact" >&2; exit 1; }
      jq --arg digest "$actual_digest" '.binary_digest=$digest' "$work/release.json" >"$work/release.actual.json"
      install -Dm0644 deploy/systemd/netlab.service /etc/systemd/system/netlab.service
      install -d -m0755 /etc/netlab /usr/local/share/netlab/templates/qemu /usr/local/share/netlab/templates/docker
      source_config=/etc/netlab/netlab.yaml
      [[ -f "$source_config" ]] || source_config=deploy/config/netlab.example.yaml
      deploy/scripts/prepare-release-config.sh "$source_config" "$work/release.actual.json" "$work/netlab.yaml"
      deploy/scripts/generate-template-readiness.sh "$work/release.actual.json" "$work/template-readiness.json"
      NETLAB_DEPLOYMENT_ROLE=authoritative "$work/netlabd" validate-config -config "$work/netlab.yaml"
      install -m0644 templates/qemu/manifest.yaml /usr/local/share/netlab/templates/qemu/manifest.yaml
      install -m0644 templates/docker/manifest.yaml /usr/local/share/netlab/templates/docker/manifest.yaml
      backup="$work/backup"
      mkdir -p "$backup"
      for path in /usr/local/bin/netlabd /etc/netlab/netlab.yaml /etc/netlab/template-readiness.json; do
        [[ -e "$path" ]] && cp -a "$path" "$backup/$(basename "$path")"
      done
      rollback() {
        status=$?
        if ((status != 0)); then
          [[ -e "$backup/netlabd" ]] && install -m0755 "$backup/netlabd" /usr/local/bin/netlabd
          [[ -e "$backup/netlab.yaml" ]] && install -m0600 "$backup/netlab.yaml" /etc/netlab/netlab.yaml
          [[ -e "$backup/template-readiness.json" ]] && install -m0644 "$backup/template-readiness.json" /etc/netlab/template-readiness.json
          systemctl daemon-reload || true
          systemctl restart netlab || true
        fi
        rm -rf "$work"
        exit "$status"
      }
      trap rollback EXIT
      install -m0755 "$work/netlabd" /usr/local/bin/netlabd
      install -m0600 "$work/netlab.yaml" /etc/netlab/netlab.yaml
      install -m0644 "$work/template-readiness.json" /etc/netlab/template-readiness.json
    else
      make install
    fi
    systemctl daemon-reload
    systemctl enable netlab
    systemctl restart netlab
    verified=0
    for _ in {1..900}; do
      if /usr/local/libexec/netlab/check-authority.sh verify >/dev/null 2>&1; then
        verified=1
        break
      fi
      systemctl is-active --quiet netlab.service || break
      sleep 0.2
    done
    ((verified == 1)) || { echo "authoritative listener did not become ready within 180 seconds" >&2; /usr/local/libexec/netlab/check-authority.sh verify || true; exit 1; }
    trap - EXIT
    [[ -n "${work:-}" ]] && rm -rf "$work"
    ;;
  uninstall) systemctl disable --now netlab || true; rm -f /etc/systemd/system/netlab.service /usr/local/bin/netlabd; systemctl daemon-reload ;;
  check) echo "host prerequisites satisfied" ;;
  *) echo "usage: $0 [install|uninstall|check]" >&2; exit 2 ;;
esac
