package qemu

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeGuest struct{ polls int }

func (f *fakeGuest) Run(command string, _ any) (json.RawMessage, error) {
	if command == "guest-exec" {
		return json.RawMessage(`{"return":{"pid":42}}`), nil
	}
	f.polls++
	if f.polls == 1 {
		return json.RawMessage(`{"return":{"exited":false}}`), nil
	}
	out := base64.StdEncoding.EncodeToString([]byte("0123456789"))
	return json.RawMessage(`{"return":{"exited":true,"exitcode":7,"out-data":"` + out + `","err-data":""}}`), nil
}

func TestGuestExecPollingDecodingAndOutputLimit(t *testing.T) {
	result, err := ExecuteGuest(context.Background(), &fakeGuest{}, GuestExecRequest{Argv: []string{"/bin/echo", "hello"}, Timeout: time.Second, OutputLimit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 || string(result.Stdout) != "01234" || !result.Truncated {
		t.Fatalf("result=%+v", result)
	}
}

func TestGuestExecTimeout(t *testing.T) {
	runner := &neverExitGuest{}
	_, err := ExecuteGuest(context.Background(), runner, GuestExecRequest{Argv: []string{"sleep"}, Timeout: 20 * time.Millisecond})
	if err == nil {
		t.Fatal("expected timeout")
	}
}

type blockingGuest struct{}

func (blockingGuest) Run(string, any) (json.RawMessage, error) { select {} }
func (blockingGuest) RunContext(ctx context.Context, _ string, _ any) (json.RawMessage, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestGuestExecCancelsBlockedAgentRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ExecuteGuest(ctx, blockingGuest{}, GuestExecRequest{Argv: []string{"true"}, Timeout: time.Minute})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

type neverExitGuest struct{}

func (*neverExitGuest) Run(command string, _ any) (json.RawMessage, error) {
	if command == "guest-exec" {
		return json.RawMessage(`{"return":{"pid":1}}`), nil
	}
	return json.RawMessage(`{"return":{"exited":false}}`), nil
}
