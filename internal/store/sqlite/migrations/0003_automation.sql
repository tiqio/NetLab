CREATE TABLE IF NOT EXISTS idempotency_records (
  scope TEXT NOT NULL,
  key TEXT NOT NULL,
  request_fingerprint TEXT NOT NULL,
  state TEXT NOT NULL,
  status_code INTEGER,
  response_json BLOB,
  error_json BLOB,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  PRIMARY KEY(scope, key)
);
CREATE INDEX IF NOT EXISTS idx_idempotency_expiry ON idempotency_records(expires_at);

CREATE TABLE IF NOT EXISTS port_mappings (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  protocol TEXT NOT NULL,
  host_address TEXT NOT NULL,
  host_port INTEGER NOT NULL,
  guest_address TEXT NOT NULL,
  guest_port INTEGER NOT NULL,
  revision INTEGER NOT NULL DEFAULT 1,
  observed_state TEXT NOT NULL DEFAULT 'pending',
  last_error_json TEXT,
  created_at TEXT NOT NULL,
  UNIQUE(protocol, host_address, host_port)
);

CREATE TABLE IF NOT EXISTS captures (
  id TEXT PRIMARY KEY,
  laboratory_id TEXT REFERENCES laboratories(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL,
  source_id TEXT NOT NULL,
  filter TEXT NOT NULL DEFAULT '',
  format TEXT NOT NULL DEFAULT 'pcap',
  state TEXT NOT NULL,
  retain INTEGER NOT NULL DEFAULT 0,
  max_bytes INTEGER NOT NULL,
  bytes_written INTEGER NOT NULL DEFAULT 0,
  packets INTEGER NOT NULL DEFAULT 0,
  truncated INTEGER NOT NULL DEFAULT 0,
  artifact_id TEXT,
  created_at TEXT NOT NULL,
  started_at TEXT,
  finished_at TEXT,
  last_error_json TEXT
);

CREATE TABLE IF NOT EXISTS traffic_filters (
  id TEXT PRIMARY KEY,
  laboratory_id TEXT NOT NULL REFERENCES laboratories(id) ON DELETE CASCADE,
  expression TEXT NOT NULL,
  state TEXT NOT NULL,
  max_observations INTEGER NOT NULL,
  observations_json TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  finished_at TEXT,
  last_error_json TEXT
);

CREATE TABLE IF NOT EXISTS network_objects (
  id TEXT PRIMARY KEY,
  laboratory_id TEXT NOT NULL REFERENCES laboratories(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  revision INTEGER NOT NULL DEFAULT 1,
  desired_state TEXT NOT NULL DEFAULT 'active',
  observed_state TEXT NOT NULL DEFAULT 'pending',
  config_json TEXT NOT NULL DEFAULT '{}',
  runtime_owner_json TEXT,
  last_error_json TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(laboratory_id, name)
);

CREATE TABLE IF NOT EXISTS network_attachments (
  id TEXT PRIMARY KEY,
  network_object_id TEXT NOT NULL REFERENCES network_objects(id) ON DELETE CASCADE,
  interface_id TEXT REFERENCES interfaces(id) ON DELETE CASCADE,
  port_name TEXT NOT NULL,
  config_json TEXT NOT NULL DEFAULT '{}',
  UNIQUE(network_object_id, port_name)
);
