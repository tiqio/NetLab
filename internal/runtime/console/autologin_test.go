package console

import (
	"bytes"
	"io"
	"testing"

	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
)

type scriptedConsole struct {
	reads  [][]byte
	writes bytes.Buffer
}

func (c *scriptedConsole) Read(buffer []byte) (int, error) {
	if len(c.reads) == 0 {
		return 0, io.EOF
	}
	value := c.reads[0]
	c.reads = c.reads[1:]
	return copy(buffer, value), nil
}

func (c *scriptedConsole) Write(buffer []byte) (int, error) { return c.writes.Write(buffer) }
func (c *scriptedConsole) Close() error                     { return nil }

func TestAutoLoginRespondsToLoginAndPasswordPrompts(t *testing.T) {
	backend := &scriptedConsole{reads: [][]byte{[]byte("VyOS login:"), []byte("Password:")}}
	console := WithAutoLogin(backend, qemuRuntime.BootstrapCredentials{Username: "vyos", Password: "secret"})
	buffer := make([]byte, 128)
	_, _ = console.Read(buffer)
	_, _ = console.Read(buffer)
	if got := backend.writes.String(); got != "vyos\rsecret\r" {
		t.Fatalf("automatic input=%q", got)
	}
}
