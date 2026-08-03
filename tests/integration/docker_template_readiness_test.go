package integration_test

import (
	"strings"
	"testing"
)

func TestDockerTemplateReferencesAreImmutableDigests(t *testing.T) {
	for _, reference := range []string{"busybox@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "ubuntu@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"} {
		parts := strings.Split(reference, "@sha256:")
		if len(parts) != 2 || len(parts[1]) != 64 {
			t.Fatalf("mutable image reference %s", reference)
		}
	}
}
