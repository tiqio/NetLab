package domain

import "fmt"

func CanTransitionNode(from, to ObservedState) bool {
	allowed := map[ObservedState]map[ObservedState]bool{
		ObservedUnknown:      {ObservedProvisioning: true, ObservedStarting: true, ObservedRunning: true, ObservedStopped: true, ObservedFailed: true},
		ObservedStopped:      {ObservedProvisioning: true, ObservedStarting: true, ObservedFailed: true, ObservedDeleting: true},
		ObservedProvisioning: {ObservedStarting: true, ObservedFailed: true, ObservedDeleting: true},
		ObservedStarting:     {ObservedRunning: true, ObservedFailed: true, ObservedStopping: true},
		ObservedRunning:      {ObservedStopping: true, ObservedDeleting: true, ObservedFailed: true},
		ObservedStopping:     {ObservedStopped: true, ObservedFailed: true, ObservedDeleting: true},
		ObservedFailed:       {ObservedProvisioning: true, ObservedStarting: true, ObservedStopping: true, ObservedDeleting: true},
		ObservedDeleting:     {ObservedFailed: true},
	}
	return allowed[from][to]
}

func ValidateNodeTransition(from, to ObservedState) error {
	if from == to || CanTransitionNode(from, to) {
		return nil
	}
	return fmt.Errorf("invalid node transition %s -> %s", from, to)
}

func CanTransitionTask(from, to TaskState) bool {
	allowed := map[TaskState]map[TaskState]bool{
		TaskQueued:     {TaskRunning: true, TaskCancelled: true},
		TaskRunning:    {TaskCancelling: true, TaskSucceeded: true, TaskFailed: true, TaskCancelled: true},
		TaskCancelling: {TaskCancelled: true, TaskFailed: true},
	}
	return from == to || allowed[from][to]
}

func CanTransitionCapture(from, to CaptureState) bool {
	allowed := map[CaptureState]map[CaptureState]bool{
		CaptureRequested: {CaptureStarting: true, CaptureCancelled: true, CaptureFailed: true},
		CaptureStarting:  {CaptureStreaming: true, CaptureCancelled: true, CaptureFailed: true},
		CaptureStreaming: {CaptureStopping: true, CaptureCompleted: true, CaptureCancelled: true, CaptureFailed: true, CaptureTruncated: true},
		CaptureStopping:  {CaptureCompleted: true, CaptureCancelled: true, CaptureFailed: true, CaptureTruncated: true},
	}
	return from == to || allowed[from][to]
}
