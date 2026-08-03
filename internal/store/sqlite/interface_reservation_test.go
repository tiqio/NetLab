package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func newInterfaceReservationNode(t *testing.T, interfaceCount, limit int) (*Database, *TopologyRepository, domain.Node, []domain.Interface) {
	t.Helper()
	ctx := context.Background()
	database, err := Open(ctx, filepath.Join(t.TempDir(), "netlab.db"))
	if err != nil {
		t.Fatal(err)
	}
	repository := NewTopologyRepository(database)
	lab, err := command.NewLaboratoryService(repository).Create(ctx, "reservation", "", domain.RecoveryAutoRestore)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	node, interfaces, err := command.NewNodeService(repository).CreateConfigured(ctx, lab.ID, command.CreateNodeRequest{Name: "qemu", Kind: "qemu", InterfaceCount: interfaceCount, InterfaceLimit: limit})
	if err != nil {
		database.Close()
		t.Fatal(err)
	}
	return database, repository, node, interfaces
}

func reservationCandidate(nodeID domain.ID, suffix string) domain.Interface {
	return domain.Interface{ID: domain.ID("iface-" + suffix), NodeID: nodeID, Driver: "virtio-net-pci", MACAddress: "02:00:00:00:00:" + suffix, OperationalState: "pending", Revision: 1}
}

func TestReserveInterfaceReusesLowestFreeSlot(t *testing.T) {
	database, repository, node, interfaces := newInterfaceReservationNode(t, 2, 64)
	defer database.Close()
	ctx := context.Background()
	if err := repository.AddInterface(ctx, domain.Interface{ID: "iface-high", NodeID: node.ID, Slot: 63, Name: "eth63", Driver: "virtio-net-pci", MACAddress: "02:00:00:00:00:63", OperationalState: "down", Revision: 1}); err != nil {
		t.Fatal(err)
	}
	if err := repository.DeleteInterface(ctx, interfaces[0].ID, interfaces[0].Revision); err != nil {
		t.Fatal(err)
	}
	reserved, err := repository.ReserveInterface(ctx, reservationCandidate(node.ID, "64"), 64)
	if err != nil {
		t.Fatal(err)
	}
	if reserved.Slot != 0 || reserved.Name != "eth0" {
		t.Fatalf("reserved=%+v", reserved)
	}
}

func TestReserveInterfaceRejectsSlot64Boundary(t *testing.T) {
	database, repository, node, _ := newInterfaceReservationNode(t, 1, 64)
	defer database.Close()
	ctx := context.Background()
	for slot := 1; slot < 64; slot++ {
		value := domain.Interface{ID: domain.ID(fmt.Sprintf("iface-%02d", slot)), NodeID: node.ID, Slot: slot, Name: fmt.Sprintf("eth%d", slot), Driver: "virtio-net-pci", MACAddress: fmt.Sprintf("02:00:00:00:01:%02x", slot), OperationalState: "down", Revision: 1}
		if err := repository.AddInterface(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repository.ReserveInterface(ctx, reservationCandidate(node.ID, "65"), 64); err == nil {
		t.Fatal("expected capacity rejection")
	} else if problem, ok := domain.ProblemFromError(err); !ok || problem.Code != "resource_exhausted" {
		t.Fatalf("err=%v", err)
	}
	var count, maximum int
	if err := database.DB.QueryRowContext(ctx, `SELECT COUNT(*),MAX(slot) FROM interfaces WHERE node_id=?`, node.ID).Scan(&count, &maximum); err != nil {
		t.Fatal(err)
	}
	if count != 64 || maximum != 63 {
		t.Fatalf("count=%d maximum=%d", count, maximum)
	}
}

func TestReserveInterfaceSerializesConcurrentRequests(t *testing.T) {
	database, repository, node, _ := newInterfaceReservationNode(t, 1, 2)
	defer database.Close()
	ctx := context.Background()
	var wait sync.WaitGroup
	errorsChannel := make(chan error, 2)
	for _, suffix := range []string{"a1", "a2"} {
		suffix := suffix
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := repository.ReserveInterface(ctx, reservationCandidate(node.ID, suffix), 2)
			errorsChannel <- err
		}()
	}
	wait.Wait()
	close(errorsChannel)
	succeeded, exhausted := 0, 0
	for err := range errorsChannel {
		if err == nil {
			succeeded++
			continue
		}
		if problem, ok := domain.ProblemFromError(err); ok && problem.Code == "resource_exhausted" {
			exhausted++
			continue
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if succeeded != 1 || exhausted != 1 {
		t.Fatalf("succeeded=%d exhausted=%d", succeeded, exhausted)
	}
}
