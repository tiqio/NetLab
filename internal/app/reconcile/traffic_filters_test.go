package reconcile

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
)

func TestTrafficFilterRejectsEmptyScope(t *testing.T) {
	manager := NewTrafficFilterManager()
	_, err := manager.StartScoped("lab", captureRuntime.Match{Protocol: "icmp"}, 100, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "select at least one") {
		t.Fatalf("err=%v", err)
	}
}

func TestTrafficFilterColorDefaultAndValidation(t *testing.T) {
	manager := NewTrafficFilterManager()
	defaulted, err := manager.StartScoped("lab", captureRuntime.Match{Protocol: "icmp"}, 100, []domain.ID{"if-a"}, nil)
	if err != nil || defaulted.Color != defaultTrafficFilterColor {
		t.Fatalf("defaulted=%+v err=%v", defaulted, err)
	}
	custom, err := manager.StartScopedAsWithColor("filter-custom", "lab", captureRuntime.Match{Protocol: "tcp"}, 100, []domain.ID{"if-a"}, nil, "#22c55e")
	if err != nil || custom.Color != "#22c55e" {
		t.Fatalf("custom=%+v err=%v", custom, err)
	}
	for _, invalid := range []string{"red", "#fff"} {
		if _, err = manager.StartScopedAsWithColor(domain.NewID(), "lab", captureRuntime.Match{Protocol: "udp"}, 100, []domain.ID{"if-a"}, nil, invalid); err == nil {
			t.Fatalf("expected invalid color %q to be rejected", invalid)
		}
	}
}

func TestTrafficFilterSharesAvailableCaptureBudgetAcrossScope(t *testing.T) {
	installFakeDumpcap(t)
	captures := NewCaptureManager(t.TempDir(), 2, 64<<20, time.Hour)
	manager := NewTrafficFilterManager(captures)
	filter, err := manager.StartScoped(
		"lab",
		captureRuntime.Match{Protocol: "icmp"},
		100,
		[]domain.ID{"if-a", "if-b"},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(captures.values) != 2 {
		t.Fatalf("captures=%d, want 2", len(captures.values))
	}
	for _, value := range captures.values {
		if value.metadata.MaxBytes != 32<<20 {
			t.Fatalf("max_bytes=%d, want %d", value.metadata.MaxBytes, 32<<20)
		}
	}
	if _, err = manager.Stop(filter.ID); err != nil {
		t.Fatal(err)
	}
}

func TestTrafficFilterListIsNewestFirst(t *testing.T) {
	manager := NewTrafficFilterManager()
	first, err := manager.StartScoped("lab", captureRuntime.Match{Protocol: "icmp"}, 100, []domain.ID{"if-a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	second, err := manager.StartScoped("lab", captureRuntime.Match{Protocol: "udp"}, 100, []domain.ID{"if-b"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	values := manager.List("lab")
	if len(values) != 2 || values[0].ID != second.ID || values[1].ID != first.ID {
		t.Fatalf("values=%+v", values)
	}
}

func TestTrafficFilterDecodesPacketsAndHonorsScope(t *testing.T) {
	manager := NewTrafficFilterManager()
	filter, err := manager.StartScoped("lab", captureRuntime.Match{Protocol: "udp", DestinationPort: 53}, 100, []domain.ID{"if-a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, 14+20+8)
	copy(frame[0:6], []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x02})
	copy(frame[6:12], []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x01})
	binary.BigEndian.PutUint16(frame[12:14], 0x0800)
	frame[14], frame[23] = 0x45, 17
	copy(frame[26:30], []byte{192, 0, 2, 1})
	copy(frame[30:34], []byte{192, 0, 2, 53})
	binary.BigEndian.PutUint16(frame[34:36], 1234)
	binary.BigEndian.PutUint16(frame[36:38], 53)
	pcap := make([]byte, 40+len(frame))
	binary.LittleEndian.PutUint32(pcap[:4], 0xa1b2c3d4)
	binary.LittleEndian.PutUint32(pcap[32:36], uint32(len(frame)))
	copy(pcap[40:], frame)
	manager.ObserveCapture("lab", "if-b", "link", "ingress", "pcap", pcap, time.Now())
	manager.ObserveCapture("lab", "if-a", "link", "ingress", "pcap", pcap, time.Now())
	value, _, err := manager.Get(filter.ID)
	if err != nil || len(value.Observations) != 1 {
		t.Fatalf("observations=%+v err=%v", value.Observations, err)
	}
	if value.Observations[0].SourceMAC != "02:00:00:00:00:01" || value.Observations[0].DestinationMAC != "02:00:00:00:00:02" {
		t.Fatalf("packet direction metadata missing: %+v", value.Observations[0])
	}
}

func TestTrafficFilterAcceptsEitherSelectedInterfaceOrLink(t *testing.T) {
	manager := NewTrafficFilterManager()
	filter, err := manager.StartScoped("lab", captureRuntime.Match{Protocol: "udp", DestinationPort: 53}, 100, []domain.ID{"if-a"}, []domain.ID{"link-a"})
	if err != nil {
		t.Fatal(err)
	}
	frame := make([]byte, 14+20+8)
	binary.BigEndian.PutUint16(frame[12:14], 0x0800)
	frame[14], frame[23] = 0x45, 17
	copy(frame[26:30], []byte{192, 0, 2, 1})
	copy(frame[30:34], []byte{192, 0, 2, 53})
	binary.BigEndian.PutUint16(frame[34:36], 1234)
	binary.BigEndian.PutUint16(frame[36:38], 53)
	pcap := make([]byte, 40+len(frame))
	binary.LittleEndian.PutUint32(pcap[:4], 0xa1b2c3d4)
	binary.LittleEndian.PutUint32(pcap[32:36], uint32(len(frame)))
	copy(pcap[40:], frame)
	now := time.Now()
	manager.ObserveCapture("lab", "if-a", "", "egress", "pcap", pcap, now)
	manager.ObserveCapture("lab", "", "link-a", "ingress", "pcap", pcap, now.Add(time.Millisecond))
	value, _, err := manager.Get(filter.ID)
	if err != nil || len(value.Observations) != 2 {
		t.Fatalf("observations=%+v err=%v", value.Observations, err)
	}
}
