package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	for _, tool := range []string{"ip", "ping", "curl", "nslookup", "tcpdump", "python3"} {
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
	runObjectLinkCommand(t, ctx, "ip", "-n", namespaceA, "address", "add", "fd62::1/64", "dev", "a0", "nodad")
	runObjectLinkCommand(t, ctx, "ip", "-n", namespaceB, "address", "add", "fd62::2/64", "dev", "b0", "nodad")
	httpIPv6Script := filepath.Join(t.TempDir(), "http6.py")
	if err = os.WriteFile(httpIPv6Script, []byte(`import http.server,socket
class Server(http.server.ThreadingHTTPServer): address_family=socket.AF_INET6
Server(('fd62::2',18081),http.server.SimpleHTTPRequestHandler).serve_forever()
`), 0o600); err != nil {
		t.Fatal(err)
	}
	servers := []*exec.Cmd{
		exec.CommandContext(ctx, "ip", "netns", "exec", namespaceB, "python3", "-m", "http.server", "18080", "--bind", "10.62.0.2"),
		exec.CommandContext(ctx, "ip", "netns", "exec", namespaceB, "python3", httpIPv6Script),
	}
	for _, server := range servers {
		if err = server.Start(); err != nil {
			t.Fatal(err)
		}
		defer server.Process.Kill()
	}
	dnsScript := filepath.Join(t.TempDir(), "dns.py")
	if err = os.WriteFile(dnsScript, []byte(`import socket,struct,sys
v6=sys.argv[1]=='6'; family=socket.AF_INET6 if v6 else socket.AF_INET
s=socket.socket(family,socket.SOCK_DGRAM);s.bind(('fd62::2' if v6 else '10.62.0.2',53))
while True:
 try:
  d,a=s.recvfrom(512)
  if len(d)<17: continue
  q=d[12:]; i=0
  while i<len(q) and q[i]: i+=q[i]+1
  end=i+5
  if end>len(q): continue
  question=q[:end]; qtype=struct.unpack('!H',q[i+1:i+3])[0]
  expected=28 if v6 else 1; answers=1 if qtype==expected else 0
  payload=socket.inet_pton(family,'fd62::2' if v6 else '10.62.0.2')
  answer=b'\xc0\x0c'+struct.pack('!HHIH',expected,1,30,len(payload))+payload if answers else b''
  s.sendto(d[:2]+b'\x81\x80'+d[4:6]+struct.pack('!H',answers)+b'\x00\x00\x00\x00'+question+answer,a)
 except Exception:
  continue
`), 0o600); err != nil {
		t.Fatal(err)
	}
	dnsServers := []*exec.Cmd{
		exec.CommandContext(ctx, "ip", "netns", "exec", namespaceB, "python3", dnsScript, "4"),
		exec.CommandContext(ctx, "ip", "netns", "exec", namespaceB, "python3", dnsScript, "6"),
	}
	for _, dnsServer := range dnsServers {
		if err = dnsServer.Start(); err != nil {
			t.Fatal(err)
		}
		defer dnsServer.Process.Kill()
	}
	resolverDirectory := filepath.Join("/etc/netns", namespaceA)
	if err = os.MkdirAll(resolverDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(resolverDirectory, "resolv.conf"), []byte("nameserver 10.62.0.2\nnameserver fd62::2\noptions timeout:1 attempts:1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(resolverDirectory)
	waitTrafficServiceReady(t, ctx, namespaceA, "curl", "--noproxy", "*", "--fail", "--silent", "--max-time", "1", "http://10.62.0.2:18080/")
	waitTrafficServiceReady(t, ctx, namespaceA, "curl", "-6", "--noproxy", "*", "--fail", "--silent", "--max-time", "1", "http://[fd62::2]:18081/")
	waitTrafficServiceReady(t, ctx, namespaceA, "nslookup", "netlab.test", "10.62.0.2")
	waitTrafficServiceReady(t, ctx, namespaceA, "nslookup", "netlab.test", "fd62::2")
	captures := reconcile.NewCaptureManager(t.TempDir(), 12, 768<<20, time.Hour)
	captures.SetNetworkObjectRepository(objectLinkObservationRepository{links: map[domain.ID]domain.NetworkObjectLink{link.ID: link}, objects: map[domain.ID]domain.NetworkObject{objectA.ID: objectA, objectB.ID: objectB}})
	filters := reconcile.NewTrafficFilterManager(captures)
	captures.SetObserver(filters.ObserveManagedCapture)
	defer captures.StopLaboratory(laboratoryID)
	protocols := []struct {
		name          string
		protocol      string
		addressFamily string
		match         captureRuntime.Match
		destination   domain.TrafficWorkloadDestination
	}{
		{"icmp-ipv4", "icmp", "ipv4", captureRuntime.Match{Protocol: "icmp", DestinationAddress: "10.62.0.2"}, domain.TrafficWorkloadDestination{Address: "10.62.0.2"}},
		{"http-ipv4", "http", "ipv4", captureRuntime.Match{Protocol: "tcp", DestinationAddress: "10.62.0.2", DestinationPort: 18080}, domain.TrafficWorkloadDestination{URL: "http://10.62.0.2:18080/"}},
		{"dns-ipv4", "dns", "ipv4", captureRuntime.Match{Protocol: "udp", DestinationAddress: "10.62.0.2", DestinationPort: 53}, domain.TrafficWorkloadDestination{Name: "netlab.test", Address: "10.62.0.2"}},
		{"icmp-ipv6", "icmp", "ipv6", captureRuntime.Match{Protocol: "icmp6", DestinationAddress: "fd62::2"}, domain.TrafficWorkloadDestination{Address: "fd62::2"}},
		{"http-ipv6", "http", "ipv6", captureRuntime.Match{Protocol: "tcp", DestinationAddress: "fd62::2", DestinationPort: 18081}, domain.TrafficWorkloadDestination{URL: "http://[fd62::2]:18081/"}},
		{"dns-ipv6", "dns", "ipv6", captureRuntime.Match{Protocol: "udp", DestinationAddress: "fd62::2", DestinationPort: 53}, domain.TrafficWorkloadDestination{Name: "netlab.test", Address: "fd62::2"}},
	}
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
		workload := domain.TrafficWorkload{ID: domain.ID("workload-" + protocol.name + "-" + suffix), LaboratoryID: laboratoryID, Name: protocol.name, Revision: 1, Source: domain.TrafficWorkloadEndpoint{Kind: "network_object", ResourceID: objectA.ID}, Protocol: protocol.protocol, AddressFamily: protocol.addressFamily, Destination: protocol.destination, IntervalSeconds: 5, TimeoutSeconds: 2, DesiredState: "running", ObservedState: "running"}
		wait.Add(1)
		go func() {
			defer wait.Done()
			var attempts int64
			var runErr error
			var previousPackets, previousBytes int64
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
				if attempts%2 == 0 {
					value, windowErr := waitForTrafficFilterGrowth(ctx, filters, filter.ID, previousPackets, previousBytes, 2*time.Second)
					if windowErr != nil {
						runErr = fmt.Errorf("10-second window ending at attempt %d: %w", attempts, windowErr)
						break
					}
					previousPackets, previousBytes = value.MatchedPackets, value.MatchedBytes
				}
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
		for _, observation := range result.filter.Observations {
			if strings.Contains(result.name, "ipv6") != strings.Contains(observation.SourceAddress, ":") {
				t.Fatalf("%s fingerprint %s has wrong address family: %+v", result.name, observation.Fingerprint, observation)
			}
		}
	}
}

func waitForTrafficFilterGrowth(ctx context.Context, filters *reconcile.TrafficFilterManager, id domain.ID, previousPackets, previousBytes int64, timeout time.Duration) (domain.TrafficFilter, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		value, _, err := filters.Get(id)
		if err == nil && value.MatchedPackets > previousPackets && value.MatchedBytes > previousBytes {
			return value, nil
		}
		select {
		case <-ctx.Done():
			return domain.TrafficFilter{}, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
	value, _, err := filters.Get(id)
	if err != nil {
		return domain.TrafficFilter{}, err
	}
	return value, fmt.Errorf("counters did not grow from packets=%d bytes=%d; current packets=%d bytes=%d", previousPackets, previousBytes, value.MatchedPackets, value.MatchedBytes)
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
