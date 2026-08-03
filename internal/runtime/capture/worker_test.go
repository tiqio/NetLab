package capture

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestTCPDumpCaptureUsesImmediateMode(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "tcpdump")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	command, err := captureCommand(context.Background(), WorkerConfig{Interface: "tap0", Filter: "icmp"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(command.Args, "--immediate-mode") {
		t.Fatalf("args=%v", command.Args)
	}
}

func TestWorkerStreamsRetainsTruncatesAndCleansSubscribers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.pcap")
	worker, err := NewWorker(WorkerConfig{Interface: "tap-test", MaxBytes: 8, Duration: time.Second, RetainPath: path})
	if err != nil {
		t.Fatal(err)
	}
	stream, cancel := worker.Subscribe()
	defer cancel()
	worker.StartReader(context.Background(), bytes.NewReader([]byte("0123456789abcdef")))
	var received []byte
	for chunk := range stream {
		received = append(received, chunk...)
	}
	if string(received) != "01234567" || worker.Bytes() != 8 || !worker.Truncated() {
		t.Fatalf("received=%q bytes=%d truncated=%v", received, worker.Bytes(), worker.Truncated())
	}
	retained, err := os.ReadFile(path)
	if err != nil || string(retained) != "01234567" {
		t.Fatalf("retained=%q err=%v", retained, err)
	}
}

func TestWorkerRejectsUnsafeInterface(t *testing.T) {
	if _, err := NewWorker(WorkerConfig{Interface: "tap0;rm -rf /"}); err == nil {
		t.Fatal("unsafe interface accepted")
	}
}

func TestLateSubscriberReceivesCapturePrefix(t *testing.T) {
	worker, err := NewWorker(WorkerConfig{Interface: "tap-test", MaxBytes: 1024, Duration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	reader, writer := io.Pipe()
	worker.StartReader(context.Background(), reader)
	_, _ = writer.Write([]byte("pcap-header"))
	time.Sleep(10 * time.Millisecond)
	stream, cancel := worker.Subscribe()
	defer cancel()
	select {
	case prefix := <-stream:
		if string(prefix) != "pcap-header" {
			t.Fatalf("prefix=%q", prefix)
		}
	case <-time.After(time.Second):
		t.Fatal("late subscriber received no prefix")
	}
	_ = writer.Close()
}

func TestExpectedCaptureTerminationDoesNotRecordWaitError(t *testing.T) {
	waitErr := errors.New("exit status 1")
	for name, record := range map[string]bool{
		"unexpected":       shouldRecordWaitError(waitErr, false, false, nil),
		"explicit stop":    shouldRecordWaitError(waitErr, true, false, context.Canceled),
		"byte truncation":  shouldRecordWaitError(waitErr, false, true, nil),
		"duration timeout": shouldRecordWaitError(waitErr, false, false, context.DeadlineExceeded),
	} {
		expected := name == "unexpected"
		if record != expected {
			t.Fatalf("%s record=%v expected=%v", name, record, expected)
		}
	}
}

func TestCaptureDirectionValidation(t *testing.T) {
	if _, err := NewWorker(WorkerConfig{Interface: "tap0", Direction: "sideways"}); err != nil {
		t.Fatal(err)
	}
	if _, err := captureCommand(context.Background(), WorkerConfig{Interface: "tap0", Direction: "sideways"}); err == nil {
		t.Fatal("invalid direction accepted")
	}
}
