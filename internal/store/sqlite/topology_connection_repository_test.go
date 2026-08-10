package sqlite

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func TestTopologyEndpointReservationsSerializeLinksAndAttachments(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:topology-connection-reservations?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "connections", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	_, interfacesA, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node-a", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	_, interfacesB, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node-b", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	now := time.Now().UTC()
	object := domain.NetworkObject{ID: "switch", LaboratoryID: lab.ID, Name: "switch", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0"}}}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	attachment, err := repositories.CreateTopologyNetworkAttachment(ctx, object.ID, interfacesA[0].ID, "eth0", nil, "operation-attach")
	if err != nil {
		t.Fatal(err)
	}
	link := domain.Link{ID: "blocked-link", LaboratoryID: lab.ID, EndpointAID: interfacesA[0].ID, EndpointBID: interfacesB[0].ID, Revision: 1, DesiredState: "connected", ObservedState: "pending"}
	err = topology.CreateLink(ctx, link)
	if problem, ok := domain.ProblemFromError(err); !ok || problem.Code != "port_in_use" {
		t.Fatalf("err=%v", err)
	}
	var reservationCount int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_id=?`, attachment.ID).Scan(&reservationCount); err != nil {
		t.Fatal(err)
	}
	if reservationCount != 2 {
		t.Fatalf("reservations=%d want 2", reservationCount)
	}
	if err = repositories.DeleteTopologyNetworkAttachment(ctx, attachment.ID, "operation-delete"); err != nil {
		t.Fatal(err)
	}
	if err = topology.CreateLink(ctx, link); err != nil {
		t.Fatalf("create link after release: %v", err)
	}
}

func TestTopologyEndpointReservationsAllowSingleConcurrentWinnerAndReleaseOnDelete(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:topology-connection-concurrency?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "concurrency", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	_, interfacesA, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node-a", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	_, interfacesB, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node-b", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		id  domain.ID
		err error
	}
	results := make(chan result, 10)
	var waitGroup sync.WaitGroup
	for index := range 10 {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			id := domain.ID(fmt.Sprintf("contended-link-%d", index))
			err := topology.CreateLink(ctx, domain.Link{ID: id, LaboratoryID: lab.ID, EndpointAID: interfacesA[0].ID, EndpointBID: interfacesB[0].ID, Revision: 1, DesiredState: "connected", ObservedState: "pending"})
			results <- result{id: id, err: err}
		}(index)
	}
	waitGroup.Wait()
	close(results)

	var winner domain.ID
	for value := range results {
		if value.err == nil {
			if winner != "" {
				t.Fatalf("multiple winners: %s and %s", winner, value.id)
			}
			winner = value.id
			continue
		}
		problem, ok := domain.ProblemFromError(value.err)
		if !ok || problem.Code != "port_in_use" {
			t.Fatalf("loser %s returned %v", value.id, value.err)
		}
	}
	if winner == "" {
		t.Fatal("expected one successful connection")
	}
	var reservations int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_type='link' AND resource_id=?`, winner).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if reservations != 2 {
		t.Fatalf("winner reservations=%d want 2", reservations)
	}
	if err = topology.DeleteLink(ctx, winner); err != nil {
		t.Fatal(err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_type='link' AND resource_id=?`, winner).Scan(&reservations); err != nil {
		t.Fatal(err)
	}
	if reservations != 0 {
		t.Fatalf("deleted connection leaked %d reservations", reservations)
	}
}

func TestTopologyEndpointReservationRollbackDoesNotLeakFirstEndpoint(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:topology-connection-rollback?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(topology).Create(ctx, "rollback", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	_, interfaces, err := command.NewNodeService(topology).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "node", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	err = topology.CreateLink(ctx, domain.Link{ID: "invalid", LaboratoryID: lab.ID, EndpointAID: interfaces[0].ID, EndpointBID: "missing", Revision: 1, DesiredState: "connected", ObservedState: "pending"})
	if err == nil {
		t.Fatal("expected missing endpoint failure")
	}
	var count int
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE owner_type='node_interface' AND owner_id=?`, interfaces[0].ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("leaked reservations=%d", count)
	}
}
