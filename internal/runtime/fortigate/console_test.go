package fortigate

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type scriptedConsole struct {
	mu     sync.Mutex
	reads  [][]byte
	writes bytes.Buffer
}

func (c *scriptedConsole) Read(body []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.reads) == 0 {
		return 0, io.EOF
	}
	value := c.reads[0]
	c.reads = c.reads[1:]
	return copy(body, value), nil
}
func (c *scriptedConsole) Write(body []byte) (int, error) { return c.writes.Write(body) }
func (c *scriptedConsole) Close() error                   { return nil }

func TestInteractAuthenticates(t *testing.T) {
	console := &scriptedConsole{reads: [][]byte{[]byte("FortiGate login:"), []byte("Password:"), []byte("FGT #")}}
	result, err := interact(context.Background(), console, domain.NodeCredentialSecret{Username: "admin", Password: []byte("old")}, nil, 0)
	if err != nil || result != "authenticated" {
		t.Fatalf("result=%s err=%v", result, err)
	}
	if got := console.writes.String(); got != "\r\nadmin\r\nold\r\n" {
		t.Fatalf("writes=%q", got)
	}
}

func TestInteractRotatesInitialPassword(t *testing.T) {
	console := &scriptedConsole{reads: [][]byte{[]byte("FortiGate login:"), []byte("Password:"), []byte("You are forced to change your password. Please input a new password:"), []byte("Please confirm new password:"), []byte("FGT #")}}
	staged := domain.NodeCredentialSecret{Password: []byte("new-secret")}
	result, err := interact(context.Background(), console, domain.NodeCredentialSecret{Username: "admin"}, &staged, 0)
	if err != nil || result != "password_rotated" {
		t.Fatalf("result=%s err=%v", result, err)
	}
	if bytes.Count(console.writes.Bytes(), []byte("new-secret")) != 2 {
		t.Fatal("new password was not sent exactly twice")
	}
}

func TestInteractLogsOutExistingSessionBeforeVerification(t *testing.T) {
	console := &scriptedConsole{reads: [][]byte{[]byte("FGT #"), []byte("FortiGate login:"), []byte("Password:"), []byte("FGT #")}}
	result, err := interact(context.Background(), console, domain.NodeCredentialSecret{Username: "admin", Password: []byte("known-secret")}, nil, 0)
	if err != nil || result != "authenticated" {
		t.Fatalf("result=%s err=%v", result, err)
	}
	if got := console.writes.String(); got != "\r\nexit\r\nadmin\r\nknown-secret\r\n" {
		t.Fatalf("writes=%q", got)
	}
}
