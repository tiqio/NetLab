package qemu

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type GuestAgent struct {
	connection net.Conn
	reader     *bufio.Reader
	mu         sync.Mutex
}

func ConnectGuestAgent(path string, timeout time.Duration) (*GuestAgent, error) {
	connection, err := net.DialTimeout("unix", path, timeout)
	if err != nil {
		return nil, err
	}
	return &GuestAgent{connection: connection, reader: bufio.NewReader(connection)}, nil
}

func (g *GuestAgent) Close() error { return g.connection.Close() }

func (g *GuestAgent) Run(command string, arguments any) (json.RawMessage, error) {
	return g.RunContext(context.Background(), command, arguments)
}

func (g *GuestAgent) RunContext(ctx context.Context, command string, arguments any) (json.RawMessage, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if deadline, ok := ctx.Deadline(); ok {
		_ = g.connection.SetDeadline(deadline)
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = g.connection.SetDeadline(time.Now())
		case <-done:
		}
	}()
	defer func() { close(done); _ = g.connection.SetDeadline(time.Time{}) }()
	request, err := json.Marshal(map[string]any{"execute": command, "arguments": arguments})
	if err != nil {
		return nil, err
	}
	if _, err = g.connection.Write(append(request, '\n')); err != nil {
		return nil, err
	}
	line, err := g.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Return json.RawMessage `json:"return"`
		Error  any             `json:"error"`
	}
	if err = json.Unmarshal(line, &envelope); err != nil {
		return nil, err
	}
	if envelope.Error != nil {
		return nil, fmt.Errorf("guest agent error: %v", envelope.Error)
	}
	return json.Marshal(map[string]json.RawMessage{"return": envelope.Return})
}

type GuestCommandRunner interface {
	Run(string, any) (json.RawMessage, error)
}

type ContextGuestCommandRunner interface {
	RunContext(context.Context, string, any) (json.RawMessage, error)
}

type GuestExecRequest struct {
	Argv        []string
	Timeout     time.Duration
	OutputLimit int
}

type GuestExecResult struct {
	ExitCode  int    `json:"exit_code"`
	Stdout    []byte `json:"stdout"`
	Stderr    []byte `json:"stderr"`
	Truncated bool   `json:"truncated"`
}

func ExecuteGuest(ctx context.Context, runner GuestCommandRunner, request GuestExecRequest) (GuestExecResult, error) {
	if len(request.Argv) == 0 || request.Argv[0] == "" {
		return GuestExecResult{}, fmt.Errorf("argv is required")
	}
	if request.Timeout <= 0 {
		request.Timeout = 30 * time.Second
	}
	if request.OutputLimit <= 0 || request.OutputLimit > 16<<20 {
		request.OutputLimit = 1 << 20
	}
	ctx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	arguments := map[string]any{"path": request.Argv[0], "capture-output": true}
	if len(request.Argv) > 1 {
		arguments["arg"] = request.Argv[1:]
	}
	body, err := runGuest(ctx, runner, "guest-exec", arguments)
	if err != nil {
		return GuestExecResult{}, err
	}
	var started struct {
		Return struct {
			PID int `json:"pid"`
		} `json:"return"`
	}
	if err = json.Unmarshal(body, &started); err != nil || started.Return.PID <= 0 {
		return GuestExecResult{}, fmt.Errorf("invalid guest-exec response")
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return GuestExecResult{}, ctx.Err()
		case <-ticker.C:
			body, err = runGuest(ctx, runner, "guest-exec-status", map[string]any{"pid": started.Return.PID})
			if err != nil {
				return GuestExecResult{}, err
			}
			var status struct {
				Return struct {
					Exited   bool   `json:"exited"`
					ExitCode int    `json:"exitcode"`
					OutData  string `json:"out-data"`
					ErrData  string `json:"err-data"`
				} `json:"return"`
			}
			if err = json.Unmarshal(body, &status); err != nil {
				return GuestExecResult{}, err
			}
			if !status.Return.Exited {
				continue
			}
			stdout, _ := base64.StdEncoding.DecodeString(status.Return.OutData)
			stderr, _ := base64.StdEncoding.DecodeString(status.Return.ErrData)
			result := GuestExecResult{ExitCode: status.Return.ExitCode}
			result.Stdout, result.Truncated = bounded(stdout, request.OutputLimit)
			remaining := request.OutputLimit - len(result.Stdout)
			var stderrTruncated bool
			result.Stderr, stderrTruncated = bounded(stderr, max(remaining, 0))
			result.Truncated = result.Truncated || stderrTruncated
			return result, nil
		}
	}
}

func runGuest(ctx context.Context, runner GuestCommandRunner, command string, arguments any) (json.RawMessage, error) {
	if contextual, ok := runner.(ContextGuestCommandRunner); ok {
		return contextual.RunContext(ctx, command, arguments)
	}
	return runner.Run(command, arguments)
}

func bounded(value []byte, limit int) ([]byte, bool) {
	if len(value) <= limit {
		return value, false
	}
	return append([]byte(nil), value[:limit]...), true
}
