package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type AutomationTaskService struct {
	exporter *ExportService
	importer *ImportService
	runner   *task.Runner
}

func NewAutomationTaskService(exporter *ExportService, importer *ImportService, runner *task.Runner) *AutomationTaskService {
	service := &AutomationTaskService{exporter: exporter, importer: importer, runner: runner}
	runner.Register("laboratory.export", service.handleExport)
	runner.Register("laboratory.import", service.handleImport)
	runner.Register("laboratory.duplicate", service.handleDuplicate)
	return service
}

func (s *AutomationTaskService) Export(ctx context.Context, laboratoryID domain.ID, ttl time.Duration, idempotencyKey string) (domain.OperationTask, error) {
	artifactID := domain.NewID()
	input := map[string]any{"laboratory_id": laboratoryID, "ttl_seconds": int64(ttl / time.Second)}
	value := automationOperation("laboratory.export", "artifact", artifactID, idempotencyKey, 2, input, input)
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *AutomationTaskService) Import(ctx context.Context, bundle LaboratoryExport, idempotencyKey string) (domain.OperationTask, error) {
	laboratoryID := domain.NewID()
	input := map[string]any{"bundle": bundle}
	value := automationOperation("laboratory.import", "laboratory", laboratoryID, idempotencyKey, 2, input, input)
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *AutomationTaskService) Duplicate(ctx context.Context, sourceID domain.ID, name, idempotencyKey string) (domain.OperationTask, error) {
	if strings.TrimSpace(name) == "" {
		return domain.OperationTask{}, domain.Problem{Code: "invalid_laboratory_name", Message: "laboratory name required", ResourceType: "laboratory", ResourceID: sourceID}
	}
	targetID := domain.NewID()
	input := map[string]any{"source_laboratory_id": sourceID, "name": name}
	value := automationOperation("laboratory.duplicate", "laboratory", targetID, idempotencyKey, 3, input, input)
	return s.runner.EnqueueOrGet(ctx, value)
}

func automationOperation(kind, resourceType string, resourceID domain.ID, idempotencyKey string, total int, input, fingerprintInput map[string]any) domain.OperationTask {
	body, _ := json.Marshal(fingerprintInput)
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: resourceType, ResourceID: resourceID, IdempotencyKey: idempotencyKey, RequestFingerprint: RequestFingerprint(body), State: domain.TaskQueued, ProgressTotal: total, Input: input, CreatedAt: time.Now().UTC()}
}

func (s *AutomationTaskService) handleExport(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	if s.exporter == nil {
		return nil, fmt.Errorf("export service unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err := s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	laboratoryID := domain.ID(taskText(value.Input["laboratory_id"]))
	ttl := time.Duration(taskInt64(value.Input["ttl_seconds"])) * time.Second
	artifact, err := s.exporter.CreateArtifactAs(ctx, value.ResourceID, laboratoryID, ttl)
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"artifact": artifact, "download_url": "/api/v1/artifacts/" + string(artifact.ID)}, nil
}

func (s *AutomationTaskService) handleImport(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	bundle, err := taskBundle(value.Input["bundle"])
	if err != nil {
		return nil, err
	}
	if err = s.importer.Preflight(ctx, bundle); err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	laboratory, err := s.importer.ImportAs(ctx, value.ResourceID, bundle)
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"laboratory": laboratory}, nil
}

func (s *AutomationTaskService) handleDuplicate(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	var bundle LaboratoryExport
	var err error
	if raw, ok := value.Input["bundle"]; ok {
		bundle, err = taskBundle(raw)
	} else {
		bundle, err = s.exporter.Build(ctx, domain.ID(taskText(value.Input["source_laboratory_id"])))
		if err == nil {
			bundle.Laboratory.Name = taskText(value.Input["name"])
			value.Input["bundle"] = bundle
		}
	}
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	if err = s.importer.Preflight(ctx, bundle); err != nil {
		return nil, err
	}
	value.ProgressCurrent = 2
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	laboratory, err := s.importer.ImportAs(ctx, value.ResourceID, bundle)
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"laboratory": laboratory}, nil
}

func taskBundle(value any) (LaboratoryExport, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return LaboratoryExport{}, err
	}
	var bundle LaboratoryExport
	if err = json.Unmarshal(body, &bundle); err != nil {
		return LaboratoryExport{}, err
	}
	return bundle, nil
}

func taskInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case json.Number:
		result, _ := typed.Int64()
		return result
	default:
		return 0
	}
}
