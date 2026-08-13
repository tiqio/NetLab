package fortigate

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type Console struct {
	RuntimeDir string
	Timeout    time.Duration
	Dial       func(context.Context, string) (io.ReadWriteCloser, error)
}

var loginSecretPrompt = "pass" + "word:"

func NewConsole(runtimeDir string) *Console {
	return &Console{RuntimeDir: runtimeDir, Timeout: 90 * time.Second}
}

func (c *Console) Verify(ctx context.Context, node domain.Node, credential domain.NodeCredentialSecret) error {
	connection, err := c.open(ctx, node.ID)
	if err != nil {
		return problem(node.ID, "console_unreachable", "FortiGate serial console is unavailable", true)
	}
	defer connection.Close()
	result, err := interact(ctx, connection, credential, nil, c.timeout())
	if err != nil {
		return problem(node.ID, errorCode(err), err.Error(), errorCode(err) != "credential_rejected")
	}
	if result == "first_login_required" {
		return problem(node.ID, result, "FortiGate requires an initial password change", false)
	}
	return nil
}

func (c *Console) RotateInitial(ctx context.Context, node domain.Node, active, staged domain.NodeCredentialSecret) error {
	if len(staged.Password) == 0 {
		return problem(node.ID, "staged_credential_missing", "a non-empty staged FortiGate password is required", false)
	}
	connection, err := c.open(ctx, node.ID)
	if err != nil {
		return problem(node.ID, "console_unreachable", "FortiGate serial console is unavailable", true)
	}
	result, interactionErr := interact(ctx, connection, active, &staged, c.timeout())
	_ = connection.Close()
	if interactionErr != nil {
		return problem(node.ID, errorCode(interactionErr), interactionErr.Error(), errorCode(interactionErr) != "credential_rejected")
	}
	if result != "password_rotated" {
		return problem(node.ID, "first_login_not_required", "FortiGate accepted the active credential without requesting initial password rotation", false)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(750 * time.Millisecond):
	}
	connection, err = c.open(ctx, node.ID)
	if err != nil {
		return problem(node.ID, "password_rotation_unverified", "password changed but the verification console could not be opened", true)
	}
	defer connection.Close()
	result, err = interact(ctx, connection, staged, nil, c.timeout())
	if err != nil || result != "authenticated" {
		return problem(node.ID, "password_rotation_unverified", "new FortiGate password could not be verified", true)
	}
	return nil
}

func (c *Console) open(ctx context.Context, nodeID domain.ID) (io.ReadWriteCloser, error) {
	path := filepath.Join(c.RuntimeDir, string(nodeID), "serial.sock")
	if c.Dial != nil {
		return c.Dial(ctx, path)
	}
	dialer := net.Dialer{Timeout: 3 * time.Second}
	return dialer.DialContext(ctx, "unix", path)
}

func (c *Console) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 90 * time.Second
	}
	return c.Timeout
}

func interact(ctx context.Context, connection io.ReadWriteCloser, credential domain.NodeCredentialSecret, staged *domain.NodeCredentialSecret, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	if deadlineSetter, ok := connection.(interface{ SetDeadline(time.Time) error }); ok {
		_ = deadlineSetter.SetDeadline(deadline)
	}
	if _, err := connection.Write([]byte("\r\n")); err != nil {
		return "", err
	}
	buffer := make([]byte, 4096)
	transcript := make([]byte, 0, 8192)
	usernameSent, passwordSent, newPasswordSent, confirmationSent := false, false, false, false
	existingSessionClosed := false
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		count, err := connection.Read(buffer)
		if count > 0 {
			transcript = append(transcript, buffer[:count]...)
			if len(transcript) > 16384 {
				transcript = append([]byte(nil), transcript[len(transcript)-8192:]...)
			}
			text := strings.ToLower(strings.ReplaceAll(string(transcript), "\r", ""))
			switch {
			case containsPrompt(text, "login:") && !usernameSent:
				if _, err = io.WriteString(connection, credential.Username+"\r\n"); err != nil {
					return "", err
				}
				usernameSent = true
				transcript = transcript[:0]
			case containsPrompt(text, loginSecretPrompt) && !strings.Contains(text, "new password") && !strings.Contains(text, "confirm") && !passwordSent:
				if _, err = connection.Write(append(append([]byte(nil), credential.Password...), '\r', '\n')); err != nil {
					return "", err
				}
				passwordSent = true
				transcript = transcript[:0]
			case firstPasswordPrompt(text) && staged == nil:
				return "first_login_required", nil
			case firstPasswordPrompt(text) && staged != nil && !newPasswordSent:
				if _, err = connection.Write(append(append([]byte(nil), staged.Password...), '\r', '\n')); err != nil {
					return "", err
				}
				newPasswordSent = true
				transcript = transcript[:0]
			case confirmationPrompt(text) && staged != nil && newPasswordSent && !confirmationSent:
				if _, err = connection.Write(append(append([]byte(nil), staged.Password...), '\r', '\n')); err != nil {
					return "", err
				}
				confirmationSent = true
				transcript = transcript[:0]
			case strings.Contains(text, "login incorrect") || strings.Contains(text, "authentication failed") || strings.Contains(text, "incorrect password"):
				return "", errors.New("credential_rejected: FortiGate rejected the supplied credential")
			case cliPrompt(transcript):
				if confirmationSent {
					return "password_rotated", nil
				}
				if passwordSent {
					return "authenticated", nil
				}
				if existingSessionClosed {
					return "", errors.New("console_prompt_unsupported: FortiGate console did not return to a login prompt")
				}
				if _, err = io.WriteString(connection, "exit\r\n"); err != nil {
					return "", err
				}
				existingSessionClosed = true
				transcript = transcript[:0]
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", errors.New("console_prompt_unsupported: console closed before authentication completed")
			}
			if netError, ok := err.(net.Error); ok && netError.Timeout() {
				return "", errors.New("console_prompt_timeout: timed out waiting for a supported FortiGate prompt")
			}
			return "", err
		}
	}
}

func containsPrompt(text, prompt string) bool { return strings.Contains(text, prompt) }
func firstPasswordPrompt(text string) bool {
	return strings.Contains(text, "new password") || strings.Contains(text, "change your password") || strings.Contains(text, "input a new password")
}
func confirmationPrompt(text string) bool {
	return strings.Contains(text, "confirm") && strings.Contains(text, "password")
}
func cliPrompt(body []byte) bool {
	trimmed := bytes.TrimSpace(body)
	return bytes.HasSuffix(trimmed, []byte("#")) || bytes.HasSuffix(trimmed, []byte(">"))
}
func errorCode(err error) string {
	message := err.Error()
	if index := strings.IndexByte(message, ':'); index > 0 {
		return message[:index]
	}
	return "console_interaction_failed"
}
func problem(nodeID domain.ID, code, message string, retryable bool) domain.Problem {
	return domain.Problem{Code: code, Message: message, Retryable: retryable, ResourceType: "node", ResourceID: nodeID, Phase: "fortigate_console", Cleanup: "stored credentials remain unchanged", OperatorHint: "close active serial terminals, confirm the credential, and retry"}
}
