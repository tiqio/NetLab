package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type staleNamespaceExecutor struct {
	commands []string
	ready    bool
}

func (e *staleNamespaceExecutor) Run(_ context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	if strings.Contains(command, "netns add") {
		e.ready = true
	}
	return nil
}

func (e *staleNamespaceExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	e.commands = append(e.commands, name+" "+strings.Join(args, " "))
	if !e.ready {
		return nil, errors.New("invalid namespace reference")
	}
	return []byte("lo"), nil
}

func TestEnsureNamespaceReplacesStaleReference(t *testing.T) {
	executor := &staleNamespaceExecutor{}
	if err := ensureNamespace(context.Background(), executor, "ip", "nl-stale"); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if !strings.Contains(commands, "ip netns delete nl-stale") || !strings.Contains(commands, "ip netns add nl-stale") {
		t.Fatalf("stale namespace was not replaced:\n%s", commands)
	}
}

func TestDeleteNamespaceIsIdempotentWhenAlreadyAbsent(t *testing.T) {
	executor := &staleNamespaceExecutor{ready: true}
	if err := deleteNamespace(context.Background(), executor, "ip", "nl-missing"); err != nil {
		t.Fatal(err)
	}
	commands := strings.Join(executor.commands, "\n")
	if strings.Contains(commands, "netns delete nl-missing") {
		t.Fatalf("already absent namespace was deleted again:\n%s", commands)
	}
}
