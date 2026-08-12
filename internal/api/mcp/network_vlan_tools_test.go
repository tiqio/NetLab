package mcp

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type vlanMCPRuntime struct{}

func (vlanMCPRuntime) Configure(context.Context, domain.NetworkObject) error { return nil }
func (vlanMCPRuntime) Delete(context.Context, domain.ID) error               { return nil }
func (vlanMCPRuntime) InspectNetworkObject(context.Context, domain.NetworkObject) (domain.RuntimeBackingObservation, error) {
	return domain.RuntimeBackingObservation{Kind: domain.RuntimeBackingNamespace, Owned: true, Usable: true}, nil
}
func (vlanMCPRuntime) DiagnosticsObject(context.Context, domain.NetworkObject) (map[string]any, error) {
	return map[string]any{
		"desired":     map[string]any{"ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20}}}},
		"observed":    map[string]any{"ports": []any{map[string]any{"name": "eth0", "pvid": 1, "tagged": []any{}}}, "ipv6_neighbors": []any{map[string]any{"address": "fd10::fe", "state": []any{"FAILED"}}}},
		"path_checks": []any{map[string]any{"destination": "fd20::/64", "gateway": "fd10::fe", "status": "unverified", "stop_at": "neighbor_discovery"}},
		"mismatches":  []string{"eth0: VLAN membership differs"},
	}, nil
}
func (v vlanMCPRuntime) ConfigurationConverged(ctx context.Context, object domain.NetworkObject) (bool, map[string]any, error) {
	diagnostics, err := v.DiagnosticsObject(ctx, object)
	return false, diagnostics, err
}

func TestNetworkVLANMCPUpdateAndDiagnosticsUseSharedServices(t *testing.T) {
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:mcp-vlan?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "mcp vlan", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: lab.ID, Name: "switch", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20}}}}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	service := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{SwitchL2: vlanMCPRuntime{}})
	tools := NetworkTools(service, reconcile.NewNetworkObjectTaskService(service, runner))
	find := func(name string) Tool {
		for _, tool := range tools {
			if tool.Name == name {
				return tool
			}
		}
		t.Fatalf("missing tool %s", name)
		return Tool{}
	}
	contextValue := &gin.Context{Request: httptest.NewRequest("POST", "/mcp", nil)}
	update := find("netlab.network_objects.update")
	_, err = update.Handler(contextValue, map[string]any{
		"object_id": string(object.ID), "name": "switch", "expected_revision": float64(1),
		"config": map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{10}}}},
	})
	if err == nil {
		t.Fatal("expected contradictory VLAN membership to be rejected")
	}
	result, err := update.Handler(contextValue, map[string]any{
		"object_id": string(object.ID), "name": "switch", "expected_revision": float64(1), "idempotency_key": "mcp-vlan-update",
		"config": map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20, 30}}}},
	})
	if err != nil || result.(map[string]any)["task"] == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	diagnostics, err := find("netlab.network_objects.diagnostics").Handler(contextValue, map[string]any{"object_id": string(object.ID)})
	if err != nil {
		t.Fatal(err)
	}
	runtime := diagnostics.(map[string]any)["runtime"].(map[string]any)
	if len(runtime["mismatches"].([]string)) != 1 {
		t.Fatalf("runtime=%+v", runtime)
	}
	checks := runtime["path_checks"].([]any)
	if len(checks) != 1 || checks[0].(map[string]any)["stop_at"] != "neighbor_discovery" {
		t.Fatalf("runtime=%+v", runtime)
	}
}
