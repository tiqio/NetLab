package linuxnet

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func lookup(name string) (string, error) { return exec.LookPath(name) }

type CommandExecutor interface {
	Run(context.Context, string, ...string) error
	Output(context.Context, string, ...string) ([]byte, error)
}

type SystemExecutor struct{}

func (SystemExecutor) Run(ctx context.Context, name string, args ...string) error {
	body, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(body)), err)
	}
	return nil
}

func (SystemExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	body, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.TrimSpace(string(body)), err)
	}
	return body, nil
}
