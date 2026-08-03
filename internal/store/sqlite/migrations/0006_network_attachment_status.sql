ALTER TABLE network_attachments ADD COLUMN observed_state TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE network_attachments ADD COLUMN last_error_json TEXT;
