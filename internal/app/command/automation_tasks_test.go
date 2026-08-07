package command

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type automationRepositoryFake struct {
	mu          sync.Mutex
	snapshots   map[domain.ID]domain.TopologySnapshot
	labs        map[domain.ID]domain.Laboratory
	imports     int
	blockImport bool
}

func newAutomationRepositoryFake() *automationRepositoryFake {
	return &automationRepositoryFake{snapshots: map[domain.ID]domain.TopologySnapshot{}, labs: map[domain.ID]domain.Laboratory{}}
}

func (r *automationRepositoryFake) Snapshot(_ context.Context, id domain.ID) (domain.TopologySnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.snapshots[id]
	if !ok {
		return domain.TopologySnapshot{}, errors.New("not found")
	}
	return value, nil
}

func (r *automationRepositoryFake) GetLaboratory(_ context.Context, id domain.ID) (domain.Laboratory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.labs[id]
	if !ok {
		return domain.Laboratory{}, errors.New("not found")
	}
	return value, nil
}

func (r *automationRepositoryFake) ImportTopology(ctx context.Context, lab domain.Laboratory, _ []domain.Node, _ []domain.Interface, _ []domain.Link, _ []domain.NetworkObject, _ []domain.NetworkObjectLink, _ []domain.TopologyPlacement) error {
	if r.blockImport {
		<-ctx.Done()
		return ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.imports++
	r.labs[lab.ID] = lab
	return nil
}

type automationArtifactWriterFake struct {
	mu        sync.Mutex
	artifacts map[domain.ID]domain.Artifact
}

func (w *automationArtifactWriterFake) Create(ctx context.Context, kind, mediaType, ownerType string, ownerID domain.ID, body []byte, ttl time.Duration) (domain.Artifact, error) {
	return w.CreateWithID(ctx, domain.NewID(), kind, mediaType, ownerType, ownerID, body, ttl)
}

func (w *automationArtifactWriterFake) CreateWithID(_ context.Context, id domain.ID, kind, mediaType, ownerType string, ownerID domain.ID, body []byte, _ time.Duration) (domain.Artifact, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if value, ok := w.artifacts[id]; ok {
		return value, nil
	}
	value := domain.Artifact{ID: id, Kind: kind, MediaType: mediaType, OwnerType: ownerType, OwnerID: ownerID, SizeBytes: int64(len(body)), CreatedAt: time.Now().UTC()}
	w.artifacts[id] = value
	return value, nil
}

func validAutomationBundle(name string) LaboratoryExport {
	return LaboratoryExport{SchemaVersion: 1, Laboratory: ExportLaboratory{Name: name, RecoveryPolicy: domain.RecoveryRemainStopped}, Redaction: ExportRedaction{ImagesExcluded: true, CredentialsExcluded: true, BootstrapSecretsExcluded: true, CapturesExcluded: true}}
}

func TestAutomationTaskIdempotencyReplaysAndRejectsConflicts(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newAutomationRepositoryFake()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewAutomationTaskService(NewExportService(repository, &automationArtifactWriterFake{artifacts: map[domain.ID]domain.Artifact{}}), NewImportService(repository, nil), runner)

	first, err := service.Import(context.Background(), validAutomationBundle("imported"), "import-key")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Import(context.Background(), validAutomationBundle("imported"), "import-key")
	if err != nil || second.ID != first.ID {
		t.Fatalf("first=%s second=%s err=%v", first.ID, second.ID, err)
	}
	if _, err = service.Import(context.Background(), validAutomationBundle("different"), "import-key"); err == nil {
		t.Fatal("expected idempotency conflict")
	} else if problem, ok := domain.ProblemFromError(err); !ok || problem.Code != "idempotency_conflict" || problem.TaskID != first.ID {
		t.Fatalf("problem=%+v ok=%t", problem, ok)
	}
	waitForTopologyTask(t, store, first.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
}

func TestAutomationDuplicateRecoveryReusesPersistedBundleAndTarget(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newAutomationRepositoryFake()
	repository.snapshots["source"] = domain.TopologySnapshot{Laboratory: domain.Laboratory{ID: "source", Name: "source", RecoveryPolicy: domain.RecoveryRemainStopped}}
	value := domain.OperationTask{ID: "duplicate-task", Kind: "laboratory.duplicate", ResourceType: "laboratory", ResourceID: "target", State: domain.TaskRunning, ProgressTotal: 3, Input: map[string]any{"source_laboratory_id": "source", "name": "copy"}, CreatedAt: time.Now().UTC()}
	store.tasks[value.ID] = value

	runner := task.NewRunner(store, 1, 8)
	NewAutomationTaskService(NewExportService(repository, &automationArtifactWriterFake{artifacts: map[domain.ID]domain.Artifact{}}), NewImportService(repository, nil), runner)
	if err := runner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	current := waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
	runner.Close()
	if current.ResourceID != "target" || current.Input["bundle"] == nil {
		t.Fatalf("task=%+v", current)
	}
	repository.mu.Lock()
	firstImports := repository.imports
	repository.snapshots["source"] = domain.TopologySnapshot{Laboratory: domain.Laboratory{ID: "source", Name: "changed", RecoveryPolicy: domain.RecoveryRemainStopped}}
	repository.mu.Unlock()
	current.State = domain.TaskRunning
	current.FinishedAt = nil
	current.Result = nil
	store.tasks[current.ID] = current

	recoveryRunner := task.NewRunner(store, 1, 8)
	defer recoveryRunner.Close()
	NewAutomationTaskService(NewExportService(repository, &automationArtifactWriterFake{artifacts: map[domain.ID]domain.Artifact{}}), NewImportService(repository, nil), recoveryRunner)
	if err := recoveryRunner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.imports != firstImports || repository.labs["target"].Name != "copy" {
		t.Fatalf("imports=%d first=%d lab=%+v", repository.imports, firstImports, repository.labs["target"])
	}
}

func TestAutomationImportCancellationInterruptsUnderlyingOperation(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newAutomationRepositoryFake()
	repository.blockImport = true
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewAutomationTaskService(NewExportService(repository, &automationArtifactWriterFake{artifacts: map[domain.ID]domain.Artifact{}}), NewImportService(repository, nil), runner)
	value, err := service.Import(context.Background(), validAutomationBundle("cancelled"), "cancel-import")
	if err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool {
		return current.State == domain.TaskRunning && current.ProgressCurrent == 1
	})
	if err = runner.Cancel(context.Background(), value.ID); err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskCancelled })
	if _, err = repository.GetLaboratory(context.Background(), value.ResourceID); err == nil {
		t.Fatal("cancelled import created a laboratory")
	}
}
