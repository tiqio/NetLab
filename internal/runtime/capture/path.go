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
	window       time.Duration
	maximum      int
	mu           sync.Mutex
	observations map[string]map[string]domain.TrafficObservation
}

func NewCorrelator(window time.Duration, maximum int) *Correlator {
	if window <= 0 {
		window = 2 * time.Second
	}
	if maximum <= 0 {
		maximum = 10000
	}
	return &Correlator{window: window, maximum: maximum, observations: map[string]map[string]domain.TrafficObservation{}}
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
	if c.observations[fingerprint] == nil && len(c.observations) >= c.maximum {
		return
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
	c.mu.Lock()
	defer c.mu.Unlock()
	values := make([]domain.TrafficObservation, 0)
	ambiguous := false
	for _, byInterface := range c.observations {
		orders := map[string]int{}
		var path []domain.TrafficObservation
		for _, observation := range byInterface {
			values = append(values, observation)
			path = append(path, observation)
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
