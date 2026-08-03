package capture

import (
	"testing"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func TestCompileFingerprintCorrelationAndAmbiguity(t *testing.T) {
	expression, err := Compile(Match{SourceAddress: "192.0.2.1", DestinationAddress: "2001:db8::/64", Protocol: "tcp", DestinationPort: 443})
	if err != nil || expression != "src host 192.0.2.1 and dst net 2001:db8::/64 and tcp and dst port 443" {
		t.Fatalf("expression=%q err=%v", expression, err)
	}
	if _, err = Compile(Match{Protocol: "icmp", DestinationPort: 80}); err == nil {
		t.Fatal("port without tcp/udp accepted")
	}
	fingerprint := Fingerprint(PacketKey{Protocol: 6, Source: "192.0.2.1", Destination: "192.0.2.2", SourcePort: 12345, DestinationPort: 443, Length: 64, PayloadPrefix: []byte("hello")})
	if fingerprint == "" || fingerprint != Fingerprint(PacketKey{Protocol: 6, Source: "192.0.2.1", Destination: "192.0.2.2", SourcePort: 12345, DestinationPort: 443, Length: 64, PayloadPrefix: []byte("hello")}) {
		t.Fatal("fingerprint is not stable")
	}
	correlator := NewCorrelator(time.Second, 100)
	now := time.Now().UTC()
	correlator.Observe(fingerprint, domain.ID("if-a"), domain.ID("link-1"), "egress", 64, now)
	correlator.Observe(fingerprint, domain.ID("if-b"), domain.ID("link-1"), "ingress", 64, now.Add(time.Millisecond))
	correlator.Observe(fingerprint, domain.ID("if-b"), domain.ID("link-1"), "ingress", 64, now.Add(2*time.Millisecond))
	values, ambiguous := correlator.Snapshot()
	if len(values) != 2 || ambiguous || values[1].Count != 2 {
		t.Fatalf("values=%+v ambiguous=%v", values, ambiguous)
	}
	other := Fingerprint(PacketKey{Protocol: 6, Source: "192.0.2.1", Destination: "192.0.2.2", SourcePort: 12346, DestinationPort: 443, Length: 64, PayloadPrefix: []byte("world")})
	correlator.Observe(other, domain.ID("if-a"), domain.ID("link-1"), "egress", 64, now.Add(5*time.Millisecond))
	correlator.Observe(other, domain.ID("if-b"), domain.ID("link-1"), "ingress", 64, now.Add(5*time.Millisecond))
	_, ambiguous = correlator.Snapshot()
	if !ambiguous {
		t.Fatal("equal-time hops should be reported as ambiguous")
	}
}

func TestCompileRecursiveBooleanFilter(t *testing.T) {
	expression, err := Compile(Match{And: []Match{
		{Protocol: "tcp"},
		{Or: []Match{{DestinationPort: 443, Protocol: "tcp"}, {Not: &Match{SourceAddress: "192.0.2.0/24"}}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := "(tcp) and ((tcp and dst port 443) or (not (src net 192.0.2.0/24)))"
	if expression != want {
		t.Fatalf("expression=%q want=%q", expression, want)
	}
}

func TestRejectsInvalidRecursiveBooleanFilter(t *testing.T) {
	for _, match := range []Match{
		{And: []Match{{Protocol: "tcp"}}},
		{Protocol: "tcp", Not: &Match{Protocol: "udp"}},
		{Or: []Match{{Protocol: "tcp"}, {}}},
	} {
		if _, err := Compile(match); err == nil {
			t.Fatalf("invalid expression accepted: %+v", match)
		}
	}
}

func TestCorrelatorPreservesThreeHopOrderAcrossLinks(t *testing.T) {
	correlator := NewCorrelator(time.Second, 100)
	now := time.Now().UTC()
	for flow := 0; flow < 100; flow++ {
		fingerprint := Fingerprint(PacketKey{Protocol: 17, Source: "192.0.2.1", Destination: "192.0.2.2", SourcePort: uint16(1000 + flow), DestinationPort: 53, Length: 64})
		for hop, link := range []domain.ID{"link-1", "link-2", "link-3"} {
			correlator.Observe(fingerprint, "", link, "egress", 64, now.Add(time.Duration(flow*10+hop)*time.Millisecond))
		}
	}
	values, ambiguous := correlator.Snapshot()
	if ambiguous || len(values) != 300 {
		t.Fatalf("observations=%d ambiguous=%t", len(values), ambiguous)
	}
	for _, link := range []domain.ID{"link-1", "link-2", "link-3"} {
		found := false
		for _, value := range values {
			if value.LinkID == link {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %s", link)
		}
	}
}

func TestCorrelatorDoesNotMarkNormalReverseFlowAmbiguous(t *testing.T) {
	correlator := NewCorrelator(time.Second, 100)
	now := time.Now().UTC()
	forward := Fingerprint(PacketKey{Protocol: 1, Source: "192.0.2.1", Destination: "192.0.2.2", Length: 64, PayloadPrefix: []byte("request")})
	reverse := Fingerprint(PacketKey{Protocol: 1, Source: "192.0.2.2", Destination: "192.0.2.1", Length: 64, PayloadPrefix: []byte("response")})
	correlator.Observe(forward, "if-a", "", "observed", 64, now)
	correlator.Observe(forward, "if-b", "", "observed", 64, now.Add(time.Millisecond))
	correlator.Observe(reverse, "if-b", "", "observed", 64, now.Add(2*time.Millisecond))
	correlator.Observe(reverse, "if-a", "", "observed", 64, now.Add(3*time.Millisecond))
	values, ambiguous := correlator.Snapshot()
	if ambiguous || len(values) != 4 {
		t.Fatalf("observations=%d ambiguous=%t", len(values), ambiguous)
	}
}

func TestCorrelatorPreservesICMPEchoRole(t *testing.T) {
	correlator := NewCorrelator(time.Second, 100)
	now := time.Now().UTC()
	correlator.ObservePacket("request", "", "link-a", "observed", PacketKey{Protocol: 1, ICMPType: 8, Source: "192.0.2.1", Destination: "192.0.2.2"}, now)
	correlator.ObservePacket("reply", "", "link-a", "observed", PacketKey{Protocol: 1, ICMPType: 0, Source: "192.0.2.2", Destination: "192.0.2.1"}, now.Add(time.Millisecond))
	values, _ := correlator.Snapshot()
	roles := map[string]bool{}
	for _, value := range values {
		roles[value.PacketRole] = true
	}
	if !roles["request"] || !roles["reply"] {
		t.Fatalf("observations=%+v", values)
	}
}
