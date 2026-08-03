package capture

import (
	"bytes"
	"context"
	"encoding/binary"
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

func TestCaptureCommandExecutesInsideSelectedNamespace(t *testing.T) {
	directory := t.TempDir()
	dumpcapPath := filepath.Join(directory, "dumpcap")
	if err := os.WriteFile(dumpcapPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	command, err := captureCommand(context.Background(), WorkerConfig{Interface: "eth0", Namespace: "nlr-test", Filter: "icmp"})
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := []string{"ip", "netns", "exec", "nlr-test", dumpcapPath, "-i", "eth0"}
	if len(command.Args) < len(wantPrefix) || !slices.Equal(command.Args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("args=%v want prefix=%v", command.Args, wantPrefix)
	}
}

func TestWorkerStopCancelsCaptureProcess(t *testing.T) {
	directory := t.TempDir()
	tcpdumpPath := filepath.Join(directory, "tcpdump")
	if err := os.WriteFile(tcpdumpPath, []byte("#!/bin/sh\nwhile :; do sleep 1; done\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	worker, err := NewWorker(WorkerConfig{Interface: "eth0", Duration: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if err = worker.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	worker.Stop()
	select {
	case <-worker.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("capture process did not stop after cancellation")
	}
	if !worker.Stopping() || worker.Error() != nil {
		t.Fatalf("stopping=%v err=%v", worker.Stopping(), worker.Error())
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

func TestWorkerCountsBothDirectionsFromOneEndpoint(t *testing.T) {
	worker, err := NewWorker(WorkerConfig{Interface: "eth0", MaxBytes: 1 << 20, Duration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	worker.StartReader(context.Background(), bytes.NewReader(twoDirectionPCAP()))
	select {
	case <-worker.Done():
	case <-time.After(time.Second):
		t.Fatal("capture reader did not finish")
	}
	if worker.Packets() != 2 {
		t.Fatalf("packets=%d want=2", worker.Packets())
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

func twoDirectionPCAP() []byte {
	frames := [][]byte{
		makeIPv4Frame([4]byte{192, 0, 2, 1}, [4]byte{192, 0, 2, 2}),
		makeIPv4Frame([4]byte{192, 0, 2, 2}, [4]byte{192, 0, 2, 1}),
	}
	result := make([]byte, 24)
	binary.LittleEndian.PutUint32(result[:4], 0xa1b2c3d4)
	for _, frame := range frames {
		record := make([]byte, 16+len(frame))
		binary.LittleEndian.PutUint32(record[8:12], uint32(len(frame)))
		binary.LittleEndian.PutUint32(record[12:16], uint32(len(frame)))
		copy(record[16:], frame)
		result = append(result, record...)
	}
	return result
}

func makeIPv4Frame(source, destination [4]byte) []byte {
	frame := make([]byte, 14+20+8)
	copy(frame[0:6], []byte{0x02, 0, 0, 0, 0, 2})
	copy(frame[6:12], []byte{0x02, 0, 0, 0, 0, 1})
	binary.BigEndian.PutUint16(frame[12:14], 0x0800)
	frame[14], frame[23] = 0x45, 17
	copy(frame[26:30], source[:])
	copy(frame[30:34], destination[:])
	return frame
}
