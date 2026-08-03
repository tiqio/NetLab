package reconcile

import (
	"errors"
	"fmt"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestStructuredProblemPreservesRuntimeDetailsAndAddsReconcileContext(t *testing.T) {
	problem := structuredProblem(fmt.Errorf("reconcile: %w", &domain.Problem{Code: "runtime_unavailable", Message: "Docker unavailable", Retryable: true}), domain.Problem{ResourceType: "node", ResourceID: "node", Phase: "starting", Cleanup: "no runtime created", OperatorHint: "start Docker", RetryAfterSeconds: 3})
	if problem.Code != "runtime_unavailable" || problem.ResourceID != "node" || problem.Phase != "starting" || problem.RetryAfterSeconds != 3 {
		t.Fatalf("problem=%+v", problem)
	}
}

func TestTerminalProblemMatrixNormalizesEveryReconcileArea(t *testing.T) {
	areas := []struct {
		resourceType string
		resourceID   domain.ID
		phase        string
	}{
		{"node", "node-1", "starting"},
		{"laboratory", "lab-1", "topology_mutation"},
		{"network_object", "network-1", "provisioning"},
		{"capture", "capture-1", "capture_stop"},
		{"laboratory", "lab-2", "laboratory_deletion"},
		{"reconciler", "ownership-discovery", "ownership_discovery"},
		{"reconciler", "data-plane", "data_plane_reconcile"},
		{"interface", "interface-1", "live_rewire"},
		{"host", "service_restart", "recovery"},
	}
	for _, area := range areas {
		t.Run(area.resourceType+"/"+area.phase, func(t *testing.T) {
			err := error(fmt.Errorf("wrapped runtime error: %w", errors.New("injected")))
			normalizeTerminalError(&err, terminalProblem(area.resourceType, area.resourceID, area.phase))
			problem, ok := domain.ProblemFromError(err)
			if !ok || problem.Code != "operation_failed" || problem.ResourceType != area.resourceType || problem.ResourceID != area.resourceID || problem.Phase != area.phase || !problem.Retryable || problem.Cleanup == "" || problem.OperatorHint == "" || problem.RetryAfterSeconds != 3 {
				t.Fatalf("problem=%+v ok=%v", problem, ok)
			}
		})
	}
}

func TestTerminalProblemPreservesWrappedTaskAndRuntimeIdentity(t *testing.T) {
	err := error(fmt.Errorf("outer: %w", &domain.Problem{Code: "runtime_unavailable", Message: "QMP unavailable", TaskID: "task-1", ResourceType: "node", ResourceID: "node-1", Phase: "qmp_ready", Cleanup: "process stopped", OperatorHint: "inspect QMP logs", Retryable: true, RetryAfterSeconds: 8}))
	normalizeTerminalError(&err, terminalProblem("reconciler", "nodes", "node_reconcile"))
	problem, ok := domain.ProblemFromError(err)
	if !ok || problem.Code != "runtime_unavailable" || problem.TaskID != "task-1" || problem.ResourceID != "node-1" || problem.Phase != "qmp_ready" || problem.RetryAfterSeconds != 8 {
		t.Fatalf("problem=%+v ok=%v", problem, ok)
	}
}
