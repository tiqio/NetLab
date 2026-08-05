package console

import (
	"io"
	"strings"
	"sync"

	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
)

type autoLoginConsole struct {
	io.ReadWriteCloser
	credentials qemuRuntime.BootstrapCredentials
	mu          sync.Mutex
	buffer      string
	stage       int
}

func WithAutoLogin(connection io.ReadWriteCloser, credentials qemuRuntime.BootstrapCredentials) io.ReadWriteCloser {
	if connection == nil || strings.TrimSpace(credentials.Username) == "" {
		return connection
	}
	return &autoLoginConsole{ReadWriteCloser: connection, credentials: credentials}
}

func (c *autoLoginConsole) Read(buffer []byte) (int, error) {
	count, err := c.ReadWriteCloser.Read(buffer)
	if count > 0 {
		c.observe(buffer[:count])
	}
	return count, err
}

func (c *autoLoginConsole) observe(buffer []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stage >= 2 {
		return
	}
	c.buffer += strings.ToLower(string(buffer))
	if len(c.buffer) > 512 {
		c.buffer = c.buffer[len(c.buffer)-512:]
	}
	if c.stage == 0 && (strings.Contains(c.buffer, "login:") || strings.Contains(c.buffer, "username:")) {
		_, _ = io.WriteString(c.ReadWriteCloser, c.credentials.Username+"\r")
		c.stage = 1
		c.buffer = ""
		return
	}
	if c.stage == 1 && strings.Contains(c.buffer, "password:") {
		_, _ = io.WriteString(c.ReadWriteCloser, c.credentials.Password+"\r")
		c.stage = 2
		c.buffer = ""
	}
}
