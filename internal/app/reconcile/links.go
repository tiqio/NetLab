package reconcile

import (
	"context"
	"fmt"

	"github.com/netlab/netlab/internal/domain"
)

type BridgeMover interface {
	MoveToBridge(context.Context, string, string, string) error
}

type LiveLinkReconciler struct{ runtime BridgeMover }

func NewLiveLinkReconciler(runtime BridgeMover) *LiveLinkReconciler {
	return &LiveLinkReconciler{runtime: runtime}
}

func (r *LiveLinkReconciler) Rewire(ctx context.Context, iface domain.Interface, hostInterface, oldBridge, newBridge string) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("interface", iface.ID, "live_rewire"))
	if iface.ID == "" || hostInterface == "" || newBridge == "" {
		return fmt.Errorf("interface and target bridge are required")
	}
	return r.runtime.MoveToBridge(ctx, hostInterface, oldBridge, newBridge)
}
