package sqlite

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	if attachment.Revision != 1 {
		t.Fatalf("attachment revision=%d want 1", attachment.Revision)
	}
	storedAttachment, err := repositories.GetNetworkAttachment(ctx, attachment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedAttachment.Revision != attachment.Revision {
		t.Fatalf("stored attachment revision=%d want %d", storedAttachment.Revision, attachment.Revision)
	}
	listedAttachments, err := repositories.ListNetworkObjectAttachments(ctx, object.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listedAttachments) != 1 || listedAttachments[0].Revision != attachment.Revision {
		t.Fatalf("listed attachments=%+v", listedAttachments)
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
	if err = repositories.DeleteTopologyNetworkAttachment(ctx, attachment.ID, attachment.Revision+1, "operation-stale-delete"); err == nil {
		t.Fatal("expected stale attachment revision conflict")
	} else if problem, ok := domain.ProblemFromError(err); !ok || problem.Code != "revision_conflict" {
		t.Fatalf("stale delete err=%v", err)
	}
	if _, err = repositories.GetNetworkAttachment(ctx, attachment.ID); err != nil {
		t.Fatalf("stale delete removed attachment: %v", err)
	}
	if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_id=?`, attachment.ID).Scan(&reservationCount); err != nil {
		t.Fatal(err)
	}
	if reservationCount != 2 {
		t.Fatalf("stale delete reservations=%d want 2", reservationCount)
	}
	if err = repositories.DeleteTopologyNetworkAttachment(ctx, attachment.ID, attachment.Revision, "operation-delete"); err != nil {
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

func TestTopologyConnectionTransactionsRollbackAtEveryPersistenceBoundary(t *testing.T) {
	ctx := context.Background()
	newFixture := func(t *testing.T, name string) (*Database, *Repositories, domain.Laboratory, domain.NetworkObject, domain.Interface) {
		t.Helper()
		database, err := Open(ctx, "file:"+name+"?mode=memory&cache=shared")
		if err != nil {
			t.Fatal(err)
		}
		topology := NewTopologyRepository(database)
		laboratory, err := command.NewLaboratoryService(topology).Create(ctx, name, "", domain.RecoveryAutoRestore)
		if err != nil {
			database.Close()
			t.Fatal(err)
		}
		_, interfaces, err := command.NewNodeService(topology).CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "node", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
		if err != nil {
			database.Close()
			t.Fatal(err)
		}
		repositories := NewRepositories(database)
		now := time.Now().UTC()
		object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "switch", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0"}}}, CreatedAt: now, UpdatedAt: now}
		if err = repositories.CreateNetworkObject(ctx, object); err != nil {
			database.Close()
			t.Fatal(err)
		}
		return database, repositories, laboratory, object, interfaces[0]
	}
	assertCounts := func(t *testing.T, database *Database, attachmentID domain.ID, backing, reservations, events int) {
		t.Helper()
		for query, expected := range map[string]int{
			`SELECT COUNT(*) FROM network_attachments WHERE id='` + string(attachmentID) + `'`:                                           backing,
			`SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_id='` + string(attachmentID) + `'`:                       reservations,
			`SELECT COUNT(*) FROM outbox_events WHERE resource_type='network_attachment' AND resource_id='` + string(attachmentID) + `'`: events,
		} {
			var count int
			if err := database.DB.QueryRowContext(ctx, query).Scan(&count); err != nil || count != expected {
				t.Fatalf("query=%s count=%d want=%d err=%v", query, count, expected, err)
			}
		}
	}

	t.Run("task outbox failure", func(t *testing.T) {
		database, repositories, _, _, _ := newFixture(t, "connection-task-rollback")
		defer database.Close()
		if _, err := database.DB.ExecContext(ctx, `CREATE TRIGGER fail_task_outbox BEFORE INSERT ON outbox_events WHEN NEW.event_type='task.created' BEGIN SELECT RAISE(ABORT,'task outbox failure'); END`); err != nil {
			t.Fatal(err)
		}
		operation := command.NewTopologyConnectionOperation(command.TopologyConnectionCreateTaskKind, "connection", "key", "fingerprint", map[string]any{"laboratory_id": "lab"})
		if err := repositories.CreateTask(ctx, operation); err == nil {
			t.Fatal("expected task outbox failure")
		}
		if _, err := repositories.GetTask(ctx, operation.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("task survived failed transaction: %v", err)
		}
	})

	for _, test := range []struct {
		name    string
		trigger string
	}{
		{name: "backing insert failure", trigger: `CREATE TRIGGER fail_attachment_backing BEFORE INSERT ON network_attachments BEGIN SELECT RAISE(ABORT,'backing failure'); END`},
		{name: "outbox failure", trigger: `CREATE TRIGGER fail_attachment_outbox BEFORE INSERT ON outbox_events WHEN NEW.event_type='network_attachment.created' BEGIN SELECT RAISE(ABORT,'outbox failure'); END`},
	} {
		t.Run(test.name, func(t *testing.T) {
			database, repositories, _, object, iface := newFixture(t, "connection-"+strings.ReplaceAll(test.name, " ", "-"))
			defer database.Close()
			if _, err := database.DB.ExecContext(ctx, test.trigger); err != nil {
				t.Fatal(err)
			}
			attachmentID := domain.NewID()
			if _, err := repositories.CreateTopologyNetworkAttachmentAs(ctx, attachmentID, object.ID, iface.ID, "eth0", nil, "operation"); err == nil {
				t.Fatal("expected transaction failure")
			}
			assertCounts(t, database, attachmentID, 0, 0, 0)
		})
	}

	t.Run("delete outbox failure", func(t *testing.T) {
		database, repositories, _, object, iface := newFixture(t, "connection-delete-rollback")
		defer database.Close()
		attachment, err := repositories.CreateTopologyNetworkAttachment(ctx, object.ID, iface.ID, "eth0", nil, "create-operation")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = database.DB.ExecContext(ctx, `CREATE TRIGGER fail_attachment_delete_outbox BEFORE INSERT ON outbox_events WHEN NEW.event_type='network_attachment.deleted' BEGIN SELECT RAISE(ABORT,'delete outbox failure'); END`); err != nil {
			t.Fatal(err)
		}
		if err = repositories.DeleteTopologyNetworkAttachment(ctx, attachment.ID, attachment.Revision, "delete-operation"); err == nil {
			t.Fatal("expected delete transaction failure")
		}
		assertCounts(t, database, attachment.ID, 1, 2, 1)
	})
}

func TestRecoverTopologyConnectionReservationsRepairsMissingAndOrphanedState(t *testing.T) {
	ctx := context.Background()
	database, err := Open(ctx, "file:topology-connection-recovery?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	topology := NewTopologyRepository(database)
	laboratory, err := command.NewLaboratoryService(topology).Create(ctx, "recovery", "", domain.RecoveryAutoRestore)
	if err != nil {
		t.Fatal(err)
	}
	_, interfacesA, err := command.NewNodeService(topology).CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "node-a", Kind: "docker", InterfaceCount: 2, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	_, interfacesB, err := command.NewNodeService(topology).CreateConfigured(ctx, laboratory.ID, command.CreateNodeRequest{Name: "node-b", Kind: "docker", InterfaceCount: 1, InterfaceLimit: 4})
	if err != nil {
		t.Fatal(err)
	}
	repositories := NewRepositories(database)
	now := time.Now().UTC()
	objectA := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "switch-a", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0"}, map[string]any{"name": "eth1"}}}, CreatedAt: now, UpdatedAt: now}
	objectB := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: laboratory.ID, Name: "switch-b", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{"ports": []any{map[string]any{"name": "eth0"}}}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, objectA); err != nil {
		t.Fatal(err)
	}
	if err = repositories.CreateNetworkObject(ctx, objectB); err != nil {
		t.Fatal(err)
	}
	link := domain.Link{ID: domain.NewID(), LaboratoryID: laboratory.ID, EndpointAID: interfacesA[0].ID, EndpointBID: interfacesB[0].ID, Revision: 1, DesiredState: "connected", ObservedState: "connected"}
	if err = topology.CreateLink(ctx, link); err != nil {
		t.Fatal(err)
	}
	attachment, err := repositories.CreateTopologyNetworkAttachment(ctx, objectA.ID, interfacesA[1].ID, "eth0", nil, "attachment-operation")
	if err != nil {
		t.Fatal(err)
	}
	objectLink := domain.NetworkObjectLink{ID: domain.NewID(), LaboratoryID: laboratory.ID, ObjectAID: objectA.ID, PortAName: "eth1", ObjectBID: objectB.ID, PortBName: "eth0", Revision: 1, DesiredState: "connected", ObservedState: "connected"}
	if err = repositories.CreateNetworkObjectLink(ctx, objectLink); err != nil {
		t.Fatal(err)
	}
	if _, err = database.DB.ExecContext(ctx, `DELETE FROM topology_endpoint_reservations WHERE resource_id IN (?,?,?)`, link.ID, attachment.ID, objectLink.ID); err != nil {
		t.Fatal(err)
	}
	failedTaskID := domain.NewID()
	if _, err = database.DB.ExecContext(ctx, `INSERT INTO operation_tasks(id,kind,resource_type,resource_id,state,progress_current,progress_total,input_json,created_at) VALUES(?,'topology_connection.create','topology_connection','missing-backing','failed',1,2,'{}',?)`, failedTaskID, now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err = database.DB.ExecContext(ctx, `INSERT INTO topology_endpoint_reservations(laboratory_id,owner_type,owner_id,port_name,resource_type,resource_id,operation_id,state,created_at) VALUES(?,'node_interface',?,'eth9','link','missing-backing',?,'occupied',?)`, laboratory.ID, interfacesA[0].ID, failedTaskID, now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	outcomes, err := repositories.RecoverTopologyConnectionReservations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 4 {
		t.Fatalf("outcomes=%+v", outcomes)
	}
	for resourceID, expected := range map[domain.ID]int{link.ID: 2, attachment.ID: 2, objectLink.ID: 2, "missing-backing": 0} {
		var count int
		if err = database.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM topology_endpoint_reservations WHERE resource_id=?`, resourceID).Scan(&count); err != nil || count != expected {
			t.Fatalf("resource=%s reservations=%d want=%d err=%v", resourceID, count, expected, err)
		}
	}
	second, err := repositories.RecoverTopologyConnectionReservations(ctx)
	if err != nil || len(second) != 0 {
		t.Fatalf("second recovery outcomes=%+v err=%v", second, err)
	}
}
