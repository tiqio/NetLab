package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
)

type CaptureTaskService struct {
	captures *CaptureManager
	filters  *TrafficFilterManager
	runner   *task.Runner
}

func NewCaptureTaskService(captures *CaptureManager, filters *TrafficFilterManager, runner *task.Runner) *CaptureTaskService {
	service := &CaptureTaskService{captures: captures, filters: filters, runner: runner}
	runner.Register("capture.start", service.handleCaptureStart)
	runner.Register("capture.stop", service.handleCaptureStop)
	runner.Register("traffic_filter.start", service.handleFilterStart)
	runner.Register("traffic_filter.stop", service.handleFilterStop)
	return service
}

func (s *CaptureTaskService) StartCapture(ctx context.Context, request CaptureRequest, idempotencyKey string) (domain.Capture, domain.OperationTask, error) {
	id := domain.NewID()
	input := map[string]any{"request": request}
	value := runtimeOperation("capture.start", "capture", id, idempotencyKey, 2, input)
	queued, err := s.runner.EnqueueOrGet(ctx, value)
	if err != nil {
		return domain.Capture{}, domain.OperationTask{}, err
	}
	if queued.ID != value.ID {
		id = queued.ResourceID
		request, _ = captureTaskRequest(queued.Input["request"])
	}
	return domain.Capture{ID: id, LaboratoryID: request.LaboratoryID, SourceType: request.SourceType, SourceID: request.SourceID, Filter: request.Filter, Format: request.Format, State: "starting", Retain: request.Retain, MaxBytes: request.MaxBytes, CreatedAt: value.CreatedAt}, queued, nil
}

func (s *CaptureTaskService) StopCapture(ctx context.Context, id domain.ID, idempotencyKey string) (domain.OperationTask, error) {
	request, err := s.captures.Request(id)
	if err != nil {
		return domain.OperationTask{}, err
	}
	input := map[string]any{"request": request}
	value := runtimeOperation("capture.stop", "capture", id, idempotencyKey, 2, input)
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *CaptureTaskService) StartFilter(ctx context.Context, laboratoryID domain.ID, match captureRuntime.Match, maximum int, interfaceIDs, linkIDs []domain.ID, color, idempotencyKey string) (domain.TrafficFilter, domain.OperationTask, error) {
	return s.StartFilterWithObjectLinks(ctx, laboratoryID, match, maximum, interfaceIDs, linkIDs, nil, color, idempotencyKey)
}

func (s *CaptureTaskService) StartFilterWithObjectLinks(ctx context.Context, laboratoryID domain.ID, match captureRuntime.Match, maximum int, interfaceIDs, linkIDs, objectLinkIDs []domain.ID, color, idempotencyKey string) (domain.TrafficFilter, domain.OperationTask, error) {
	expression, err := captureRuntime.Compile(match)
	if err != nil {
		return domain.TrafficFilter{}, domain.OperationTask{}, err
	}
	id := domain.NewID()
	input := map[string]any{"laboratory_id": laboratoryID, "match": match, "max_observations": maximum, "interface_ids": interfaceIDs, "link_ids": linkIDs, "network_object_link_ids": objectLinkIDs, "color": color}
	value := runtimeOperation("traffic_filter.start", "traffic_filter", id, idempotencyKey, 2, input)
	queued, err := s.runner.EnqueueOrGet(ctx, value)
	if err != nil {
		return domain.TrafficFilter{}, domain.OperationTask{}, err
	}
	if queued.ID != value.ID {
		id = queued.ResourceID
		laboratoryID = domain.ID(runtimeTaskText(queued.Input["laboratory_id"]))
		maximum = int(runtimeTaskInt64(queued.Input["max_observations"]))
		interfaceIDs = runtimeTaskIDs(queued.Input["interface_ids"])
		linkIDs = runtimeTaskIDs(queued.Input["link_ids"])
		objectLinkIDs = runtimeTaskIDs(queued.Input["network_object_link_ids"])
		color = runtimeTaskText(queued.Input["color"])
	}
	return domain.TrafficFilter{ID: id, LaboratoryID: laboratoryID, Expression: expression, Color: color, State: "starting", MaxObservations: maximum, InterfaceIDs: interfaceIDs, LinkIDs: linkIDs, NetworkObjectLinkIDs: objectLinkIDs, CreatedAt: value.CreatedAt}, queued, nil
}

func (s *CaptureTaskService) StopFilter(ctx context.Context, id domain.ID, idempotencyKey string) (domain.OperationTask, error) {
	value, match, err := s.filters.Definition(id)
	if err != nil {
		return domain.OperationTask{}, err
	}
	input := map[string]any{"laboratory_id": value.LaboratoryID, "match": match, "max_observations": value.MaxObservations, "interface_ids": value.InterfaceIDs, "link_ids": value.LinkIDs, "network_object_link_ids": value.NetworkObjectLinkIDs, "color": value.Color}
	operation := runtimeOperation("traffic_filter.stop", "traffic_filter", id, idempotencyKey, 2, input)
	return s.runner.EnqueueOrGet(ctx, operation)
}

func runtimeOperation(kind, resourceType string, resourceID domain.ID, idempotencyKey string, total int, input map[string]any) domain.OperationTask {
	body, _ := json.Marshal(input)
	sum := sha256.Sum256(body)
	return domain.OperationTask{ID: domain.NewID(), Kind: kind, ResourceType: resourceType, ResourceID: resourceID, IdempotencyKey: idempotencyKey, RequestFingerprint: hex.EncodeToString(sum[:]), State: domain.TaskQueued, ProgressTotal: total, Input: input, CreatedAt: time.Now().UTC()}
}

func (s *CaptureTaskService) handleCaptureStart(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	request, err := captureTaskRequest(value.Input["request"])
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	capture, err := s.captures.StartAs(context.Background(), value.ResourceID, request)
	if ctx.Err() != nil {
		_, _ = s.captures.Stop(value.ResourceID)
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return captureTaskEnvelope(capture), nil
}

func (s *CaptureTaskService) handleCaptureStop(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	request, err := captureTaskRequest(value.Input["request"])
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	if _, err = s.captures.Stop(value.ResourceID); err != nil {
		return nil, err
	}
	capture, err := s.waitCaptureTerminal(ctx, value.ResourceID)
	if ctx.Err() != nil {
		s.restartCapture(value.ResourceID, request)
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return captureTaskEnvelope(capture), nil
}

func (s *CaptureTaskService) restartCapture(id domain.ID, request CaptureRequest) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := s.captures.Get(id)
		if err != nil || (value.State != "starting" && value.State != "running" && value.State != "stopping") {
			_, _ = s.captures.StartAs(context.Background(), id, request)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *CaptureTaskService) handleFilterStart(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	match, err := runtimeTaskMatch(value.Input["match"])
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	filter, err := s.filters.StartScopedAsWithObjectLinks(value.ResourceID, domain.ID(runtimeTaskText(value.Input["laboratory_id"])), match, int(runtimeTaskInt64(value.Input["max_observations"])), runtimeTaskIDs(value.Input["interface_ids"]), runtimeTaskIDs(value.Input["link_ids"]), runtimeTaskIDs(value.Input["network_object_link_ids"]), runtimeTaskText(value.Input["color"]))
	if ctx.Err() != nil {
		_, _ = s.filters.Stop(value.ResourceID)
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"traffic_filter": filter}, nil
}

func (s *CaptureTaskService) handleFilterStop(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	match, err := runtimeTaskMatch(value.Input["match"])
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	filter, err := s.filters.Stop(value.ResourceID)
	if ctx.Err() != nil {
		_, _ = s.filters.StartScopedAsWithObjectLinks(value.ResourceID, domain.ID(runtimeTaskText(value.Input["laboratory_id"])), match, int(runtimeTaskInt64(value.Input["max_observations"])), runtimeTaskIDs(value.Input["interface_ids"]), runtimeTaskIDs(value.Input["link_ids"]), runtimeTaskIDs(value.Input["network_object_link_ids"]), runtimeTaskText(value.Input["color"]))
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"traffic_filter": filter}, nil
}

func (s *CaptureTaskService) waitCaptureTerminal(ctx context.Context, id domain.ID) (domain.Capture, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		capture, err := s.captures.Get(id)
		if err != nil {
			return domain.Capture{}, err
		}
		if capture.State != "starting" && capture.State != "running" && capture.State != "stopping" {
			return capture, nil
		}
		select {
		case <-ctx.Done():
			return domain.Capture{}, ctx.Err()
		case <-timer.C:
			return domain.Capture{}, domain.Problem{Code: "capture_stop_timeout", Message: "capture did not stop before the task deadline", Retryable: true, ResourceType: "capture", ResourceID: id, Phase: "capture_stop", Cleanup: "capture stop remains requested", OperatorHint: "inspect the capture worker and retry", RetryAfterSeconds: 2}
		case <-ticker.C:
		}
	}
}

func captureTaskEnvelope(value domain.Capture) map[string]any {
	return map[string]any{"capture": value, "stream_url": "/api/v1/captures/" + string(value.ID) + "/stream", "wireshark": map[string]any{"mode": "http_stream", "media_type": captureMediaType(value.Format)}}
}

func captureMediaType(format string) string {
	if format == "pcapng" {
		return "application/x-pcapng"
	}
	return "application/vnd.tcpdump.pcap"
}

func captureTaskRequest(value any) (CaptureRequest, error) {
	var request CaptureRequest
	return request, runtimeTaskDecode(value, &request)
}

func runtimeTaskMatch(value any) (captureRuntime.Match, error) {
	var match captureRuntime.Match
	return match, runtimeTaskDecode(value, &match)
}

func runtimeTaskDecode(value any, target any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(body, target); err != nil {
		return err
	}
	return nil
}

func runtimeTaskText(value any) string {
	text, _ := value.(string)
	return text
}

func runtimeTaskInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func runtimeTaskIDs(value any) []domain.ID {
	var ids []domain.ID
	_ = runtimeTaskDecode(value, &ids)
	return ids
}
