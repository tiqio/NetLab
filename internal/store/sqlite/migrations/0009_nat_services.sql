CREATE TABLE IF NOT EXISTS nat_service_observations (
  network_object_id TEXT PRIMARY KEY REFERENCES network_objects(id) ON DELETE CASCADE,
  config_digest TEXT NOT NULL,
  unit_name TEXT NOT NULL,
  config_path TEXT NOT NULL,
  lease_path TEXT NOT NULL,
  pid INTEGER,
  state TEXT NOT NULL,
  problem_json TEXT,
  observed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_nat_service_state ON nat_service_observations(state, observed_at);
