package ownership

import (
	"strings"
	"testing"
)

func TestValidationScopeSeparatesNamesAndMarkers(t *testing.T) {
	t.Setenv("NETLAB_OWNERSHIP_DOMAIN", "acceptance-run-1")
	if value := Name("nlpc", "object-1", 15); strings.HasPrefix(value, "nlpc-") || len(value) > 15 {
		t.Fatalf("scoped name=%q", value)
	}
	if value := Marker("netlab", "object-1"); strings.HasPrefix(value, "netlab:") {
		t.Fatalf("scoped marker=%q", value)
	}
}
