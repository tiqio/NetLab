package readiness

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestNotifyWritesReadyAfterSanitizingStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notify.sock")
	listener, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: path, Net: "unixgram"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	t.Setenv("NOTIFY_SOCKET", path)
	if err = Notify("database recovered\nlistener bound"); err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 256)
	count, _, err := listener.ReadFromUnix(buffer)
	if err != nil {
		t.Fatal(err)
	}
	message := string(buffer[:count])
	if message != "READY=1\nSTATUS=database recovered listener bound" {
		t.Fatalf("unexpected notification %q", message)
	}
}

func TestNotifyIsOptionalOutsideSystemd(t *testing.T) {
	_ = os.Unsetenv("NOTIFY_SOCKET")
	if err := Notify("ready"); err != nil {
		t.Fatal(err)
	}
}
