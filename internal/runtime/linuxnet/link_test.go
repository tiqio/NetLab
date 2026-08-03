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
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	command := name + " " + strings.Join(args, " ")
	r.commands = append(r.commands, command)
	if r.failContains != "" && strings.Contains(command, r.failContains) {
		return errors.New("injected")
	}
	return nil
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
