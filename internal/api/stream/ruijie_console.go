package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const ruijieTranscriptLimit = 64 << 10

type ruijieSession interface {
	Write([]byte) error
	Subscribe(int) (<-chan []byte, func())
	Done() <-chan struct{}
}

type ruijieCommandTimings struct {
	readyTimeout   time.Duration
	readyPulse     time.Duration
	commandTimeout time.Duration
}

type ruijieExecutionError struct {
	code       string
	message    string
	retryable  bool
	transcript string
}

func (e *ruijieExecutionError) Error() string { return e.message }

var (
	ruijiePromptPattern = regexp.MustCompile(`(?m)(?:^|[\r\n])[^\r\n]{0,128}[>#][ \t]*$`)
	ruijieErrorPattern  = regexp.MustCompile(`(?im)(%\s*(?:invalid input|incomplete command|ambiguous command|unknown command|error)|command not found|syntax error)`)
	ruijieANSIPattern   = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
)

func executeRuijieCommands(ctx context.Context, session ruijieSession, commands []string) (string, error) {
	return executeRuijieCommandsWithTimings(ctx, session, commands, ruijieCommandTimings{
		readyTimeout:   60 * time.Second,
		readyPulse:     time.Second,
		commandTimeout: 12 * time.Second,
	})
}

func executeRuijieCommandsWithTimings(ctx context.Context, session ruijieSession, commands []string, timings ruijieCommandTimings) (string, error) {
	updates, cancel := session.Subscribe(512)
	defer cancel()

	readyOutput, err := waitForRuijieReady(ctx, session, updates, timings)
	if err != nil {
		return readyOutput, err
	}
	transcript := readyOutput

	resetOutput, err := runRuijieCommand(ctx, session, updates, "\x1a", timings.commandTimeout)
	transcript = appendRuijieTranscript(transcript, resetOutput)
	if err != nil {
		return transcript, err
	}
	for _, command := range commands {
		output, commandErr := runRuijieCommand(ctx, session, updates, command, timings.commandTimeout)
		transcript = appendRuijieTranscript(transcript, output)
		if commandErr != nil {
			return transcript, commandErr
		}
	}
	return transcript, nil
}

func waitForRuijieReady(ctx context.Context, session ruijieSession, updates <-chan []byte, timings ruijieCommandTimings) (string, error) {
	deadline := time.NewTimer(timings.readyTimeout)
	defer deadline.Stop()
	pulse := time.NewTicker(timings.readyPulse)
	defer pulse.Stop()
	transcript := ""
	if err := session.Write([]byte("\r\n")); err != nil {
		return transcript, ruijieWriteError(err)
	}
	for {
		select {
		case value := <-updates:
			transcript = appendRuijieTranscript(transcript, string(value))
			if ruijiePromptPattern.MatchString(cleanRuijieOutput(transcript)) {
				return transcript, nil
			}
		case <-pulse.C:
			if err := session.Write([]byte("\r\n")); err != nil {
				return transcript, ruijieWriteError(err)
			}
		case <-deadline.C:
			return transcript, &ruijieExecutionError{code: "console_not_ready", message: "Ruijie CLI did not become ready before the timeout; wait for the startup prompt and retry", retryable: true, transcript: transcript}
		case <-session.Done():
			return transcript, &ruijieExecutionError{code: "console_unavailable", message: "Ruijie console closed while waiting for the CLI prompt", retryable: true, transcript: transcript}
		case <-ctx.Done():
			return transcript, &ruijieExecutionError{code: "console_canceled", message: ctx.Err().Error(), retryable: errors.Is(ctx.Err(), context.DeadlineExceeded), transcript: transcript}
		}
	}
}

func runRuijieCommand(ctx context.Context, session ruijieSession, updates <-chan []byte, command string, timeout time.Duration) (string, error) {
	if err := session.Write([]byte(command + "\r\n")); err != nil {
		return "", ruijieWriteError(err)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	transcript := ""
	for {
		select {
		case value := <-updates:
			transcript = appendRuijieTranscript(transcript, string(value))
			cleaned := cleanRuijieOutput(transcript)
			if !ruijiePromptPattern.MatchString(cleaned) {
				continue
			}
			if match := ruijieErrorPattern.FindString(cleaned); match != "" {
				return transcript, &ruijieExecutionError{code: "console_command_rejected", message: fmt.Sprintf("Ruijie rejected command %q: %s", command, strings.TrimSpace(match)), transcript: transcript}
			}
			return transcript, nil
		case <-timer.C:
			return transcript, &ruijieExecutionError{code: "console_command_timeout", message: fmt.Sprintf("Ruijie did not return a prompt after command %q", command), retryable: true, transcript: transcript}
		case <-session.Done():
			return transcript, &ruijieExecutionError{code: "console_unavailable", message: "Ruijie console closed while applying configuration", retryable: true, transcript: transcript}
		case <-ctx.Done():
			return transcript, &ruijieExecutionError{code: "console_canceled", message: ctx.Err().Error(), retryable: errors.Is(ctx.Err(), context.DeadlineExceeded), transcript: transcript}
		}
	}
}

func ruijieWriteError(err error) error {
	if errors.Is(err, io.ErrClosedPipe) {
		return &ruijieExecutionError{code: "console_unavailable", message: "Ruijie console is closed", retryable: true}
	}
	return &ruijieExecutionError{code: "console_write_failed", message: err.Error(), retryable: true}
}

func cleanRuijieOutput(value string) string {
	value = ruijieANSIPattern.ReplaceAllString(value, "")
	return strings.ReplaceAll(value, "\x00", "")
}

func appendRuijieTranscript(current, value string) string {
	current += value
	if len(current) > ruijieTranscriptLimit {
		current = current[len(current)-ruijieTranscriptLimit:]
	}
	return current
}
