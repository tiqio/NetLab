package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

func TestNetworkPathExportImportPreservesForwardingVLANRolesAndWorkloads(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:network-path-export?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := storesqlite.NewTopologyRepository(database)
	repositories := storesqlite.NewRepositories(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: "source", Name: "source", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	node := domain.Node{ID: "router", LaboratoryID: lab.ID, Name: "router", Kind: "docker", Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped, Config: map[string]any{"forward_ipv4": true, "forward_ipv6": true, "device_roles": []any{map[string]any{"interface_id": "eth0", "role": "wan"}}}, CreatedAt: now, UpdatedAt: now}
	iface := domain.Interface{ID: "router-if0", NodeID: node.ID, Slot: 0, Name: "eth0", MACAddress: "02:00:00:00:00:01", OperationalState: "down", Revision: 1}
	object := domain.NetworkObject{ID: "switch", LaboratoryID: lab.ID, Name: "switch", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20}}}}, CreatedAt: now, UpdatedAt: now}
	if err = topology.ImportTopology(ctx, lab, []domain.Node{node}, []domain.Interface{iface}, nil, []domain.NetworkObject{object}, nil, []domain.TopologyPlacement{{LaboratoryID: lab.ID, ResourceID: node.ID, ResourceType: domain.PlacementNode, X: 100, Y: 100, Revision: 1}, {LaboratoryID: lab.ID, ResourceID: object.ID, ResourceType: domain.PlacementNetworkObject, X: 300, Y: 100, Revision: 1}}); err != nil {
		t.Fatal(err)
	}
	workload := domain.TrafficWorkload{ID: "workload", LaboratoryID: lab.ID, Name: "ping", Revision: 1, Source: domain.TrafficWorkloadEndpoint{Kind: "node", ResourceID: node.ID, InterfaceID: iface.ID}, Protocol: "icmp", AddressFamily: "ipv4", Destination: domain.TrafficWorkloadDestination{Address: "192.0.2.1"}, IntervalSeconds: 5, TimeoutSeconds: 2, DesiredState: "running", ObservedState: "running", CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateTrafficWorkload(ctx, workload); err != nil {
		t.Fatal(err)
	}
	bundle, err := command.NewExportService(topology, nil).Build(ctx, lab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.TrafficWorkloads) != 1 {
		t.Fatalf("workloads=%+v", bundle.TrafficWorkloads)
	}
	bundle.Laboratory.Name = "imported"
	imported, err := command.NewImportService(topology, nil).ImportAs(ctx, "imported", bundle)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := topology.Snapshot(ctx, imported.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.TrafficWorkloads) != 1 || snapshot.TrafficWorkloads[0].Source.ResourceID == node.ID || snapshot.TrafficWorkloads[0].Source.InterfaceID == iface.ID {
		t.Fatalf("workload remap failed: %+v", snapshot.TrafficWorkloads)
	}
	if snapshot.Nodes[0].Config["forward_ipv4"] != true || len(snapshot.NetworkObjects) == 0 {
		t.Fatalf("configuration missing: %+v %+v", snapshot.Nodes, snapshot.NetworkObjects)
	}
	configBody, _ := json.Marshal(map[string]any{"node": snapshot.Nodes[0].Config, "switch": snapshot.NetworkObjects[0].Config})
	for _, expected := range []string{`"forward_ipv6":true`, `"role":"wan"`, `"pvid":10`, `"tagged":[20]`} {
		if !strings.Contains(string(configBody), expected) {
			t.Fatalf("missing %s in %s", expected, configBody)
		}
	}
	bundle.Laboratory.Name = "duplicate"
	duplicated, err := command.NewImportService(topology, nil).ImportAs(ctx, "duplicate", bundle)
	if err != nil || duplicated.ID != "duplicate" {
		t.Fatalf("duplicate=%+v err=%v", duplicated, err)
	}
}
