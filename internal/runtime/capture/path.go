package capture

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type PacketKey struct {
	Protocol        uint8
	Source          string
	Destination     string
	SourceMAC       string
	DestinationMAC  string
	SourcePort      uint16
	DestinationPort uint16
	ICMPType        uint8
	Length          int
	PayloadPrefix   []byte
}

func Fingerprint(packet PacketKey) string {
	hash := sha256.New()
	hash.Write([]byte{packet.Protocol})
	hash.Write([]byte(packet.Source))
	hash.Write([]byte{0})
	hash.Write([]byte(packet.Destination))
	var numeric [6]byte
	binary.BigEndian.PutUint16(numeric[0:2], packet.SourcePort)
	binary.BigEndian.PutUint16(numeric[2:4], packet.DestinationPort)
	binary.BigEndian.PutUint16(numeric[4:6], uint16(packet.Length))
	hash.Write(numeric[:])
	prefix := packet.PayloadPrefix
	if len(prefix) > 64 {
		prefix = prefix[:64]
	}
	hash.Write(prefix)
	return hex.EncodeToString(hash.Sum(nil)[:16])
}

type Correlator struct {
	window           time.Duration
	maximum          int
	mu               sync.Mutex
	observations     map[string]map[string]domain.TrafficObservation
	fingerprints     map[string]struct{}
	fingerprintCount int64
	matchedPackets   int64
	matchedBytes     int64
	firstMatchAt     time.Time
	lastMatchAt      time.Time
}

type CorrelationStatistics struct {
	FingerprintCount int64
	MatchedPackets   int64
	MatchedBytes     int64
	FirstMatchAt     time.Time
	LastMatchAt      time.Time
}

func NewCorrelator(window time.Duration, maximum int) *Correlator {
	if window <= 0 {
		window = 2 * time.Second
	}
	if maximum <= 0 {
		maximum = 10000
	}
	return &Correlator{window: window, maximum: maximum, observations: map[string]map[string]domain.TrafficObservation{}, fingerprints: map[string]struct{}{}}
}

func (c *Correlator) Observe(fingerprint string, interfaceID, linkID domain.ID, direction string, length int, at time.Time) {
	c.observe(fingerprint, interfaceID, linkID, "", direction, PacketKey{Length: length}, at)
}

func (c *Correlator) ObservePacket(fingerprint string, interfaceID, linkID domain.ID, direction string, packet PacketKey, at time.Time) {
	c.observe(fingerprint, interfaceID, linkID, "", direction, packet, at)
}

func (c *Correlator) ObserveNetworkObjectLinkPacket(fingerprint string, linkID domain.ID, direction string, packet PacketKey, at time.Time) {
	c.observe(fingerprint, "", "", linkID, direction, packet, at)
}

func (c *Correlator) observe(fingerprint string, interfaceID, linkID, objectLinkID domain.ID, direction string, packet PacketKey, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.fingerprints[fingerprint]; !exists {
		if c.fingerprintCount >= int64(c.maximum) {
			return
		}
		c.fingerprints[fingerprint] = struct{}{}
		c.fingerprintCount++
	}
	if c.observations[fingerprint] == nil {
		c.observations[fingerprint] = map[string]domain.TrafficObservation{}
	}
	key := string(interfaceID) + ":" + string(linkID) + ":" + string(objectLinkID) + ":" + direction
	value, exists := c.observations[fingerprint][key]
	if !exists || at.Sub(value.LastSeen) > c.window {
		value = domain.TrafficObservation{
			Fingerprint: fingerprint, InterfaceID: interfaceID, LinkID: linkID, NetworkObjectLinkID: objectLinkID, Direction: direction,
			SourceAddress: packet.Source, DestinationAddress: packet.Destination,
			SourceMAC: packet.SourceMAC, DestinationMAC: packet.DestinationMAC,
			PacketRole: packetRole(packet), FirstSeen: at,
		}
		if objectLinkID != "" {
			value.ResourceType, value.ResourceID = "network_object_link", objectLinkID
		} else if linkID != "" {
			value.ResourceType, value.ResourceID = "link", linkID
		} else {
			value.ResourceType, value.ResourceID = "interface", interfaceID
		}
	}
	value.LastSeen = at
	value.Count++
	value.Bytes += int64(packet.Length)
	c.observations[fingerprint][key] = value
	c.matchedPackets++
	c.matchedBytes += int64(packet.Length)
	if c.firstMatchAt.IsZero() || at.Before(c.firstMatchAt) {
		c.firstMatchAt = at
	}
	if c.lastMatchAt.IsZero() || at.After(c.lastMatchAt) {
		c.lastMatchAt = at
	}
}

func (c *Correlator) Statistics() CorrelationStatistics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CorrelationStatistics{
		FingerprintCount: c.fingerprintCount,
		MatchedPackets:   c.matchedPackets,
		MatchedBytes:     c.matchedBytes,
		FirstMatchAt:     c.firstMatchAt,
		LastMatchAt:      c.lastMatchAt,
	}
}

func (c *Correlator) Restore(observations []domain.TrafficObservation, statistics CorrelationStatistics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, observation := range observations {
		if c.observations[observation.Fingerprint] == nil {
			c.observations[observation.Fingerprint] = map[string]domain.TrafficObservation{}
		}
		key := string(observation.InterfaceID) + ":" + string(observation.LinkID) + ":" + string(observation.NetworkObjectLinkID) + ":" + observation.Direction
		c.observations[observation.Fingerprint][key] = observation
		if _, exists := c.fingerprints[observation.Fingerprint]; !exists {
			c.fingerprints[observation.Fingerprint] = struct{}{}
			c.fingerprintCount++
		}
		if statistics.MatchedPackets == 0 {
			c.matchedPackets += observation.Count
			c.matchedBytes += observation.Bytes
		}
		if c.firstMatchAt.IsZero() || observation.FirstSeen.Before(c.firstMatchAt) {
			c.firstMatchAt = observation.FirstSeen
		}
		if c.lastMatchAt.IsZero() || observation.LastSeen.After(c.lastMatchAt) {
			c.lastMatchAt = observation.LastSeen
		}
	}
	if statistics.FingerprintCount > c.fingerprintCount {
		c.fingerprintCount = statistics.FingerprintCount
	}
	if statistics.MatchedPackets > 0 {
		c.matchedPackets = statistics.MatchedPackets
		c.matchedBytes = statistics.MatchedBytes
	}
	if !statistics.FirstMatchAt.IsZero() {
		c.firstMatchAt = statistics.FirstMatchAt
	}
	if !statistics.LastMatchAt.IsZero() {
		c.lastMatchAt = statistics.LastMatchAt
	}
}

func packetRole(packet PacketKey) string {
	switch packet.Protocol {
	case 1:
		if packet.ICMPType == 8 {
			return "request"
		}
		if packet.ICMPType == 0 {
			return "reply"
		}
	case 58:
		if packet.ICMPType == 128 {
			return "request"
		}
		if packet.ICMPType == 129 {
			return "reply"
		}
	}
	return ""
}

func (c *Correlator) Snapshot() ([]domain.TrafficObservation, bool) {
	return c.SnapshotAt(time.Now().UTC())
}

func (c *Correlator) SnapshotAt(now time.Time) ([]domain.TrafficObservation, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	values := make([]domain.TrafficObservation, 0)
	ambiguous := false
	for fingerprint, byInterface := range c.observations {
		orders := map[string]int{}
		var path []domain.TrafficObservation
		for key, observation := range byInterface {
			if now.Sub(observation.LastSeen) > c.window {
				delete(byInterface, key)
				continue
			}
			if observation.Direction == "ambiguous" {
				ambiguous = true
			}
			values = append(values, observation)
			path = append(path, observation)
		}
		if len(byInterface) == 0 {
			delete(c.observations, fingerprint)
			continue
		}
		sort.Slice(path, func(i, j int) bool { return path[i].FirstSeen.Before(path[j].FirstSeen) })
		for index := 1; index < len(path); index++ {
			if path[index-1].FirstSeen.Equal(path[index].FirstSeen) {
				ambiguous = true
			}
		}
		for left := 0; left < len(path); left++ {
			for right := left + 1; right < len(path); right++ {
				from := observationLocation(path[left])
				to := observationLocation(path[right])
				if from == to {
					continue
				}
				if orders[to+">"+from] > 0 {
					ambiguous = true
				}
				orders[from+">"+to]++
			}
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].FirstSeen.Before(values[j].FirstSeen) })
	return values, ambiguous
}

func observationLocation(value domain.TrafficObservation) string {
	return string(value.InterfaceID) + ":" + string(value.LinkID) + ":" + string(value.NetworkObjectLinkID) + ":" + value.Direction
}
