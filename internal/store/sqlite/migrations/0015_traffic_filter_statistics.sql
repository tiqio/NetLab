ALTER TABLE traffic_filters ADD COLUMN fingerprint_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE traffic_filters ADD COLUMN matched_packets INTEGER NOT NULL DEFAULT 0;
ALTER TABLE traffic_filters ADD COLUMN matched_bytes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE traffic_filters ADD COLUMN first_match_at TEXT;
ALTER TABLE traffic_filters ADD COLUMN last_match_at TEXT;
