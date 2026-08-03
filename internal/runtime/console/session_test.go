package console

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestPersistentSessionReattachesWithoutClosingBackend(t *testing.T) {
	backend, backendPeer := net.Pipe()
	session := NewPersistentSession(backend, Limits{MaximumSession: time.Minute}, time.Second, 1024, nil)
	defer session.Close()
	defer backendPeer.Close()

	firstConnection, firstBrowser := net.Pipe()
	firstDone := make(chan error, 1)
	go func() { firstDone <- session.Attach(firstConnection) }()

	writeAndExpect(t, firstBrowser, backendPeer, "export NETLAB_REFRESH_MARKER=kept123\n")
	_ = firstBrowser.Close()
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("first attachment did not detach")
	}

	if _, err := backendPeer.Write([]byte("detached output\n")); err != nil {
		t.Fatalf("backend was closed after browser detach: %v", err)
	}

	secondConnection, secondBrowser := net.Pipe()
	secondDone := make(chan error, 1)
	go func() { secondDone <- session.Attach(secondConnection) }()
	expectRead(t, secondBrowser, "detached output\n")
	writeAndExpect(t, secondBrowser, backendPeer, "pwd\n")
	_ = secondBrowser.Close()
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("second attachment did not detach")
	}
}

func TestPersistentSessionPublishesBackendOutputToSubscribers(t *testing.T) {
	backend, backendPeer := net.Pipe()
	session := NewPersistentSession(backend, Limits{MaximumSession: time.Minute}, time.Second, 1024, nil)
	defer session.Close()
	defer backendPeer.Close()

	updates, cancel := session.Subscribe(2)
	defer cancel()
	if _, err := backendPeer.Write([]byte("Switch#")); err != nil {
		t.Fatal(err)
	}
	select {
	case value := <-updates:
		if string(value) != "Switch#" {
			t.Fatalf("value=%q", value)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive backend output")
	}
}

func TestPersistentSessionExpiresAfterDetachGrace(t *testing.T) {
	backend, backendPeer := net.Pipe()
	closed := make(chan struct{})
	session := NewPersistentSession(backend, Limits{MaximumSession: time.Minute}, 30*time.Millisecond, 1024, func() { close(closed) })
	defer backendPeer.Close()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("detached session did not expire")
	}
	if _, err := backendPeer.Write([]byte("closed")); err == nil {
		t.Fatal("backend remained writable after session expiry")
	}
	select {
	case <-session.Done():
	default:
		t.Fatal("session did not report closure")
	}
}

func TestPersistentSessionsKeepBackendsIsolated(t *testing.T) {
	firstBackend, firstPeer := net.Pipe()
	secondBackend, secondPeer := net.Pipe()
	first := NewPersistentSession(firstBackend, Limits{MaximumSession: time.Minute}, time.Second, 1024, nil)
	second := NewPersistentSession(secondBackend, Limits{MaximumSession: time.Minute}, time.Second, 1024, nil)
	defer first.Close()
	defer second.Close()
	defer firstPeer.Close()
	defer secondPeer.Close()

	firstConnection, firstBrowser := net.Pipe()
	secondConnection, secondBrowser := net.Pipe()
	go first.Attach(firstConnection)
	go second.Attach(secondConnection)
	defer firstBrowser.Close()
	defer secondBrowser.Close()

	writeAndExpect(t, firstBrowser, firstPeer, "first\n")
	writeAndExpect(t, secondBrowser, secondPeer, "second\n")
}

func writeAndExpect(t *testing.T, source net.Conn, destination net.Conn, value string) {
	t.Helper()
	writeDone := make(chan error, 1)
	go func() {
		_, err := source.Write([]byte(value))
		writeDone <- err
	}()
	expectRead(t, destination, value)
	if err := <-writeDone; err != nil {
		t.Fatalf("write %q: %v", value, err)
	}
}

func expectRead(t *testing.T, source net.Conn, expected string) {
	t.Helper()
	if err := source.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	value := make([]byte, len(expected))
	if _, err := io.ReadFull(source, value); err != nil {
		t.Fatalf("read %q: %v", expected, err)
	}
	if string(value) != expected {
		t.Fatalf("value=%q expected=%q", value, expected)
	}
}
