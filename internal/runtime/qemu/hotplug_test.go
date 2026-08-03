package qemu

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/go-qemu/qmp"
)

type fakeQMP struct {
	commands []string
	args     []any
	fail     string
	events   chan qmp.Event
}

func (f *fakeQMP) Run(command string, args any) (json.RawMessage, error) {
	f.commands = append(f.commands, command)
	f.args = append(f.args, args)
	if command == f.fail {
		return nil, errors.New("injected")
	}
	return json.RawMessage(`{}`), nil
}
func (f *fakeQMP) Events(context.Context) (<-chan qmp.Event, error) { return f.events, nil }

func TestHotAddRollbackAndDeleteEvent(t *testing.T) {
	failure := &fakeQMP{fail: "device_add", events: make(chan qmp.Event, 1)}
	if err := AddNIC(context.Background(), failure, HotplugNIC{ID: "nic1", NetdevID: "net1", TapName: "tap1", Driver: "virtio-net-pci"}); err == nil {
		t.Fatal("expected failure")
	}
	if len(failure.commands) != 3 || failure.commands[2] != "netdev_del" {
		t.Fatalf("commands=%v", failure.commands)
	}
	successfulAdd := &fakeQMP{events: make(chan qmp.Event, 1)}
	if err := AddNIC(context.Background(), successfulAdd, HotplugNIC{ID: "nic2", NetdevID: "net2", TapName: "tap2", Driver: "virtio-net-pci", Bus: "netlab-rp-2"}); err != nil {
		t.Fatal(err)
	}
	netdevArgs := successfulAdd.args[0].(map[string]any)
	if netdevArgs["vnet_hdr"] != false {
		t.Fatalf("netdev_add args=%v", netdevArgs)
	}
	deviceArgs := successfulAdd.args[1].(map[string]any)
	if deviceArgs["bus"] != "netlab-rp-2" {
		t.Fatalf("device_add args=%v", deviceArgs)
	}
	success := &fakeQMP{events: make(chan qmp.Event, 1)}
	success.events <- qmp.Event{Event: "DEVICE_DELETED", Data: map[string]any{"device": "nic1"}}
	if err := RemoveNIC(context.Background(), success, "nic1", "net1", time.Second); err != nil {
		t.Fatal(err)
	}
	if success.commands[1] != "netdev_del" {
		t.Fatalf("commands=%v", success.commands)
	}
	pathEvent := &fakeQMP{events: make(chan qmp.Event, 1)}
	pathEvent.events <- qmp.Event{Event: "DEVICE_DELETED", Data: map[string]any{"path": "/machine/peripheral/nic2"}}
	if err := RemoveNIC(context.Background(), pathEvent, "nic2", "net2", time.Second); err != nil {
		t.Fatal(err)
	}
	noEvent := &fakeQMP{events: make(chan qmp.Event)}
	if err := RemoveNIC(context.Background(), noEvent, "nic3", "net3", 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if len(noEvent.commands) != 3 || noEvent.commands[1] != "query-pci" || noEvent.commands[2] != "netdev_del" {
		t.Fatalf("commands=%v", noEvent.commands)
	}
}
