package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type objectLinkObservationRepository struct {
	links   map[domain.ID]domain.NetworkObjectLink
	objects map[domain.ID]domain.NetworkObject
}

func (r objectLinkObservationRepository) GetNetworkObjectLink(_ context.Context, id domain.ID) (domain.NetworkObjectLink, error) {
	value, ok := r.links[id]
	if !ok {
		return domain.NetworkObjectLink{}, fmt.Errorf("network object link %s not found", id)
	}
	return value, nil
}

func (r objectLinkObservationRepository) GetNetworkObject(_ context.Context, id domain.ID) (domain.NetworkObject, error) {
	value, ok := r.objects[id]
	if !ok {
		return domain.NetworkObject{}, fmt.Errorf("network object %s not found", id)
	}
	return value, nil
}

func TestPrivilegedNetworkObjectLinkCaptureAndTrafficAttribution(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED") != "1" {
		t.Skip("set NETLAB_PRIVILEGED=1 on the acceptance host")
	}
	if os.Geteuid() != 0 {
		t.Skip("NETLAB_PRIVILEGED requires root")
	}
	for _, tool := range []string{"ip", "ping", "python3", "tcpdump"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 7 {
		suffix = suffix[len(suffix)-7:]
	}
	laboratoryID := domain.ID("observation-lab-" + suffix)
	objectA := networkObjectLinkL3Object("observation-a-"+suffix, map[string]string{
		"a0": "10.61.0.1/30",
		"a1": "10.61.1.1/30",
	})
	objectB := networkObjectLinkL3Object("observation-b-"+suffix, map[string]string{
		"b0": "10.61.0.2/30",
		"b1": "10.61.1.2/30",
	})
	primary := domain.NetworkObjectLink{ID: domain.ID("observation-primary-" + suffix), LaboratoryID: laboratoryID, ObjectAID: objectA.ID, PortAName: "a0", ObjectBID: objectB.ID, PortBName: "b0"}
	parallel := domain.NetworkObjectLink{ID: domain.ID("observation-parallel-" + suffix), LaboratoryID: laboratoryID, ObjectAID: objectA.ID, PortAName: "a1", ObjectBID: objectB.ID, PortBName: "b1"}
	objects := []domain.NetworkObject{objectA, objectB}
	for _, object := range objects {
		namespace, err := linuxnet.NetworkObjectNamespaceName(object)
		if err != nil {
			t.Fatal(err)
		}
		runObjectLinkCommand(t, ctx, "ip", "netns", "add", namespace)
		runObjectLinkCommand(t, ctx, "ip", "-n", namespace, "link", "set", "lo", "up")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		for _, object := range objects {
			namespace, _ := linuxnet.NetworkObjectNamespaceName(object)
			_ = exec.CommandContext(cleanupCtx, "ip", "netns", "delete", namespace).Run()
		}
	})

	dataPlane, err := linuxnet.NewDataPlane(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range []domain.NetworkObjectLink{primary, parallel} {
		if err = dataPlane.EnsureNetworkObjectLink(ctx, link, objectA, objectB); err != nil {
			t.Fatalf("ensure %s: %v", link.ID, err)
		}
	}
	namespaceA, _ := linuxnet.NetworkObjectNamespaceName(objectA)
	namespaceB, _ := linuxnet.NetworkObjectNamespaceName(objectB)
	assertObjectLinkPing(t, ctx, namespaceA, "10.61.0.2")
	assertObjectLinkPing(t, ctx, namespaceA, "10.61.1.2")

	repository := objectLinkObservationRepository{
		links: map[domain.ID]domain.NetworkObjectLink{
			primary.ID:  primary,
			parallel.ID: parallel,
		},
		objects: map[domain.ID]domain.NetworkObject{
			objectA.ID: objectA,
			objectB.ID: objectB,
		},
	}
	captures := reconcile.NewCaptureManager(t.TempDir(), 12, 512<<20, time.Hour)
	captures.SetNetworkObjectRepository(repository)
	t.Cleanup(func() { captures.StopLaboratory(laboratoryID) })

	directCapture, err := captures.Start(ctx, reconcile.CaptureRequest{
		LaboratoryID: laboratoryID,
		SourceType:   "network_object_link",
		SourceID:     primary.ID,
		Filter:       "icmp",
		Format:       "pcap",
		MaxBytes:     4 << 20,
		Duration:     20 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	runObjectLinkCommand(t, ctx, "ip", "netns", "exec", namespaceA, "ping", "-c", "1", "-W", "1", "10.61.0.2")
	waitForObjectLinkCapturePackets(t, captures, directCapture.ID, 2, time.Second)
	stoppedCapture, err := captures.Stop(directCapture.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stoppedCapture.SourceType != "network_object_link" || stoppedCapture.SourceID != primary.ID {
		t.Fatalf("capture source=%s/%s", stoppedCapture.SourceType, stoppedCapture.SourceID)
	}
	if stoppedCapture.Packets != 2 {
		t.Fatalf("one-endpoint ICMP capture counted %d packets, want exactly request and reply", stoppedCapture.Packets)
	}

	filters := reconcile.NewTrafficFilterManager(captures)
	captures.SetObserver(filters.ObserveManagedCapture)
	protocolFilters := map[string]domain.TrafficFilter{}
	for _, protocol := range []string{"icmp", "tcp", "udp"} {
		value, startErr := filters.StartScopedAsWithObjectLinks(
			domain.ID("observation-"+protocol+"-"+suffix),
			laboratoryID,
			captureRuntime.Match{Protocol: protocol},
			1000,
			nil,
			nil,
			[]domain.ID{primary.ID},
			"#22c55e",
		)
		if startErr != nil {
			t.Fatalf("start %s filter: %v", protocol, startErr)
		}
		protocolFilters[protocol] = value
		filterID := value.ID
		t.Cleanup(func() { _, _ = filters.Stop(filterID) })
	}
	time.Sleep(200 * time.Millisecond)

	runObjectLinkCommand(t, ctx, "ip", "netns", "exec", namespaceA, "ping", "-c", "1", "-W", "1", "10.61.0.2")
	assertObjectLinkAttribution(t, captures, filters, protocolFilters["icmp"].ID, primary.ID, parallel.ID, time.Now())
	assertObjectLinkSocket(t, ctx, "tcp", namespaceA, namespaceB, "10.61.0.2", 18601)
	assertObjectLinkAttribution(t, captures, filters, protocolFilters["tcp"].ID, primary.ID, parallel.ID, time.Now())
	assertObjectLinkSocket(t, ctx, "udp", namespaceB, namespaceA, "10.61.0.1", 18602)
	assertObjectLinkAttribution(t, captures, filters, protocolFilters["udp"].ID, primary.ID, parallel.ID, time.Now())
}

func waitForObjectLinkCapturePackets(t *testing.T, captures *reconcile.CaptureManager, id domain.ID, minimum int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		value, err := captures.Get(id)
		if err == nil && value.Packets >= minimum {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	value, err := captures.Get(id)
	t.Fatalf("capture %s packets=%d err=%v after %s", id, value.Packets, err, timeout)
}

func assertObjectLinkAttribution(t *testing.T, captures *reconcile.CaptureManager, filters *reconcile.TrafficFilterManager, filterID, primaryID, unusedParallelID domain.ID, trafficCompletedAt time.Time) {
	t.Helper()
	deadline := trafficCompletedAt.Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		value, ambiguous, err := filters.Get(filterID)
		if err == nil && !ambiguous {
			directions := map[string]bool{}
			isolated := true
			for _, observation := range value.Observations {
				if observation.NetworkObjectLinkID == unusedParallelID || observation.ResourceID == unusedParallelID {
					isolated = false
				}
				if observation.NetworkObjectLinkID != primaryID || observation.ResourceType != "network_object_link" || observation.ResourceID != primaryID {
					continue
				}
				directions[observation.Direction] = true
			}
			if isolated && directions["a_to_b"] && directions["b_to_a"] {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	value, ambiguous, err := filters.Get(filterID)
	statistics := make([]domain.Capture, 0)
	for _, capture := range captures.List() {
		current, getErr := captures.Get(capture.ID)
		if getErr == nil {
			statistics = append(statistics, current)
		}
	}
	t.Fatalf("filter %s was not attributed within 500ms: observations=%+v ambiguous=%v err=%v captures=%+v", filterID, value.Observations, ambiguous, err, statistics)
}
