package mcp

import (
	"testing"

	"github.com/netlab/netlab/internal/app/reconcile"
)

func TestNetworkRecoveryMCPToolsAreRegistered(t *testing.T) {
	tools := NetworkTools(reconcile.NewNetworkObjectService(nil, reconcile.NetworkRuntimeDispatch{}), nil)
	want := []string{"netlab.network_objects.diagnostics", "netlab.network_objects.reconcile", "netlab.network_object_links.reconcile"}
	for index, name := range want {
		if tools[index].Name != name {
			t.Fatalf("tool[%d]=%q want %q", index, tools[index].Name, name)
		}
	}
}
