package stream

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
)

type networkObjectConsoleReaderStub struct{ object domain.NetworkObject }

func (s networkObjectConsoleReaderStub) GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error) {
	return s.object, nil
}

type networkObjectConsoleRuntimeStub struct{ peer io.Closer }

func (s *networkObjectConsoleRuntimeStub) OpenConsole(domain.NetworkObject) (io.ReadWriteCloser, error) {
	left, right := net.Pipe()
	s.peer = right
	return left, nil
}

func TestNetworkObjectConsoleTracksPCSessionOwnership(t *testing.T) {
	runtime := &networkObjectConsoleRuntimeStub{}
	handler := NewNetworkObjectConsoleHandlers(
		networkObjectConsoleReaderStub{object: domain.NetworkObject{ID: "pc-1", Kind: domain.NetworkPC, ObservedState: "active"}},
		runtime,
		consoleRuntime.Limits{MaximumSession: time.Minute},
	)
	session, created, err := handler.getOrCreate(domain.NetworkObject{ID: "pc-1", Kind: domain.NetworkPC, ObservedState: "active"}, "session-1")
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	records, err := handler.DiscoverRuntimeOwnership(context.Background())
	if err != nil || len(records) != 1 || records[0].ResourceType != "network_object" || records[0].ResourceID != "pc-1" {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	session.Close()
	if runtime.peer != nil {
		_ = runtime.peer.Close()
	}
}

func TestNetworkObjectConsoleRejectsNonPCObject(t *testing.T) {
	handler := NewNetworkObjectConsoleHandlers(
		networkObjectConsoleReaderStub{object: domain.NetworkObject{ID: "bridge-1", Kind: domain.NetworkBridge, ObservedState: "active"}},
		&networkObjectConsoleRuntimeStub{},
		consoleRuntime.Limits{},
	)
	if _, err := handler.pc(context.Background(), "bridge-1"); err == nil {
		t.Fatal("bridge unexpectedly exposed a PC console")
	}
}
