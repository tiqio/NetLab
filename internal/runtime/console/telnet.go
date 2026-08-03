package console

import (
	"context"
	"fmt"
	"net"
	"sync"
)

type TelnetDescriptor struct {
	Mode        string `json:"mode"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	StreamURL   string `json:"stream_url"`
	IdleSeconds int    `json:"idle_seconds"`
}

type TelnetManager struct {
	limits    Limits
	mu        sync.Mutex
	listeners map[string]net.Listener
}

func NewTelnetManager(limits Limits) *TelnetManager {
	return &TelnetManager{limits: limits, listeners: map[string]net.Listener{}}
}

func (m *TelnetManager) Open(ctx context.Context, nodeID, socketPath string) (TelnetDescriptor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing := m.listeners[nodeID]; existing != nil {
		address := existing.Addr().(*net.TCPAddr)
		return descriptor(nodeID, address.Port, m.limits), nil
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return TelnetDescriptor{}, err
	}
	m.listeners[nodeID] = listener
	go m.serve(ctx, listener, socketPath)
	return descriptor(nodeID, listener.Addr().(*net.TCPAddr).Port, m.limits), nil
}

func (m *TelnetManager) serve(ctx context.Context, listener net.Listener, socketPath string) {
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		client, err := listener.Accept()
		if err != nil {
			return
		}
		backend, err := net.Dial("unix", socketPath)
		if err != nil {
			_ = client.Close()
			continue
		}
		go func() { _ = Bridge(ctx, client, backend, m.limits) }()
	}
}

func (m *TelnetManager) Close(nodeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	listener := m.listeners[nodeID]
	delete(m.listeners, nodeID)
	if listener == nil {
		return nil
	}
	return listener.Close()
}

func descriptor(nodeID string, port int, limits Limits) TelnetDescriptor {
	return TelnetDescriptor{Mode: "telnet", Host: "127.0.0.1", Port: port, StreamURL: fmt.Sprintf("/api/v1/nodes/%s/consoles/telnet/stream", nodeID), IdleSeconds: int(limits.IdleTimeout.Seconds())}
}
