package linuxnet

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"golang.org/x/sys/unix"
)

var namespaceNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
var staleNamespaceCleanup = cleanupStaleNamespacePath

func namespaceReady(ctx context.Context, executor CommandExecutor, ip, namespace string) bool {
	_, err := executor.Output(ctx, ip, "-n", namespace, "link", "show", "lo")
	return err == nil
}

func inspectNamespaceBacking(ctx context.Context, executor CommandExecutor, ip, namespace string) domain.RuntimeBackingObservation {
	observation := domain.RuntimeBackingObservation{Kind: domain.RuntimeBackingNamespace, RuntimeName: namespace, Owned: true, Adoptable: true, Recreatable: true, ObservedAt: time.Now().UTC()}
	observation.Usable = namespaceReady(ctx, executor, ip, namespace)
	if !observation.Usable {
		problem := domain.Problem{Code: "runtime_backing_unusable", Message: "network namespace backing is not usable", Retryable: true, Phase: "runtime_inspection", Cleanup: "owned namespace will be recreated during reconciliation", OperatorHint: "retry reconciliation and inspect host namespace support", Details: map[string]any{"runtime_name": namespace}}
		observation.Problem = &problem
	}
	return observation
}

func ensureNamespace(ctx context.Context, executor CommandExecutor, ip, namespace string) error {
	if namespaceReady(ctx, executor, ip, namespace) {
		return nil
	}
	if err := deleteNamespace(ctx, executor, ip, namespace); err != nil {
		return err
	}
	return executor.Run(ctx, ip, "netns", "add", namespace)
}

func deleteNamespace(ctx context.Context, executor CommandExecutor, ip, namespace string) error {
	if body, err := executor.Output(ctx, ip, "netns", "list"); err == nil {
		found := false
		for _, line := range strings.Split(string(body), "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == namespace {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	err := executor.Run(ctx, ip, "netns", "delete", namespace)
	if err == nil {
		return nil
	}
	if namespaceReady(ctx, executor, ip, namespace) {
		return err
	}
	if cleanupErr := staleNamespaceCleanup(namespace); cleanupErr != nil {
		return fmt.Errorf("delete invalid namespace reference: %v; cleanup stale reference: %w", err, cleanupErr)
	}
	return nil
}

func cleanupStaleNamespacePath(namespace string) error {
	if !namespaceNamePattern.MatchString(namespace) || filepath.Base(namespace) != namespace {
		return fmt.Errorf("invalid namespace name")
	}
	path := filepath.Join("/run/netns", namespace)
	if err := unix.Unmount(path, unix.MNT_DETACH); err != nil && !errors.Is(err, unix.EINVAL) && !errors.Is(err, unix.ENOENT) {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
