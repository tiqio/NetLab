package command

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type LinkReconnectService struct {
	repository ports.LinkReconnectRepository
	runtime    ports.LinkReconnectRuntime
	runner     *task.Runner
	timeout    time.Duration
}

func NewLinkReconnectService(repository ports.LinkReconnectRepository, runner *task.Runner) *LinkReconnectService {
	service := &LinkReconnectService{repository: repository, runner: runner, timeout: 2 * time.Minute}
	runner.Register("link.reconnect", service.handle)
	return service
}

func (s *LinkReconnectService) SetRuntime(runtime ports.LinkReconnectRuntime) { s.runtime = runtime }

func (s *LinkReconnectService) Reconnect(ctx context.Context, linkID domain.ID, revision domain.Revision, retainedEndpointID, replacementEndpointID domain.ID, idempotencyKey string) (domain.OperationTask, error) {
	link, err := s.repository.GetLink(ctx, linkID)
	if err != nil {
		return domain.OperationTask{}, err
	}
	if revision != link.Revision {
		return domain.OperationTask{}, domain.Problem{Code: "revision_conflict", Message: "link revision changed", Retryable: true, ResourceType: "link", ResourceID: linkID}
	}
	if retainedEndpointID != link.EndpointAID && retainedEndpointID != link.EndpointBID {
		return domain.OperationTask{}, domain.Problem{Code: "invalid_reconnect_endpoint", Message: "retained endpoint is not attached to the link", ResourceType: "link", ResourceID: linkID}
	}
	if replacementEndpointID == "" || replacementEndpointID == retainedEndpointID || replacementEndpointID == link.EndpointAID || replacementEndpointID == link.EndpointBID {
		return domain.OperationTask{}, domain.Problem{Code: "invalid_reconnect_endpoint", Message: "replacement endpoint must be a different interface", ResourceType: "link", ResourceID: linkID}
	}
	fingerprint, _ := json.Marshal(map[string]any{"link_id": linkID, "revision": revision, "retained_endpoint_id": retainedEndpointID, "replacement_endpoint_id": replacementEndpointID})
	value := domain.OperationTask{
		ID: domain.NewID(), Kind: "link.reconnect", ResourceType: "link", ResourceID: linkID,
		IdempotencyKey: idempotencyKey, RequestFingerprint: RequestFingerprint(fingerprint), RequestedRevision: revision,
		State: domain.TaskQueued, ProgressTotal: 3, CreatedAt: time.Now().UTC(),
		Input: map[string]any{"revision": int64(revision), "retained_endpoint_id": string(retainedEndpointID), "replacement_endpoint_id": string(replacementEndpointID), "previous_endpoint_a_id": string(link.EndpointAID), "previous_endpoint_b_id": string(link.EndpointBID), "cancellation_mode": "before_commit"},
	}
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *LinkReconnectService) handle(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	if s.runtime == nil {
		return nil, domain.Problem{Code: "link_runtime_unavailable", Message: "link runtime is unavailable", Retryable: true, TaskID: value.ID, ResourceType: "link", ResourceID: value.ResourceID, Phase: "reconnect_runtime", Cleanup: "canonical endpoints remain unchanged", OperatorHint: "restore the data-plane runtime and retry"}
	}
	previousA := domain.ID(taskText(value.Input["previous_endpoint_a_id"]))
	previousB := domain.ID(taskText(value.Input["previous_endpoint_b_id"]))
	retained := domain.ID(taskText(value.Input["retained_endpoint_id"]))
	replacement := domain.ID(taskText(value.Input["replacement_endpoint_id"]))
	expectedRevision := domain.Revision(taskNumber(value.Input["revision"]))
	current, err := s.repository.GetLink(ctx, value.ResourceID)
	if err != nil {
		return nil, err
	}
	candidate, candidateMatches := reconnectCandidate(current, retained, replacement)
	if endpointsEqual(current, candidate) && current.Revision != expectedRevision {
		value.ProgressCurrent = value.ProgressTotal
		return map[string]any{"link": current, "previous_endpoint_a_id": previousA, "previous_endpoint_b_id": previousB}, nil
	}
	if !candidateMatches || current.Revision != expectedRevision || current.EndpointAID != previousA || current.EndpointBID != previousB {
		return nil, domain.Problem{Code: "revision_conflict", Message: "link endpoints changed before reconnect completed", Retryable: true, TaskID: value.ID, ResourceType: "link", ResourceID: value.ResourceID, Phase: "reconnect_validation", Cleanup: "canonical endpoints remain unchanged", OperatorHint: "reload the laboratory and retry against the current link revision"}
	}
	previousEndpointA, err := s.repository.GetInterface(ctx, previousA)
	if err != nil {
		return nil, err
	}
	previousEndpointB, err := s.repository.GetInterface(ctx, previousB)
	if err != nil {
		return nil, err
	}
	candidateEndpointA, err := s.repository.GetInterface(ctx, candidate.EndpointAID)
	if err != nil {
		return nil, err
	}
	candidateEndpointB, err := s.repository.GetInterface(ctx, candidate.EndpointBID)
	if err != nil {
		return nil, err
	}
	rollbackRuntime := func(cause error, code string) error {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rollbackErr := s.runtime.EnsureLink(rollbackCtx, current, previousEndpointA, previousEndpointB)
		cleanup := "canonical endpoints remained unchanged and original runtime was restored"
		if rollbackErr != nil {
			cleanup = "canonical endpoints remain unchanged; original runtime restoration failed"
		}
		return domain.Problem{Code: code, Message: cause.Error(), Retryable: true, TaskID: value.ID, ResourceType: "link", ResourceID: value.ResourceID, Phase: "reconnect_runtime", Cleanup: cleanup, OperatorHint: "inspect link diagnostics and retry", Details: map[string]any{"runtime_rollback_succeeded": rollbackErr == nil, "previous_endpoint_a_id": previousA, "previous_endpoint_b_id": previousB}}
	}
	runtimeCtx, cancel := context.WithTimeout(ctx, s.timeout)
	err = s.runtime.EnsureLink(runtimeCtx, candidate, candidateEndpointA, candidateEndpointB)
	runtimeCause := runtimeCtx.Err()
	cancel()
	if err != nil {
		code := "link_reconnect_failed"
		if errors.Is(runtimeCause, context.DeadlineExceeded) {
			code = "link_reconnect_timeout"
		} else if errors.Is(runtimeCause, context.Canceled) || errors.Is(err, context.Canceled) {
			code = "link_reconnect_cancelled"
		}
		return nil, rollbackRuntime(err, code)
	}
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, rollbackRuntime(err, "link_reconnect_cancelled")
	}
	commitCtx, commitCancel := context.WithTimeout(context.Background(), 15*time.Second)
	committed, err := s.repository.CommitLinkReconnect(commitCtx, value.ResourceID, expectedRevision, retained, replacement)
	commitCancel()
	if err != nil {
		return nil, rollbackRuntime(err, "link_reconnect_commit_failed")
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"link": committed, "previous_endpoint_a_id": previousA, "previous_endpoint_b_id": previousB}, nil
}

func reconnectCandidate(current domain.Link, retained, replacement domain.ID) (domain.Link, bool) {
	candidate := current
	switch retained {
	case current.EndpointAID:
		candidate.EndpointBID = replacement
	case current.EndpointBID:
		candidate.EndpointAID = replacement
	default:
		return domain.Link{}, false
	}
	return candidate, true
}

func endpointsEqual(left, right domain.Link) bool {
	return left.EndpointAID == right.EndpointAID && left.EndpointBID == right.EndpointBID
}
