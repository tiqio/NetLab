package linuxnet

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type pcConsole struct {
	file    *os.File
	command *exec.Cmd
	once    sync.Once
}

func (r *PCRuntime) OpenConsole(object domain.NetworkObject) (io.ReadWriteCloser, error) {
	if object.Kind != domain.NetworkPC {
		return nil, fmt.Errorf("console is only available for PC network objects")
	}
	if object.ObservedState != "active" {
		return nil, fmt.Errorf("PC network object is not active")
	}
	namespace := ownershipNameForPC(object.ID)
	command := exec.Command(r.ip, "netns", "exec", namespace, "/bin/bash", "--noprofile", "--norc", "-i")
	command.Env = append(os.Environ(), "TERM=xterm-256color", "PS1=pc:\\w# ")
	file, err := pty.Start(command)
	if err != nil {
		return nil, fmt.Errorf("start PC console: %w", err)
	}
	console := &pcConsole{file: file, command: command}
	go func() {
		_ = command.Wait()
		_ = console.Close()
	}()
	return console, nil
}

func ownershipNameForPC(id domain.ID) string {
	return ownership.Name("nlpc", id, 15)
}

func (c *pcConsole) Read(buffer []byte) (int, error)  { return c.file.Read(buffer) }
func (c *pcConsole) Write(buffer []byte) (int, error) { return c.file.Write(buffer) }

func (c *pcConsole) Close() error {
	var err error
	c.once.Do(func() {
		err = c.file.Close()
		if c.command.Process != nil {
			_ = c.command.Process.Kill()
		}
	})
	return err
}
