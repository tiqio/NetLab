ALTER TABLE captures ADD COLUMN purpose TEXT NOT NULL DEFAULT '';
ALTER TABLE captures ADD COLUMN parent_resource_id TEXT;
ALTER TABLE captures ADD COLUMN artifact_url TEXT NOT NULL DEFAULT '';
ALTER TABLE captures ADD COLUMN expires_at TEXT;
ALTER TABLE captures ADD COLUMN completion_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE traffic_filters ADD COLUMN color TEXT NOT NULL DEFAULT '#f59e0b';
ALTER TABLE traffic_filters ADD COLUMN interface_ids_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE traffic_filters ADD COLUMN link_ids_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE traffic_filters ADD COLUMN network_object_link_ids_json TEXT NOT NULL DEFAULT '[]';

CREATE TABLE IF NOT EXISTS traffic_observations (
  traffic_filter_id TEXT NOT NULL REFERENCES traffic_filters(id) ON DELETE CASCADE,
  fingerprint TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  interface_id TEXT,
  link_id TEXT,
  network_object_link_id TEXT,
  direction TEXT NOT NULL,
  source_address TEXT NOT NULL DEFAULT '',
  destination_address TEXT NOT NULL DEFAULT '',
  source_mac TEXT NOT NULL DEFAULT '',
  destination_mac TEXT NOT NULL DEFAULT '',
  packet_role TEXT NOT NULL DEFAULT '',
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  packet_count INTEGER NOT NULL DEFAULT 0,
  byte_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY(traffic_filter_id, fingerprint, resource_type, resource_id, direction)
);
CREATE INDEX IF NOT EXISTS idx_traffic_observations_resource
  ON traffic_observations(resource_type, resource_id, last_seen_at);
CREATE INDEX IF NOT EXISTS idx_captures_source
  ON captures(source_type, source_id, state);
