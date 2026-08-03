CREATE TABLE IF NOT EXISTS network_object_links (
  id TEXT PRIMARY KEY,
  laboratory_id TEXT NOT NULL REFERENCES laboratories(id) ON DELETE CASCADE,
  object_a_id TEXT NOT NULL REFERENCES network_objects(id) ON DELETE CASCADE,
  port_a_name TEXT NOT NULL,
  object_b_id TEXT NOT NULL REFERENCES network_objects(id) ON DELETE CASCADE,
  port_b_name TEXT NOT NULL,
  revision INTEGER NOT NULL DEFAULT 1,
  desired_state TEXT NOT NULL DEFAULT 'connected',
  observed_state TEXT NOT NULL DEFAULT 'pending',
  last_error_json BLOB,
  CHECK(object_a_id <> object_b_id),
  UNIQUE(object_a_id, port_a_name),
  UNIQUE(object_b_id, port_b_name)
);

CREATE INDEX IF NOT EXISTS idx_network_object_links_lab ON network_object_links(laboratory_id);
CREATE INDEX IF NOT EXISTS idx_network_object_links_object_a ON network_object_links(object_a_id);
CREATE INDEX IF NOT EXISTS idx_network_object_links_object_b ON network_object_links(object_b_id);
