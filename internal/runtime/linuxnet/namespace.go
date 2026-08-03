package linuxnet

import (
	"context"
	"strings"
)

func namespaceReady(ctx context.Context, executor CommandExecutor, ip, namespace string) bool {
	_, err := executor.Output(ctx, ip, "-n", namespace, "link", "show", "lo")
	return err == nil
}

func ensureNamespace(ctx context.Context, executor CommandExecutor, ip, namespace string) error {
	if namespaceReady(ctx, executor, ip, namespace) {
		return nil
	}
	_ = deleteNamespace(ctx, executor, ip, namespace)
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
	return executor.Run(ctx, ip, "netns", "delete", namespace)
}
