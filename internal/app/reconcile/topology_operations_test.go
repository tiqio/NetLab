package reconcile

import (
	"context"
	"errors"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type topologyOperationsDataPlaneFake struct {
	steps       []string
	deleteError error
	ensureError error
}

func (f *topologyOperationsDataPlaneFake) DeleteLink(context.Context, domain.ID) error {
	f.steps = append(f.steps, "delete")
	return f.deleteError
}

func (f *topologyOperationsDataPlaneFake) EnsureLink(context.Context, domain.Link, domain.Interface, domain.Interface) error {
	f.steps = append(f.steps, "ensure")
	return f.ensureError
}

func TestTopologyOperationsReplacesLinkWithoutLeavingOldAttachments(t *testing.T) {
	runtime := &topologyOperationsDataPlaneFake{}
	operations := NewTopologyOperations(runtime)
	if err := operations.EnsureLink(context.Background(), domain.Link{ID: "link"}, domain.Interface{ID: "a"}, domain.Interface{ID: "b"}); err != nil {
		t.Fatal(err)
	}
	if len(runtime.steps) != 2 || runtime.steps[0] != "delete" || runtime.steps[1] != "ensure" {
		t.Fatalf("steps=%v", runtime.steps)
	}
}

func TestTopologyOperationsStopsWhenDeleteFails(t *testing.T) {
	runtime := &topologyOperationsDataPlaneFake{deleteError: errors.New("delete failed")}
	err := NewTopologyOperations(runtime).EnsureLink(context.Background(), domain.Link{ID: "link"}, domain.Interface{ID: "a"}, domain.Interface{ID: "b"})
	if err == nil || len(runtime.steps) != 1 || runtime.steps[0] != "delete" {
		t.Fatalf("steps=%v err=%v", runtime.steps, err)
	}
}
