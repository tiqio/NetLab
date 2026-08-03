CREATE TABLE IF NOT EXISTS topology_endpoint_reservations (
  laboratory_id TEXT NOT NULL REFERENCES laboratories(id) ON DELETE CASCADE,
  owner_type TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  port_name TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  PRIMARY KEY(laboratory_id, owner_type, owner_id, port_name)
);

CREATE INDEX IF NOT EXISTS idx_topology_endpoint_resource
  ON topology_endpoint_reservations(resource_type, resource_id);

INSERT OR IGNORE INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id)
SELECT o.laboratory_id,'network_object',a.network_object_id,a.port_name,'network_attachment',a.id
FROM network_attachments a JOIN network_objects o ON o.id=a.network_object_id
WHERE a.port_name <> '';

INSERT OR IGNORE INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id)
SELECT laboratory_id,'network_object',object_a_id,port_a_name,'network_object_link',id FROM network_object_links
;

INSERT OR IGNORE INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id)
SELECT laboratory_id,'network_object',object_b_id,port_b_name,'network_object_link',id FROM network_object_links
;
