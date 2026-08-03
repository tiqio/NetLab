package stream

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	consoleRuntime "github.com/netlab/netlab/internal/runtime/console"
)

type consoleNodeReaderStub struct{ node domain.Node }

func (s consoleNodeReaderStub) GetNode(context.Context, domain.ID) (domain.Node, error) {
	return s.node, nil
}

type dockerConsoleStub struct{}

func (dockerConsoleStub) OpenConsole(context.Context, domain.Node) (io.ReadWriteCloser, error) {
	return nil, nil
}

func TestConsoleSessionsExposeRuntimeOwnership(t *testing.T) {
	handler := NewConsoleHandlers("", consoleRuntime.Limits{})
	handler.beginSession("session-1", "node-1", "telnet")
	records, err := handler.DiscoverRuntimeOwnership(context.Background())
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if records[0].ResourceType != "node" || records[0].ResourceID != domain.ID("node-1") || records[0].ObjectKind != "console_proxy" {
		t.Fatalf("record=%+v", records[0])
	}
	handler.endSession("session-1")
	records, err = handler.DiscoverRuntimeOwnership(context.Background())
	if err != nil || len(records) != 0 {
		t.Fatalf("ended session remained discoverable: records=%+v err=%v", records, err)
	}
}

func TestDockerNodesExposeTelnetConsole(t *testing.T) {
	handler := NewConsoleHandlers("", consoleRuntime.Limits{}, consoleNodeReaderStub{node: domain.Node{ID: "node-1", Kind: "docker"}})
	handler.SetDockerConsole(dockerConsoleStub{})
	modes, err := handler.modes(context.Background(), "node-1")
	if err != nil || len(modes) != 1 || modes[0] != "telnet" {
		t.Fatalf("modes=%v err=%v", modes, err)
	}
}

func TestQEMUTelnetAllowsOnlyOneSerialSession(t *testing.T) {
	runtimeDir := t.TempDir()
	nodeID := domain.ID("node-1")
	nodeDir := filepath.Join(runtimeDir, string(nodeID))
	if err := os.MkdirAll(nodeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", filepath.Join(nodeDir, "serial.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- connection
		}
	}()

	handler := NewConsoleHandlers(runtimeDir, consoleRuntime.Limits{MaximumSession: time.Minute}, consoleNodeReaderStub{node: domain.Node{ID: nodeID, Kind: "qemu"}})
	first, created, err := handler.getOrCreateSession(context.Background(), nodeID, "telnet", "terminal-1")
	if err != nil || !created {
		t.Fatalf("first session created=%v err=%v", created, err)
	}
	defer first.Close()
	backend := <-accepted
	defer backend.Close()

	if _, _, err = handler.getOrCreateSession(context.Background(), nodeID, "telnet", "terminal-2"); err == nil {
		t.Fatal("second QEMU serial session was accepted")
	}
	reconnected, created, err := handler.getOrCreateSession(context.Background(), nodeID, "telnet", "terminal-1")
	if err != nil || created || reconnected != first {
		t.Fatalf("existing session created=%v same=%v err=%v", created, reconnected == first, err)
	}
}
