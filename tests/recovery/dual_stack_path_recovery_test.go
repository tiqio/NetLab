package recovery_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
	"github.com/netlab/netlab/tests/testsupport"
)

func TestDualStackForwardingAndReturnRoutesPersistAcrossServiceRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "dual-stack.db")
	database, err := storesqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	repository := storesqlite.NewTopologyRepository(database)
	laboratory, err := command.NewLaboratoryService(repository).Create(ctx, "dual-stack-recovery", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	fixture := testsupport.ComponentMatrixDualStackFixture()
	node, _, err := command.NewNodeService(repository).CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "service-router", Kind: string(domain.RuntimeDocker), InterfaceCount: 2, Config: fixture.DockerRouter.Config})
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := storesqlite.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	recovered, err := storesqlite.NewTopologyRepository(restarted).GetNode(ctx, node.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Config["forward_ipv4"] != true || recovered.Config["forward_ipv6"] != true {
		t.Fatalf("forwarding=%+v", recovered.Config)
	}
	interfaces, ok := recovered.Config["network_interfaces"].([]any)
	if !ok || len(interfaces) != 2 {
		t.Fatalf("network_interfaces=%#v", recovered.Config["network_interfaces"])
	}
}
