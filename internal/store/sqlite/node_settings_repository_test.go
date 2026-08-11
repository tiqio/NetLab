package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func TestUpdateNodeSettingsRequiresStoppedAndUpdatesAtomically(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:node-settings?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "node-settings", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	node, _, err := command.NewNodeService(repository).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{
		Name: "old-name", Kind: "pc", CPUCount: 1, CPUQuotaMicros: 100000, MemoryMiB: 64,
		StorageGiB: 1, InterfaceLimit: 4, ProcessLimit: 8, InterfaceCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := repository.UpdateNodeSettings(ctx, node.ID, node.Revision, domain.NodeSettings{Name: " new-name ", CPUCount: 2, CPUQuotaMicros: 0, MemoryMiB: 128, InterfaceLimit: 6, ProcessLimit: 32})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "new-name" || updated.CPUCount != 2 || updated.CPUQuotaMicros != 0 || updated.MemoryMiB != 128 || updated.InterfaceLimit != 6 || updated.ProcessLimit != 32 || updated.Revision != node.Revision.Next() {
		t.Fatalf("updated=%+v", updated)
	}
	running, err := repository.SetNodeDesiredState(ctx, updated.ID, updated.Revision, domain.DesiredRunning)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.UpdateNodeSettings(ctx, running.ID, running.Revision, domain.NodeSettings{Name: "blocked", CPUCount: 1, CPUQuotaMicros: 100000, MemoryMiB: 64, InterfaceLimit: 4, ProcessLimit: 8})
	var problem domain.Problem
	if !errors.As(err, &problem) || problem.Code != "node_not_stopped" {
		t.Fatalf("expected node_not_stopped, got %v", err)
	}
}

func TestUpdateNodeSettingsRejectsDuplicateNameAndLowInterfaceLimit(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:node-settings-validation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "node-settings-validation", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	service := command.NewNodeService(repository)
	node, _, err := service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node-a", Kind: "pc", CPUCount: 1, CPUQuotaMicros: 100000, MemoryMiB: 64, StorageGiB: 1, InterfaceLimit: 4, ProcessLimit: 8, InterfaceCount: 2})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node-b", Kind: "pc", CPUCount: 1, CPUQuotaMicros: 100000, MemoryMiB: 64, StorageGiB: 1, InterfaceLimit: 4, ProcessLimit: 8, InterfaceCount: 1}); err != nil {
		t.Fatal(err)
	}
	_, err = repository.UpdateNodeSettings(ctx, node.ID, node.Revision, domain.NodeSettings{Name: "node-b", CPUCount: 1, CPUQuotaMicros: 100000, MemoryMiB: 64, InterfaceLimit: 4, ProcessLimit: 8})
	var problem domain.Problem
	if !errors.As(err, &problem) || problem.Code != "node_name_conflict" {
		t.Fatalf("expected node_name_conflict, got %v", err)
	}
	_, err = repository.UpdateNodeSettings(ctx, node.ID, node.Revision, domain.NodeSettings{Name: "node-a", CPUCount: 1, CPUQuotaMicros: 100000, MemoryMiB: 64, InterfaceLimit: 1, ProcessLimit: 8})
	if !errors.As(err, &problem) || problem.Code != "invalid_node_settings" {
		t.Fatalf("expected invalid_node_settings, got %v", err)
	}
}

func TestUpdateNodeSettingsPersistsInterfaceDriverAndNetworkConfig(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:node-network-settings?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, _ := command.NewLaboratoryService(repository).Create(ctx, "node-network-settings", "", domain.RecoveryRemainStopped)
	node, interfaces, err := command.NewNodeService(repository).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "ubuntu", Kind: "qemu", CPUCount: 1, MemoryMiB: 512, StorageGiB: 8, InterfaceLimit: 4, ProcessLimit: 32, InterfaceCount: 1, Config: map[string]any{"network_interfaces": []any{map[string]any{"name": "eth0", "modes": []any{}, "addresses": []any{}}}}})
	if err != nil {
		t.Fatal(err)
	}
	settings := domain.NodeSettings{Name: node.Name, CPUCount: 1, MemoryMiB: 512, InterfaceLimit: 4, ProcessLimit: 32, NetworkInterfaces: []domain.NodeNetworkInterfaceSettings{{ID: interfaces[0].ID, Name: interfaces[0].Name, Driver: "e1000", Modes: []string{"static"}, Addresses: []string{"192.0.2.10/24"}, Routes: []domain.RouteConfig{{Destination: "198.51.100.99/24", Gateway: "192.0.2.1"}}}}}
	updated, err := repository.UpdateNodeSettings(ctx, node.ID, node.Revision, settings)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repository.GetInterface(ctx, interfaces[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Driver != "e1000" {
		t.Fatalf("driver=%q", stored.Driver)
	}
	values, _ := updated.Config["network_interfaces"].([]any)
	if len(values) != 1 {
		t.Fatalf("network_interfaces=%#v", updated.Config["network_interfaces"])
	}
	body, _ := json.Marshal(values)
	if !strings.Contains(string(body), `"destination":"198.51.100.0/24"`) {
		t.Fatalf("network_interfaces=%s", body)
	}
}

func TestUpdateNodeSettingsPersistsDockerForwardingFlags(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:"+string(domain.NewID())+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "docker-forwarding", "", domain.RecoveryRemainStopped)
	if err != nil {
		t.Fatal(err)
	}
	node, _, err := command.NewNodeService(repository).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "router", Kind: string(domain.RuntimeDocker), CPUCount: 1, MemoryMiB: 128, InterfaceLimit: 4, ProcessLimit: 32, InterfaceCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	enabled, disabled := true, false
	updated, err := repository.UpdateNodeSettings(ctx, node.ID, node.Revision, domain.NodeSettings{Name: node.Name, CPUCount: 1, MemoryMiB: 128, InterfaceLimit: 4, ProcessLimit: 32, ForwardIPv4: &enabled, ForwardIPv6: &disabled})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Config["forward_ipv4"] != true || updated.Config["forward_ipv6"] != false || updated.Revision != node.Revision+1 {
		t.Fatalf("updated=%+v", updated)
	}
}
