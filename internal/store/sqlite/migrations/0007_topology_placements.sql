CREATE TABLE topology_placements (
    laboratory_id TEXT NOT NULL REFERENCES laboratories(id) ON DELETE CASCADE,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL CHECK(resource_type IN ('node', 'network_object')),
    x REAL NOT NULL,
    y REAL NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL,
    PRIMARY KEY(laboratory_id, resource_id)
);
CREATE INDEX idx_topology_placements_lab ON topology_placements(laboratory_id, resource_type);
CREATE TRIGGER topology_placement_delete_node AFTER DELETE ON nodes
BEGIN
    DELETE FROM topology_placements WHERE resource_type='node' AND resource_id=OLD.id;
END;
CREATE TRIGGER topology_placement_delete_network_object AFTER DELETE ON network_objects
BEGIN
    DELETE FROM topology_placements WHERE resource_type='network_object' AND resource_id=OLD.id;
END;
