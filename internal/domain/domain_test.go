package domain

import "testing"

func TestIDAndRevision(t *testing.T) {
	id := NewID()
	if err := id.Validate(); err != nil {
		t.Fatal(err)
	}
	if Revision(4).Next() != 5 {
		t.Fatal("revision did not increment")
	}
}

func TestNodeTransitions(t *testing.T) {
	if !CanTransitionNode(ObservedStopped, ObservedStarting) {
		t.Fatal("expected start transition")
	}
	if CanTransitionNode(ObservedStopped, ObservedRunning) {
		t.Fatal("unexpected direct transition")
	}
	if err := ValidateNodeTransition(ObservedRunning, ObservedStopped); err == nil {
		t.Fatal("expected invalid transition")
	}
}

func TestTaskAndCaptureTransitions(t *testing.T) {
	if !CanTransitionTask(TaskQueued, TaskRunning) || CanTransitionTask(TaskSucceeded, TaskRunning) {
		t.Fatal("task transitions invalid")
	}
	if !CanTransitionCapture(CaptureStreaming, CaptureTruncated) || CanTransitionCapture(CaptureCompleted, CaptureStreaming) {
		t.Fatal("capture transitions invalid")
	}
}
