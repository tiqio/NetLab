ALTER TABLE topology_endpoint_reservations ADD COLUMN operation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE topology_endpoint_reservations ADD COLUMN state TEXT NOT NULL DEFAULT 'occupied';
ALTER TABLE topology_endpoint_reservations ADD COLUMN created_at TEXT NOT NULL DEFAULT '';

INSERT OR IGNORE INTO topology_endpoint_reservations(
  laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id,operation_id,state,created_at
)
SELECT l.laboratory_id,'node_interface',i.id,i.name,'link',l.id,'','occupied',''
FROM links l
JOIN interfaces i ON i.id=l.endpoint_a_id OR i.id=l.endpoint_b_id;

INSERT OR IGNORE INTO topology_endpoint_reservations(
  laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id,operation_id,state,created_at
)
SELECT o.laboratory_id,'node_interface',i.id,i.name,'network_attachment',a.id,'','occupied',''
FROM network_attachments a
JOIN interfaces i ON i.id=a.interface_id
JOIN network_objects o ON o.id=a.network_object_id
WHERE a.interface_id IS NOT NULL AND a.interface_id <> '';
