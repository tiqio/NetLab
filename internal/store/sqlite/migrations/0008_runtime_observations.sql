CREATE TABLE IF NOT EXISTS node_runtime_capabilities (
  node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
  capability TEXT NOT NULL,
  revision INTEGER NOT NULL,
  state TEXT NOT NULL,
  required INTEGER NOT NULL DEFAULT 0,
  details_json TEXT NOT NULL DEFAULT '{}',
  problem_json TEXT,
  observed_at TEXT NOT NULL,
  PRIMARY KEY(node_id, capability)
);
CREATE INDEX IF NOT EXISTS idx_node_runtime_capabilities_state ON node_runtime_capabilities(state, observed_at);
CREATE INDEX IF NOT EXISTS idx_network_objects_observed_state ON network_objects(observed_state, updated_at);
CREATE TABLE IF NOT EXISTS network_object_tombstones (
  id TEXT PRIMARY KEY,
  laboratory_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  deleted_at TEXT NOT NULL
);
