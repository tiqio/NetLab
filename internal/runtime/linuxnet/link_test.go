package linuxnet

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type recordingRunner struct {
	commands     []string
	failContains string
	failError    error
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	r.commands = append(r.commands, command)
	if r.failContains != "" && strings.Contains(command, r.failContains) {
		if r.failError != nil {
			return r.failError
		}
		return errors.New("injected")
	}
	return nil
}

func TestDeleteIsIdempotentWhenInterfaceIsMissing(t *testing.T) {
	runner := &recordingRunner{failContains: "link delete dev missing0", failError: errors.New("Cannot find device missing0")}
	runtime, _ := NewLinkRuntime(runner)
	if err := runtime.Delete(context.Background(), "missing0"); err != nil {
		t.Fatal(err)
	}
}

func TestLiveBridgeMoveRollsBackOnFailure(t *testing.T) {
	runner := &recordingRunner{failContains: "master newbr"}
	runtime, _ := NewLinkRuntime(runner)
	if err := runtime.MoveToBridge(context.Background(), "tap0", "oldbr", "newbr"); err == nil {
		t.Fatal("expected failure")
	}
	if got := strings.Join(runner.commands, "\n"); !strings.Contains(got, "nomaster") || !strings.Contains(got, "master oldbr") {
		t.Fatalf("commands:\n%s", got)
	}
}

func TestTapAndVethNamesAreValidated(t *testing.T) {
	runtime, _ := NewLinkRuntime(&recordingRunner{})
	if err := runtime.CreateTap(context.Background(), "tap;bad", "owner"); err == nil {
		t.Fatal("unsafe tap accepted")
	}
	if err := runtime.CreateVeth(context.Background(), "veth0", "peer0", "owner"); err != nil {
		t.Fatal(err)
	}
}
