PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  checksum TEXT NOT NULL,
  applied_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS laboratories (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL COLLATE NOCASE UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL DEFAULT 1,
  recovery_policy TEXT NOT NULL DEFAULT 'auto_restore',
  lifecycle_state TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  laboratory_id TEXT NOT NULL REFERENCES laboratories(id),
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  template_version_id TEXT,
  revision INTEGER NOT NULL DEFAULT 1,
  desired_state TEXT NOT NULL DEFAULT 'stopped',
  observed_state TEXT NOT NULL DEFAULT 'unknown',
  cpu_count INTEGER,
  cpu_quota_micros INTEGER,
  memory_mib INTEGER,
  config_json TEXT NOT NULL DEFAULT '{}',
  runtime_owner_json TEXT,
  last_error_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(laboratory_id, name)
);
CREATE TABLE IF NOT EXISTS interfaces (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL REFERENCES nodes(id),
  slot INTEGER NOT NULL,
  name TEXT NOT NULL,
  driver TEXT,
  mac_address TEXT NOT NULL,
  desired_link_id TEXT,
  oper_state TEXT NOT NULL DEFAULT 'unknown',
  revision INTEGER NOT NULL DEFAULT 1,
  UNIQUE(node_id, slot), UNIQUE(mac_address)
);
CREATE TABLE IF NOT EXISTS links (
  id TEXT PRIMARY KEY,
  laboratory_id TEXT NOT NULL REFERENCES laboratories(id),
  endpoint_a_id TEXT NOT NULL REFERENCES interfaces(id),
  endpoint_b_id TEXT NOT NULL REFERENCES interfaces(id),
  revision INTEGER NOT NULL DEFAULT 1,
  desired_state TEXT NOT NULL,
  observed_state TEXT NOT NULL,
  runtime_bridge_name TEXT,
  CHECK(endpoint_a_id <> endpoint_b_id)
);
CREATE TABLE IF NOT EXISTS operation_tasks (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  idempotency_key TEXT,
  request_fingerprint TEXT,
  requested_revision INTEGER,
  state TEXT NOT NULL,
  progress_current INTEGER NOT NULL DEFAULT 0,
  progress_total INTEGER NOT NULL DEFAULT 0,
  result_json TEXT,
  error_json TEXT,
  cancel_requested_at TEXT,
  created_at TEXT NOT NULL,
  started_at TEXT,
  finished_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_idempotency ON operation_tasks(kind, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_state_created ON operation_tasks(state, created_at);
CREATE TABLE IF NOT EXISTS audit_events (
  id TEXT PRIMARY KEY, actor_class TEXT NOT NULL, action TEXT NOT NULL,
  resource_type TEXT NOT NULL, resource_id TEXT NOT NULL, task_id TEXT,
  outcome TEXT NOT NULL, correlation_id TEXT NOT NULL, details_json TEXT,
  occurred_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS artifacts (
  id TEXT PRIMARY KEY, kind TEXT NOT NULL, path TEXT NOT NULL UNIQUE,
  media_type TEXT NOT NULL, size_bytes INTEGER NOT NULL, sha256 TEXT NOT NULL,
  owner_type TEXT NOT NULL, owner_id TEXT NOT NULL, created_at TEXT NOT NULL,
  expires_at TEXT, deletion_state TEXT NOT NULL DEFAULT 'active'
);
CREATE INDEX IF NOT EXISTS idx_artifact_expiry ON artifacts(expires_at, deletion_state);
CREATE TABLE IF NOT EXISTS outbox_events (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  event_type TEXT NOT NULL, laboratory_id TEXT, resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL, revision INTEGER NOT NULL, task_id TEXT,
  payload_json TEXT NOT NULL DEFAULT '{}', occurred_at TEXT NOT NULL,
  published_at TEXT
);
CREATE TABLE IF NOT EXISTS runtime_ownership (
  resource_type TEXT NOT NULL, resource_id TEXT NOT NULL, object_kind TEXT NOT NULL,
  object_name TEXT NOT NULL, metadata_json TEXT NOT NULL DEFAULT '{}',
  cleanup_state TEXT NOT NULL DEFAULT 'active', PRIMARY KEY(resource_type, resource_id, object_kind, object_name)
);
