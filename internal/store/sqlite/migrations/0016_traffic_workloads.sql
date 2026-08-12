CREATE TABLE traffic_workloads (
 id TEXT PRIMARY KEY,
 laboratory_id TEXT NOT NULL REFERENCES laboratories(id) ON DELETE CASCADE,
 name TEXT NOT NULL,
 revision INTEGER NOT NULL,
 source_json BLOB NOT NULL,
 protocol TEXT NOT NULL,
 address_family TEXT NOT NULL,
 destination_json BLOB NOT NULL,
 interval_seconds INTEGER NOT NULL,
 timeout_seconds INTEGER NOT NULL,
 desired_state TEXT NOT NULL,
 observed_state TEXT NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0,
 successes INTEGER NOT NULL DEFAULT 0,
 failures INTEGER NOT NULL DEFAULT 0,
 matched_bytes INTEGER NOT NULL DEFAULT 0,
 last_success_at TEXT,
 last_error_json BLOB,
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE INDEX traffic_workloads_lab_idx ON traffic_workloads(laboratory_id, created_at);
