package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type compatibilityExecutor struct {
	commands [][]string
}

func (e *compatibilityExecutor) Run(_ context.Context, name string, args ...string) error {
	e.commands = append(e.commands, append([]string{name}, args...))
	return nil
}

func (e *compatibilityExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	e.commands = append(e.commands, append([]string{name}, args...))
	return nil, errors.New("not found")
}

func (e *compatibilityExecutor) contains(parts ...string) bool {
	for _, command := range e.commands {
		joined := strings.Join(command, " ")
		matched := true
		for _, part := range parts {
			matched = matched && strings.Contains(joined, part)
		}
		if matched {
			return true
		}
	}
	return false
}

func TestExistingNetworkPrimitivesRetainTheirRuntimeSemantics(t *testing.T) {
	executor := &compatibilityExecutor{}
	dataPlane, err := linuxnet.NewDataPlane(executor)
	if err != nil {
		t.Fatal(err)
	}
	standardLink := domain.Link{ID: "standard-link"}
	interfaceA := domain.Interface{ID: "interface-a"}
	interfaceB := domain.Interface{ID: "interface-b"}
	if err = dataPlane.EnsureLink(context.Background(), standardLink, interfaceA, interfaceB); err != nil {
		t.Fatal(err)
	}
	bridge := domain.NetworkObject{ID: "bridge", Kind: domain.NetworkBridge}
	nat := domain.NetworkObject{ID: "nat", Kind: domain.NetworkNAT}
	if err = dataPlane.Attach(context.Background(), interfaceA, bridge); err != nil {
		t.Fatal(err)
	}
	if err = dataPlane.Attach(context.Background(), interfaceB, nat); err != nil {
		t.Fatal(err)
	}
	for _, expected := range [][]string{
		{"link add", linuxnet.LinkBridgeName(standardLink.ID), "type bridge"},
		{linuxnet.HostInterfaceName(interfaceA.ID), "master", linuxnet.LinkBridgeName(standardLink.ID)},
		{linuxnet.HostInterfaceName(interfaceB.ID), "master", linuxnet.LinkBridgeName(standardLink.ID)},
		{linuxnet.HostInterfaceName(interfaceA.ID), "master", "nlbr"},
		{linuxnet.HostInterfaceName(interfaceB.ID), "master", "nlnat"},
	} {
		if !executor.contains(expected...) {
			t.Fatalf("missing command containing %v in %#v", expected, executor.commands)
		}
	}
}

func TestExistingCaptureAndTrafficFilterScopesRemainDistinct(t *testing.T) {
	installCompatibilityDumpcap(t)
	manager := reconcile.NewCaptureManager(t.TempDir(), 2, 2<<20, time.Hour)
	for _, source := range []struct {
		typeName string
		id       domain.ID
		want     string
	}{
		{typeName: "interface", id: "interface-a", want: linuxnet.HostInterfaceName("interface-a")},
		{typeName: "link", id: "standard-link", want: linuxnet.LinkBridgeName("standard-link")},
	} {
		capture, err := manager.Start(context.Background(), reconcile.CaptureRequest{LaboratoryID: "lab", SourceType: source.typeName, SourceID: source.id, Format: "pcap", MaxBytes: 1 << 20})
		if err != nil {
			t.Fatalf("start %s capture: %v", source.typeName, err)
		}
		request, err := manager.Request(capture.ID)
		if err != nil || request.Interface != source.want || request.Namespace != "" {
			t.Fatalf("source=%s request=%+v err=%v", source.typeName, request, err)
		}
		if _, err = manager.Stop(capture.ID); err != nil {
			t.Fatal(err)
		}
	}

	correlator := captureRuntime.NewCorrelator(time.Second, 10)
	now := time.Now().UTC()
	correlator.Observe("interface-flow", "interface-a", "", "ingress", 64, now)
	correlator.Observe("link-flow", "interface-a", "standard-link", "a_to_b", 128, now.Add(time.Millisecond))
	observations, _ := correlator.SnapshotAt(now.Add(2 * time.Millisecond))
	if len(observations) != 2 {
		t.Fatalf("observations=%+v", observations)
	}
	seen := map[string]bool{}
	for _, observation := range observations {
		seen[observation.ResourceType+":"+string(observation.ResourceID)] = true
	}
	if !seen["interface:interface-a"] || !seen["link:standard-link"] {
		t.Fatalf("existing scopes changed: %+v", observations)
	}
}

func installCompatibilityDumpcap(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "dumpcap"), []byte("#!/bin/sh\nprintf compatibility-pcap\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}
