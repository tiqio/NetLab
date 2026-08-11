package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type staleNamespaceExecutor struct {
	commands  []string
	ready     bool
	listed    bool
	deleteErr error
}

func (e *staleNamespaceExecutor) Run(_ context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	e.commands = append(e.commands, command)
	if strings.Contains(command, "netns delete") && e.deleteErr != nil {
		return e.deleteErr
	}
	if strings.Contains(command, "netns add") {
		e.ready = true
	}
	return nil
}

func TestEnsureNamespaceCleansInvalidNamedReferenceBeforeRecreate(t *testing.T) {
	executor := &staleNamespaceExecutor{listed: true, deleteErr: errors.New("Invalid argument")}
	original := staleNamespaceCleanup
	cleaned := ""
	staleNamespaceCleanup = func(namespace string) error {
		cleaned = namespace
		return nil
	}
	t.Cleanup(func() { staleNamespaceCleanup = original })
	if err := ensureNamespace(context.Background(), executor, "ip", "nl-stale"); err != nil {
		t.Fatal(err)
	}
	if cleaned != "nl-stale" {
		t.Fatalf("cleaned=%q", cleaned)
	}
}

func TestDeleteNamespaceDoesNotCleanUsableNamespaceWhenDeleteFails(t *testing.T) {
	executor := &staleNamespaceExecutor{ready: true, listed: true, deleteErr: errors.New("permission denied")}
	original := staleNamespaceCleanup
	called := false
	staleNamespaceCleanup = func(string) error { called = true; return nil }
	t.Cleanup(func() { staleNamespaceCleanup = original })
	if err := deleteNamespace(context.Background(), executor, "ip", "nl-live"); err == nil {
		t.Fatal("expected delete failure")
	}
	if called {
		t.Fatal("usable namespace must not be force-cleaned")
	}
}

func (e *staleNamespaceExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	e.commands = append(e.commands, name+" "+strings.Join(args, " "))
	if len(args) >= 2 && args[0] == "netns" && args[1] == "list" {
		if e.listed {
			return []byte("nl-live\nnl-stale"), nil
		}
		return nil, nil
	}
	if !e.ready {
		return nil, errors.New("invalid namespace reference")
	}
	return []byte("lo"), nil
}

func TestEnsureNamespaceReplacesStaleReference(t *testing.T) {
	executor := &staleNamespaceExecutor{listed: true}
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
