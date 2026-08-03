#!/usr/bin/env bash
set -euo pipefail

config_file=${NETLAB_CONFIG:-/etc/netlab/netlab.yaml}
state_dir=${NETLAB_STATE_DIR:-/var/lib/netlab}
database=${NETLAB_DATABASE:-$state_dir/netlab.db}
retention_days=${NETLAB_RETENTION_DAYS:-7}

usage() {
  echo "usage: $0 {backup|cleanup|verify|restore BACKUP|vacuum}" >&2
  exit 2
}

require_stopped() {
  if systemctl is-active --quiet netlab; then
    echo "netlab must be stopped for restore" >&2
    exit 1
  fi
}

case "${1:-}" in
  backup)
    destination=${2:-$state_dir/backups/netlab-$(date -u +%Y%m%dT%H%M%SZ).db}
    install -d -m 0700 "$(dirname "$destination")"
    sqlite3 "$database" ".timeout 10000" ".backup '$destination'"
    chmod 0600 "$destination"
    sha256sum "$destination" >"$destination.sha256"
    echo "$destination"
    ;;
  cleanup)
    find "$state_dir/artifacts" "$state_dir/captures" -type f -mtime "+$retention_days" -delete 2>/dev/null || true
    find "$state_dir/runtime" -type s -delete 2>/dev/null || true
    find "$state_dir/backups" -type f -mtime "+$retention_days" -delete 2>/dev/null || true
    sqlite3 "$database" "DELETE FROM artifacts WHERE expires_at IS NOT NULL AND expires_at < strftime('%Y-%m-%dT%H:%M:%fZ','now'); DELETE FROM idempotency_records WHERE expires_at < strftime('%Y-%m-%dT%H:%M:%fZ','now');"
    ;;
  verify)
    sqlite3 "$database" "PRAGMA integrity_check; PRAGMA foreign_key_check;"
    find "$state_dir/images" -type f -name 'sha256-*' -print0 2>/dev/null | xargs -0 -r sha256sum
    ;;
  restore)
    backup=${2:-}; test -n "$backup" || usage
    require_stopped
    test -f "$backup" || { echo "backup not found" >&2; exit 1; }
    cp -a "$database" "$database.before-restore.$(date -u +%s)" 2>/dev/null || true
    install -m 0600 "$backup" "$database"
    sqlite3 "$database" "PRAGMA integrity_check;"
    ;;
  vacuum)
    sqlite3 "$database" "PRAGMA wal_checkpoint(TRUNCATE); VACUUM; ANALYZE;"
    ;;
  *) usage ;;
esac
