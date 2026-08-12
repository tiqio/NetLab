package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

func TestPrivilegedTenMinuteTrafficWorkloadObservation(t *testing.T) {
	if os.Getenv("NETLAB_PRIVILEGED") != "1" {
		t.Skip("set NETLAB_PRIVILEGED=1 on the acceptance host")
	}
	if os.Getenv("NETLAB_TRAFFIC_OBSERVATION_10M") != "1" {
		t.Skip("set NETLAB_TRAFFIC_OBSERVATION_10M=1 for the ten-minute gate")
	}
	if os.Geteuid() != 0 {
		t.Skip("privileged traffic observation requires root")
	}
	for _, tool := range []string{"ip", "ping", "curl", "getent", "tcpdump", "python3"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("acceptance host missing %s: %v", tool, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 11*time.Minute)
	defer cancel()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	laboratoryID := domain.ID("traffic-lab-" + suffix)
	objectA := networkObjectLinkL3Object("traffic-a-"+suffix, map[string]string{"a0": "10.62.0.1/30"})
	objectB := networkObjectLinkL3Object("traffic-b-"+suffix, map[string]string{"b0": "10.62.0.2/30"})
	link := domain.NetworkObjectLink{ID: domain.ID("traffic-link-" + suffix), LaboratoryID: laboratoryID, ObjectAID: objectA.ID, PortAName: "a0", ObjectBID: objectB.ID, PortBName: "b0"}
	for _, object := range []domain.NetworkObject{objectA, objectB} {
		namespace, _ := linuxnet.NetworkObjectNamespaceName(object)
		runObjectLinkCommand(t, ctx, "ip", "netns", "add", namespace)
		runObjectLinkCommand(t, ctx, "ip", "-n", namespace, "link", "set", "lo", "up")
		defer exec.Command("ip", "netns", "delete", namespace).Run()
	}
	dataPlane, err := linuxnet.NewDataPlane(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = dataPlane.EnsureNetworkObjectLink(ctx, link, objectA, objectB); err != nil {
		t.Fatal(err)
	}
	namespaceA, _ := linuxnet.NetworkObjectNamespaceName(objectA)
	namespaceB, _ := linuxnet.NetworkObjectNamespaceName(objectB)
	server := exec.CommandContext(ctx, "ip", "netns", "exec", namespaceB, "python3", "-m", "http.server", "18080", "--bind", "10.62.0.2")
	if err = server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Process.Kill()
	dnsScript := filepath.Join(t.TempDir(), "dns.py")
	if err = os.WriteFile(dnsScript, []byte(`import socket,struct
s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('10.62.0.2',53))
while True:
 try:
  d,a=s.recvfrom(512)
  if len(d)<17: continue
  q=d[12:]; i=0
  while i<len(q) and q[i]: i+=q[i]+1
  end=i+5
  if end>len(q): continue
  question=q[:end]; qtype=struct.unpack('!H',q[i+1:i+3])[0]
  answers=1 if qtype==1 else 0
  answer=b'\xc0\x0c'+struct.pack('!HHIH',1,1,30,4)+socket.inet_aton('10.62.0.2') if answers else b''
  s.sendto(d[:2]+b'\x81\x80'+d[4:6]+struct.pack('!H',answers)+b'\x00\x00\x00\x00'+question+answer,a)
 except Exception:
  continue
`), 0o600); err != nil {
		t.Fatal(err)
	}
	dnsServer := exec.CommandContext(ctx, "ip", "netns", "exec", namespaceB, "python3", dnsScript)
	if err = dnsServer.Start(); err != nil {
		t.Fatal(err)
	}
	defer dnsServer.Process.Kill()
	resolverDirectory := filepath.Join("/etc/netns", namespaceA)
	if err = os.MkdirAll(resolverDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(resolverDirectory, "resolv.conf"), []byte("nameserver 10.62.0.2\noptions timeout:1 attempts:1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(resolverDirectory)
	waitTrafficServiceReady(t, ctx, namespaceA, "curl", "--fail", "--silent", "--max-time", "1", "http://10.62.0.2:18080/")
	waitTrafficServiceReady(t, ctx, namespaceA, "getent", "ahostsv4", "netlab.test")
	captures := reconcile.NewCaptureManager(t.TempDir(), 12, 768<<20, time.Hour)
	captures.SetNetworkObjectRepository(objectLinkObservationRepository{links: map[domain.ID]domain.NetworkObjectLink{link.ID: link}, objects: map[domain.ID]domain.NetworkObject{objectA.ID: objectA, objectB.ID: objectB}})
	filters := reconcile.NewTrafficFilterManager(captures)
	captures.SetObserver(filters.ObserveManagedCapture)
	defer captures.StopLaboratory(laboratoryID)
	protocols := []struct {
		name        string
		match       captureRuntime.Match
		destination domain.TrafficWorkloadDestination
	}{{"icmp", captureRuntime.Match{Protocol: "icmp"}, domain.TrafficWorkloadDestination{Address: "10.62.0.2"}}, {"http", captureRuntime.Match{Protocol: "tcp"}, domain.TrafficWorkloadDestination{URL: "http://10.62.0.2:18080/"}}, {"dns", captureRuntime.Match{Protocol: "udp", DestinationPort: 53}, domain.TrafficWorkloadDestination{Name: "netlab.test"}}}
	runtime := linuxnet.NewTrafficWorkloadRuntime(nil, nil, nil)
	target := ports.TrafficWorkloadTarget{Kind: "namespace", Namespace: namespaceA}
	type observationResult struct {
		name     string
		attempts int64
		filter   domain.TrafficFilter
		err      error
	}
	results := make(chan observationResult, len(protocols))
	var wait sync.WaitGroup
	for _, protocol := range protocols {
		protocol := protocol
		filter, filterErr := filters.StartScopedAsWithObjectLinks(domain.ID("traffic-"+protocol.name+"-"+suffix), laboratoryID, protocol.match, 100000, nil, nil, []domain.ID{link.ID}, "#22c55e")
		if filterErr != nil {
			t.Fatal(filterErr)
		}
		defer filters.Stop(filter.ID)
		workload := domain.TrafficWorkload{ID: domain.ID("workload-" + protocol.name + "-" + suffix), LaboratoryID: laboratoryID, Name: protocol.name, Revision: 1, Source: domain.TrafficWorkloadEndpoint{Kind: "network_object", ResourceID: objectA.ID}, Protocol: protocol.name, AddressFamily: "ipv4", Destination: protocol.destination, IntervalSeconds: 5, TimeoutSeconds: 2, DesiredState: "running", ObservedState: "running"}
		wait.Add(1)
		go func() {
			defer wait.Done()
			var attempts int64
			var runErr error
			for attempts < 120 {
				started := time.Now()
				result, err := runtime.ExecuteTrafficWorkload(ctx, workload, target)
				if err != nil {
					runErr = fmt.Errorf("attempt %d: %w", attempts, err)
					break
				}
				if result.MatchedBytes <= 0 {
					runErr = fmt.Errorf("attempt %d returned zero bytes", attempts)
					break
				}
				attempts++
				if attempts == 120 {
					break
				}
				timer := time.NewTimer(time.Until(started.Add(5 * time.Second)))
				select {
				case <-ctx.Done():
					timer.Stop()
					runErr = ctx.Err()
				case <-timer.C:
				}
				if runErr != nil {
					break
				}
			}
			value, _, getErr := filters.Get(filter.ID)
			if runErr == nil {
				runErr = getErr
			}
			results <- observationResult{name: protocol.name, attempts: attempts, filter: value, err: runErr}
		}()
	}
	wait.Wait()
	close(results)
	for result := range results {
		if result.err != nil || result.attempts < 120 || result.filter.MatchedPackets == 0 || result.filter.MatchedBytes == 0 || result.filter.FingerprintCount == 0 {
			t.Fatalf("%s attempts=%d filter=%+v err=%v", result.name, result.attempts, result.filter, result.err)
		}
	}
}

func waitTrafficServiceReady(t *testing.T, ctx context.Context, namespace string, argv ...string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var last []byte
	for time.Now().Before(deadline) {
		command := exec.CommandContext(ctx, "ip", append([]string{"netns", "exec", namespace}, argv...)...)
		last, _ = command.CombinedOutput()
		if command.ProcessState != nil && command.ProcessState.Success() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("service did not become ready: %v: %s", argv, last)
}
