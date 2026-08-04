#!/usr/bin/env bash
set -euo pipefail

config_file=${NETLAB_CONFIG:-/etc/netlab/netlab.yaml}
state_dir=${NETLAB_STATE_DIR:-/var/lib/netlab}
database=${NETLAB_DATABASE:-$state_dir/netlab.db}
service_name=${NETLAB_SERVICE_NAME:-netlab.service}
retention_days=${NETLAB_RETENTION_DAYS:-30}
batch_size=${NETLAB_RETENTION_BATCH_SIZE:-10000}
lock_file=${NETLAB_MAINTENANCE_LOCK:-$state_dir/.maintenance.lock}
max_io_pressure=${NETLAB_MAX_IO_PRESSURE:-80}

usage() {
  echo "usage: $0 {inventory|backup [DESTINATION]|cleanup|verify|restore BACKUP|vacuum|reset-labs [--execute]|prune [--execute]}" >&2
  exit 2
}

require_tools() {
  local tool
  for tool in sqlite3 sha256sum awk df stat flock sync; do
    command -v "$tool" >/dev/null || { echo "missing required command: $tool" >&2; exit 1; }
  done
}

require_database() { test -f "$database" || { echo "database not found: $database" >&2; exit 1; }; }

require_stopped() {
  [[ ${NETLAB_SKIP_SERVICE_CHECK:-0} == 1 ]] && return
  if systemctl is-active --quiet "$service_name"; then
    echo "$service_name must be stopped for destructive database maintenance" >&2
    exit 1
  fi
}

sqlite_literal() { printf "'%s'" "${1//\'/\'\'}"; }

integrity_check() {
  local path=$1 result
  result=$(sqlite3 -batch "$path" ".timeout 30000" "PRAGMA integrity_check; PRAGMA foreign_key_check;")
  [[ "$result" == "ok" ]] || { echo "database integrity check failed for $path: $result" >&2; return 1; }
}

fsync_path() {
  sync -f "$1"
  sync -f "$(dirname "$1")"
}

io_preflight() {
  [[ ${NETLAB_SKIP_IO_HEALTH_CHECK:-0} == 1 ]] && return
  local pressure=0
  if [[ -r /proc/pressure/io ]]; then
    pressure=$(awk '/^full / {for(i=1;i<=NF;i++) if($i ~ /^avg10=/){sub("avg10=","",$i); print int($i)}}' /proc/pressure/io)
    pressure=${pressure:-0}
    if ((pressure >= max_io_pressure)); then
      echo "host I/O pressure ${pressure}% is at or above safety limit ${max_io_pressure}%; retry after storage recovery or set NETLAB_MAX_IO_PRESSURE deliberately" >&2
      exit 1
    fi
  fi
  local size available
  size=$(stat -c %s "$database" 2>/dev/null || echo 0)
  available=$(df -PB1 "$(dirname "$database")" | awk 'NR==2 {print $4}')
  if ((available < size * 2 + 67108864)); then
    echo "insufficient free space for atomic SQLite maintenance: need at least $((size * 2 + 67108864)) bytes, have $available" >&2
    exit 1
  fi
}

backup_database() {
  local destination=${1:-$state_dir/backups/netlab-$(date -u +%Y%m%dT%H%M%SZ).db}
  local temporary="${destination}.tmp.$$"
  install -d -m 0700 "$(dirname "$destination")"
  rm -f "$temporary"
  sqlite3 -batch "$database" ".timeout 30000" ".backup $(sqlite_literal "$temporary")"
  integrity_check "$temporary"
  chmod 0600 "$temporary"
  fsync_path "$temporary"
  mv -f "$temporary" "$destination"
  fsync_path "$destination"
  sha256sum "$destination" >"$destination.sha256.tmp"
  mv -f "$destination.sha256.tmp" "$destination.sha256"
  fsync_path "$destination.sha256"
  echo "$destination"
}

verify_checksum() {
  local path=$1 checksum expected actual
  checksum="$path.sha256"
  [[ -f "$checksum" ]] || return 0
  expected=$(awk 'NR==1 {print $1}' "$checksum")
  actual=$(sha256sum "$path" | awk '{print $1}')
  [[ "$actual" == "$expected" ]] || { echo "backup checksum mismatch for $path" >&2; return 1; }
}

atomic_replace() {
  local staged=$1 stamp rollback_dir committed=0
  stamp=$(date -u +%Y%m%dT%H%M%SZ)-$$
  rollback_dir="$state_dir/backups/rollback-$stamp"
  install -d -m 0700 "$rollback_dir"
  rollback() {
    local status=$?
    if ((committed == 0)); then
      rm -f "$database" "$database-wal" "$database-shm"
      for name in netlab.db netlab.db-wal netlab.db-shm; do
        [[ -e "$rollback_dir/$name" ]] && mv -f "$rollback_dir/$name" "$(dirname "$database")/$name"
      done
      sync -f "$(dirname "$database")" || true
    fi
    exit "$status"
  }
  trap rollback ERR INT TERM
  [[ -e "$database" ]] && mv -f "$database" "$rollback_dir/netlab.db"
  [[ -e "$database-wal" ]] && mv -f "$database-wal" "$rollback_dir/netlab.db-wal"
  [[ -e "$database-shm" ]] && mv -f "$database-shm" "$rollback_dir/netlab.db-shm"
  mv -f "$staged" "$database"
  chmod 0600 "$database"
  fsync_path "$database"
  integrity_check "$database"
  committed=1
  trap - ERR INT TERM
  echo "$rollback_dir"
}

inventory() {
  require_database
  sqlite3 -json "$database" ".timeout 30000" "SELECT
    (SELECT count(*) FROM laboratories) AS laboratories,
    (SELECT count(*) FROM nodes) AS nodes,
    (SELECT count(*) FROM network_objects) AS network_objects,
    (SELECT count(*) FROM operation_tasks) AS operation_tasks,
    (SELECT count(*) FROM audit_events) AS audit_events,
    (SELECT count(*) FROM outbox_events) AS outbox_events,
    (SELECT count(*) FROM captures) AS captures,
    (SELECT count(*) FROM traffic_filters) AS traffic_filters,
    (SELECT count(*) FROM traffic_observations) AS traffic_observations,
    (SELECT count(*) FROM image_versions) AS image_versions,
    (SELECT count(*) FROM device_templates) AS device_templates,
    (SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()) AS database_bytes;"
}

cleanup_owned_runtime() {
  if command -v docker >/dev/null; then
    mapfile -t containers < <(docker ps -aq --filter label=io.netlab.node_id 2>/dev/null || true)
    ((${#containers[@]} == 0)) || docker rm -f "${containers[@]}" >/dev/null
  fi
  if command -v jq >/dev/null && command -v ip >/dev/null; then
    mapfile -t links < <(ip -j -d link show 2>/dev/null | jq -r '.[] | select((.ifalias // "") | startswith("netlab:")) | .ifname')
    local link
    for link in "${links[@]}"; do ip link delete dev "$link" 2>/dev/null || true; done
    mapfile -t namespaces < <(ip netns list 2>/dev/null | awk '$1 ~ /^(nlpc|nlsw|nlr|n2sw|n2r)/ {print $1}')
    local namespace
    for namespace in "${namespaces[@]}"; do ip netns delete "$namespace" 2>/dev/null || true; done
  fi
  if command -v nft >/dev/null && nft list table inet netlab_nat >/dev/null 2>&1; then
    nft delete table inet netlab_nat
  fi
  if command -v jq >/dev/null; then
    local manifest pid command_line
    while IFS= read -r -d '' manifest; do
      pid=$(jq -r '.pid // 0' "$manifest" 2>/dev/null || echo 0)
      [[ "$pid" =~ ^[0-9]+$ ]] || continue
      ((pid > 1)) || continue
      command_line=$(tr '\0' ' ' <"/proc/$pid/cmdline" 2>/dev/null || true)
      [[ "$command_line" == *qemu-system* && "$command_line" == *guest=netlab:* ]] || continue
      kill -TERM "$pid" 2>/dev/null || true
    done < <(find "$state_dir/runtime/qemu" -name launch.json -type f -print0 2>/dev/null)
    sleep 1
    while IFS= read -r -d '' manifest; do
      pid=$(jq -r '.pid // 0' "$manifest" 2>/dev/null || echo 0)
      [[ "$pid" =~ ^[0-9]+$ ]] || continue
      command_line=$(tr '\0' ' ' <"/proc/$pid/cmdline" 2>/dev/null || true)
      [[ "$command_line" == *qemu-system* && "$command_line" == *guest=netlab:* ]] && kill -KILL "$pid" 2>/dev/null || true
    done < <(find "$state_dir/runtime/qemu" -name launch.json -type f -print0 2>/dev/null)
  fi
  for directory in "$state_dir/runtime" "$state_dir/captures" "$state_dir/artifacts"; do
    [[ -d "$directory" ]] || continue
    find "$directory" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
  done
}

reset_labs() {
  local mode=${1:-} database_mode=${NETLAB_RESET_DATABASE_MODE:-fresh}
  inventory
  [[ "$mode" == "--execute" ]] || { echo "dry run only; rerun with --execute and NETLAB_RESET_CONFIRM=DELETE-ALL-NETLAB-LABORATORIES" >&2; return 0; }
  [[ ${NETLAB_RESET_CONFIRM:-} == DELETE-ALL-NETLAB-LABORATORIES ]] || { echo "set NETLAB_RESET_CONFIRM=DELETE-ALL-NETLAB-LABORATORIES" >&2; exit 1; }
  require_stopped
  io_preflight
  local backup=${NETLAB_RESET_BACKUP:-}
  if [[ -n "$backup" ]]; then
    test -f "$backup" || { echo "reset backup not found: $backup" >&2; exit 1; }
    verify_checksum "$backup"
    integrity_check "$backup"
  else
    backup=$(backup_database)
  fi
  cleanup_owned_runtime
  if [[ "$database_mode" == "fresh" ]]; then
    local staged schema
    staged="$(dirname "$database")/.netlab.reset.$$.db"
    schema="$(dirname "$database")/.netlab.reset.$$.schema"
    rm -f "$staged" "$schema"
    sqlite3 -batch "$database" ".schema --nosys" >"$schema"
    sqlite3 -batch "$staged" <"$schema"
    rm -f "$schema"
    sqlite3 -batch "$staged" <<SQL
.timeout 30000
PRAGMA foreign_keys=ON;
ATTACH DATABASE $(sqlite_literal "$database") AS source;
BEGIN IMMEDIATE;
INSERT INTO schema_migrations SELECT * FROM source.schema_migrations;
INSERT INTO image_versions SELECT * FROM source.image_versions;
INSERT INTO device_templates SELECT * FROM source.device_templates;
INSERT INTO template_versions SELECT * FROM source.template_versions;
COMMIT;
DETACH DATABASE source;
SQL
    integrity_check "$staged"
    atomic_replace "$staged" >/dev/null
  elif [[ "$database_mode" == "in-place" ]]; then
    sqlite3 -batch "$database" <<'SQL'
.timeout 30000
PRAGMA foreign_keys=ON;
BEGIN IMMEDIATE;
DELETE FROM traffic_observations;
DELETE FROM traffic_filters;
DELETE FROM captures;
DELETE FROM port_mappings;
DELETE FROM topology_endpoint_reservations;
DELETE FROM network_object_links;
DELETE FROM network_attachments;
DELETE FROM nat_service_observations;
DELETE FROM topology_placements;
DELETE FROM links;
DELETE FROM interfaces;
DELETE FROM node_runtime_capabilities;
DELETE FROM nodes;
DELETE FROM network_object_tombstones;
DELETE FROM network_objects;
DELETE FROM laboratories;
DELETE FROM operation_tasks;
DELETE FROM audit_events;
DELETE FROM outbox_events;
DELETE FROM idempotency_records;
DELETE FROM artifacts;
DELETE FROM runtime_ownership WHERE COALESCE(json_extract(metadata_json,'$.ownership_class'),'managed') <> 'foreign_observed';
COMMIT;
SQL
  else
    echo "NETLAB_RESET_DATABASE_MODE must be fresh or in-place" >&2
    exit 1
  fi
  integrity_check "$database"
  echo "reset completed; mode=$database_mode backup=$backup"
}

prune_history() {
  local mode=${1:-} cutoff
  cutoff=$(date -u -d "$retention_days days ago" +%Y-%m-%dT%H:%M:%S.%NZ)
  sqlite3 -json "$database" "SELECT
    (SELECT count(*) FROM operation_tasks WHERE state IN ('succeeded','failed','cancelled') AND COALESCE(finished_at,created_at) < '$cutoff') AS tasks,
    (SELECT count(*) FROM audit_events WHERE occurred_at < '$cutoff') AS audits,
    (SELECT count(*) FROM outbox_events WHERE published_at IS NOT NULL AND occurred_at < '$cutoff') AS outbox,
    (SELECT count(*) FROM traffic_observations WHERE last_seen_at < '$cutoff') AS observations;"
  [[ "$mode" == "--execute" ]] || { echo "dry run only; rerun with --execute" >&2; return 0; }
  require_stopped
  local backup
  backup=$(backup_database)
  sqlite3 -batch "$database" ".timeout 30000" "BEGIN IMMEDIATE;
DELETE FROM traffic_observations WHERE rowid IN (SELECT rowid FROM traffic_observations WHERE last_seen_at < '$cutoff' LIMIT $batch_size);
DELETE FROM operation_tasks WHERE id IN (SELECT id FROM operation_tasks WHERE state IN ('succeeded','failed','cancelled') AND COALESCE(finished_at,created_at) < '$cutoff' ORDER BY COALESCE(finished_at,created_at) LIMIT $batch_size);
DELETE FROM audit_events WHERE id IN (SELECT id FROM audit_events WHERE occurred_at < '$cutoff' ORDER BY occurred_at LIMIT $batch_size);
DELETE FROM outbox_events WHERE sequence IN (SELECT sequence FROM outbox_events WHERE published_at IS NOT NULL AND occurred_at < '$cutoff' AND sequence <= (SELECT COALESCE(MAX(sequence),0)-10000 FROM outbox_events) ORDER BY sequence LIMIT $batch_size);
COMMIT;"
  integrity_check "$database"
  echo "prune completed; backup=$backup"
}

require_tools
install -d -m 0700 "$state_dir"
exec 9>"$lock_file"
flock -n 9 || { echo "another NetLab maintenance operation is running" >&2; exit 1; }

case "${1:-}" in
  inventory) inventory ;;
  backup) require_database; backup_database "${2:-}" ;;
  cleanup)
    find "$state_dir/artifacts" "$state_dir/captures" -type f -mtime "+$retention_days" -delete 2>/dev/null || true
    find "$state_dir/runtime" -type s -delete 2>/dev/null || true
    find "$state_dir/backups" -type f -mtime "+$retention_days" -delete 2>/dev/null || true
    sqlite3 "$database" "DELETE FROM artifacts WHERE expires_at IS NOT NULL AND expires_at < strftime('%Y-%m-%dT%H:%M:%fZ','now'); DELETE FROM idempotency_records WHERE expires_at < strftime('%Y-%m-%dT%H:%M:%fZ','now');"
    ;;
  verify) require_database; integrity_check "$database"; find "$state_dir/images" -type f -name 'sha256-*' -print0 2>/dev/null | xargs -0 -r sha256sum ;;
  restore)
    require_database; require_stopped; io_preflight
    backup=${2:-}; test -n "$backup" || usage; test -f "$backup" || { echo "backup not found: $backup" >&2; exit 1; }
    verify_checksum "$backup"; integrity_check "$backup"
    staged="$(dirname "$database")/.netlab.restore.$$.db"
    rm -f "$staged"
    sqlite3 -batch "$backup" ".timeout 30000" ".backup $(sqlite_literal "$staged")"
    integrity_check "$staged"
    atomic_replace "$staged"
    ;;
  vacuum)
    require_database; require_stopped; io_preflight
    staged="$(dirname "$database")/.netlab.vacuum.$$.db"
    rm -f "$staged"
    sqlite3 -batch "$database" ".timeout 30000" "PRAGMA wal_checkpoint(TRUNCATE); VACUUM INTO $(sqlite_literal "$staged");"
    integrity_check "$staged"
    atomic_replace "$staged"
    ;;
  reset-labs) require_database; reset_labs "${2:-}" ;;
  prune) require_database; prune_history "${2:-}" ;;
  *) usage ;;
esac
