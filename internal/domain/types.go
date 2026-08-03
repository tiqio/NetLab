package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type ID string

func NewID() ID {
	var random [10]byte
	if _, err := rand.Read(random[:]); err != nil {
		panic(fmt.Errorf("generate id: %w", err))
	}
	millis := uint64(time.Now().UTC().UnixMilli())
	return ID(fmt.Sprintf("%012x-%s", millis, hex.EncodeToString(random[:])))
}

func (id ID) Validate() error {
	if len(id) != 33 || id[12] != '-' {
		return errors.New("invalid id")
	}
	return nil
}

type Revision int64

func (r Revision) Next() Revision { return r + 1 }

type Problem struct {
	Code              string         `json:"code"`
	Message           string         `json:"message"`
	Retryable         bool           `json:"retryable"`
	ResourceType      string         `json:"resource_type,omitempty"`
	ResourceID        ID             `json:"resource_id,omitempty"`
	TaskID            ID             `json:"task_id,omitempty"`
	Phase             string         `json:"phase,omitempty"`
	Cleanup           string         `json:"cleanup,omitempty"`
	OperatorHint      string         `json:"operator_hint,omitempty"`
	RetryAfterSeconds int            `json:"retry_after_seconds,omitempty"`
	Details           map[string]any `json:"details,omitempty"`
}

func (p Problem) Error() string { return p.Message }

func ProblemFromError(err error) (Problem, bool) {
	var value Problem
	if errors.As(err, &value) {
		return value, true
	}
	var pointer *Problem
	if errors.As(err, &pointer) && pointer != nil {
		return *pointer, true
	}
	return Problem{}, false
}

func NormalizeProblem(err error, fallback Problem) Problem {
	problem, ok := ProblemFromError(err)
	if !ok {
		problem = fallback
		if err != nil {
			problem.Message = err.Error()
		}
	}
	if problem.Code == "" {
		problem.Code = fallback.Code
	}
	if problem.Message == "" {
		problem.Message = fallback.Message
	}
	if problem.ResourceType == "" {
		problem.ResourceType = fallback.ResourceType
	}
	if problem.ResourceID == "" {
		problem.ResourceID = fallback.ResourceID
	}
	if problem.TaskID == "" {
		problem.TaskID = fallback.TaskID
	}
	if problem.Phase == "" {
		problem.Phase = fallback.Phase
	}
	if problem.Cleanup == "" {
		problem.Cleanup = fallback.Cleanup
	}
	if problem.OperatorHint == "" {
		problem.OperatorHint = fallback.OperatorHint
	}
	if problem.RetryAfterSeconds == 0 {
		problem.RetryAfterSeconds = fallback.RetryAfterSeconds
	}
	if problem.Details == nil {
		problem.Details = fallback.Details
	}
	return problem
}

type DesiredState string

const (
	DesiredStopped DesiredState = "stopped"
	DesiredRunning DesiredState = "running"
	DesiredDeleted DesiredState = "deleted"
)

type ObservedState string

const (
	ObservedUnknown      ObservedState = "unknown"
	ObservedProvisioning ObservedState = "provisioning"
	ObservedStarting     ObservedState = "starting"
	ObservedRunning      ObservedState = "running"
	ObservedStopping     ObservedState = "stopping"
	ObservedStopped      ObservedState = "stopped"
	ObservedFailed       ObservedState = "failed"
	ObservedDeleting     ObservedState = "deleting"
)

type TaskState string

const (
	TaskQueued     TaskState = "queued"
	TaskRunning    TaskState = "running"
	TaskCancelling TaskState = "cancelling"
	TaskSucceeded  TaskState = "succeeded"
	TaskFailed     TaskState = "failed"
	TaskCancelled  TaskState = "cancelled"
)

type CaptureState string

const (
	CaptureRequested CaptureState = "requested"
	CaptureStarting  CaptureState = "starting"
	CaptureStreaming CaptureState = "streaming"
	CaptureStopping  CaptureState = "stopping"
	CaptureCompleted CaptureState = "completed"
	CaptureCancelled CaptureState = "cancelled"
	CaptureFailed    CaptureState = "failed"
	CaptureTruncated CaptureState = "truncated"
)
