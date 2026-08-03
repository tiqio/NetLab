package reconcile

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type recoveryTaskStoreFake struct {
	created   []domain.OperationTask
	updated   []domain.OperationTask
	recovered []domain.ID
}

func (s *recoveryTaskStoreFake) PublishNetworkObjectLinkRecovered(_ context.Context, id, _ domain.ID) error {
	s.recovered = append(s.recovered, id)
	return nil
}

func (s *recoveryTaskStoreFake) CreateTask(_ context.Context, task domain.OperationTask) error {
	s.created = append(s.created, task)
	return nil
}

func (s *recoveryTaskStoreFake) UpdateTask(_ context.Context, task domain.OperationTask) error {
	s.updated = append(s.updated, task)
	return nil
}

type recoveryParticipantFake struct {
	name string
	err  error
	runs int
}

type checkpointParticipantFake struct{ recoveryParticipantFake }

func (p *checkpointParticipantFake) ReconcileWithCheckpoints(_ context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	p.runs++
	if err := checkpoint(RecoveryResourceOutcome{ResourceType: "node", ResourceID: "node-1", State: "recovered", RuntimeID: "pid-42", Details: map[string]string{"kind": "qemu"}}); err != nil {
		return err
	}
	return p.err
}

type objectLinkCheckpointParticipant struct{ recoveryParticipantFake }

func (p *objectLinkCheckpointParticipant) ReconcileWithCheckpoints(_ context.Context, checkpoint func(RecoveryResourceOutcome) error) error {
	p.runs++
	return checkpoint(RecoveryResourceOutcome{ResourceType: "network_object_link", ResourceID: "object-link-1", State: "recovered"})
}

func (p *recoveryParticipantFake) Name() string { return p.name }
func (p *recoveryParticipantFake) Reconcile(context.Context) error {
	p.runs++
	return p.err
}

func TestRecoveryCoordinatorPublishesProgressAndCompletion(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	first := &recoveryParticipantFake{name: "nodes"}
	second := &recoveryParticipantFake{name: "links"}
	prepared := false
	task, err := NewRecoveryCoordinator(store, first, second).Execute(context.Background(), "host_restart", func(context.Context) error {
		prepared = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !prepared || first.runs != 1 || second.runs != 1 {
		t.Fatalf("prepared=%t first=%d second=%d", prepared, first.runs, second.runs)
	}
	if task.State != domain.TaskSucceeded || task.ProgressCurrent != 3 || task.ProgressTotal != 3 {
		t.Fatalf("task=%+v", task)
	}
	if len(store.created) != 1 || len(store.updated) != 4 {
		t.Fatalf("created=%d updated=%d", len(store.created), len(store.updated))
	}
}

func TestRecoveryCoordinatorContinuesWithActionableParticipantFailure(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	failed := &recoveryParticipantFake{name: "port-mappings", err: errors.New("nft unavailable")}
	next := &recoveryParticipantFake{name: "data-plane"}
	task, err := NewRecoveryCoordinator(store, failed, next).Execute(context.Background(), "service_restart", nil)
	if err == nil || !strings.Contains(err.Error(), "port-mappings") {
		t.Fatalf("err=%v", err)
	}
	if next.runs != 1 || task.State != domain.TaskFailed || task.Error == nil || !strings.Contains(task.Error.Message, "nft unavailable") {
		t.Fatalf("next=%d task=%+v", next.runs, task)
	}
	if task.Error.TaskID != task.ID || task.Error.ResourceType != "host" || task.Error.Phase != "recovery" || task.Error.Cleanup == "" || task.Error.OperatorHint == "" || task.Error.RetryAfterSeconds == 0 {
		t.Fatalf("unstructured recovery error: %+v", task.Error)
	}
}

func TestRecoveryCoordinatorPersistsResourceCheckpoints(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	participant := &checkpointParticipantFake{recoveryParticipantFake: recoveryParticipantFake{name: "nodes"}}
	task, err := NewRecoveryCoordinator(store, participant).Execute(context.Background(), "service_restart", nil)
	if err != nil {
		t.Fatal(err)
	}
	outcomes, ok := task.Result["resource_outcomes"].([]map[string]any)
	if !ok || len(outcomes) != 2 || outcomes[0]["resource_id"] != domain.ID("node-1") || outcomes[0]["runtime_id"] != "pid-42" {
		t.Fatalf("outcomes=%#v", task.Result["resource_outcomes"])
	}
	if len(store.updated) < 3 {
		t.Fatalf("checkpoint was not durably updated: %d", len(store.updated))
	}
}

func TestRecoveryCoordinatorPublishesRecoveredObjectLinkBeforeTaskCheckpoint(t *testing.T) {
	store := &recoveryTaskStoreFake{}
	participant := &objectLinkCheckpointParticipant{recoveryParticipantFake: recoveryParticipantFake{name: "data-plane"}}
	if _, err := NewRecoveryCoordinator(store, participant).Execute(context.Background(), "service_restart", nil); err != nil {
		t.Fatal(err)
	}
	if len(store.recovered) != 1 || store.recovered[0] != "object-link-1" {
		t.Fatalf("recovered=%v", store.recovered)
	}
}
