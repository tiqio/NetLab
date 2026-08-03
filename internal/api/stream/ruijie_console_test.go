package stream

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type ruijieSessionStub struct {
	mu       sync.Mutex
	writes   []string
	updates  chan []byte
	done     chan struct{}
	respond  func(string) string
	writeErr error
}

func newRuijieSessionStub(respond func(string) string) *ruijieSessionStub {
	return &ruijieSessionStub{updates: make(chan []byte, 32), done: make(chan struct{}), respond: respond}
}

func (s *ruijieSessionStub) Write(value []byte) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	command := string(value)
	s.mu.Lock()
	s.writes = append(s.writes, command)
	s.mu.Unlock()
	if s.respond != nil {
		if output := s.respond(command); output != "" {
			s.updates <- []byte(output)
		}
	}
	return nil
}

func (s *ruijieSessionStub) Subscribe(int) (<-chan []byte, func()) {
	return s.updates, func() {}
}

func (s *ruijieSessionStub) Done() <-chan struct{} { return s.done }

func TestExecuteRuijieCommandsWaitsForPromptAndVerifiesEachCommand(t *testing.T) {
	session := newRuijieSessionStub(func(command string) string {
		switch strings.TrimSpace(command) {
		case "":
			return "\r\nSwitch>"
		case "\x1a", "enable", "end":
			return command + "\r\nSwitch#"
		default:
			return command + "\r\nSwitch(config)#"
		}
	})
	commands := []string{"enable", "configure terminal", "vlan 10", "end"}
	transcript, err := executeRuijieCommandsWithTimings(context.Background(), session, commands, ruijieCommandTimings{
		readyTimeout:   100 * time.Millisecond,
		readyPulse:     5 * time.Millisecond,
		commandTimeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(transcript, "Switch(config)#") {
		t.Fatalf("transcript=%q", transcript)
	}
	session.mu.Lock()
	writes := append([]string(nil), session.writes...)
	session.mu.Unlock()
	expected := []string{"\r\n", "\x1a\r\n", "enable\r\n", "configure terminal\r\n", "vlan 10\r\n", "end\r\n"}
	if strings.Join(writes, "|") != strings.Join(expected, "|") {
		t.Fatalf("writes=%q expected=%q", writes, expected)
	}
}

func TestExecuteRuijieCommandsReportsCLIStartupTimeout(t *testing.T) {
	session := newRuijieSessionStub(nil)
	_, err := executeRuijieCommandsWithTimings(context.Background(), session, []string{"enable"}, ruijieCommandTimings{
		readyTimeout:   30 * time.Millisecond,
		readyPulse:     5 * time.Millisecond,
		commandTimeout: 30 * time.Millisecond,
	})
	var executionErr *ruijieExecutionError
	if err == nil || !asRuijieExecutionError(err, &executionErr) {
		t.Fatalf("err=%v", err)
	}
	if executionErr.code != "console_not_ready" || !executionErr.retryable {
		t.Fatalf("execution error=%+v", executionErr)
	}
	if len(session.writes) < 2 {
		t.Fatalf("expected repeated readiness probes, writes=%q", session.writes)
	}
}

func TestExecuteRuijieCommandsReportsRejectedCommand(t *testing.T) {
	session := newRuijieSessionStub(func(command string) string {
		trimmed := strings.TrimSpace(command)
		if trimmed == "" {
			return "Switch#"
		}
		if trimmed == "bad command" {
			return "bad command\r\n% Invalid input detected\r\nSwitch#"
		}
		return command + "\r\nSwitch#"
	})
	_, err := executeRuijieCommandsWithTimings(context.Background(), session, []string{"bad command"}, ruijieCommandTimings{
		readyTimeout:   100 * time.Millisecond,
		readyPulse:     5 * time.Millisecond,
		commandTimeout: 100 * time.Millisecond,
	})
	var executionErr *ruijieExecutionError
	if err == nil || !asRuijieExecutionError(err, &executionErr) {
		t.Fatalf("err=%v", err)
	}
	if executionErr.code != "console_command_rejected" {
		t.Fatalf("execution error=%+v", executionErr)
	}
}

func asRuijieExecutionError(err error, target **ruijieExecutionError) bool {
	value, ok := err.(*ruijieExecutionError)
	if ok {
		*target = value
	}
	return ok
}
