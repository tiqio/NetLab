package capture

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

type Packet struct {
	Key PacketKey
}

type PacketDecoder struct {
	format string
	buffer []byte
	order  binary.ByteOrder
	header bool
}

func NewPacketDecoder(format string) *PacketDecoder {
	return &PacketDecoder{format: strings.ToLower(format)}
}

func (d *PacketDecoder) Add(chunk []byte) ([]Packet, error) {
	d.buffer = append(d.buffer, chunk...)
	if d.format == "pcapng" {
		return d.decodePCAPNG()
	}
	return d.decodePCAP()
}

func (d *PacketDecoder) decodePCAP() ([]Packet, error) {
	if !d.header {
		if len(d.buffer) < 24 {
			return nil, nil
		}
		magic := binary.LittleEndian.Uint32(d.buffer[:4])
		switch magic {
		case 0xa1b2c3d4, 0xa1b23c4d:
			d.order = binary.LittleEndian
		case 0xd4c3b2a1, 0x4d3cb2a1:
			d.order = binary.BigEndian
		default:
			return nil, fmt.Errorf("unsupported pcap magic %08x", magic)
		}
		d.buffer = d.buffer[24:]
		d.header = true
	}
	var packets []Packet
	for len(d.buffer) >= 16 {
		length := int(d.order.Uint32(d.buffer[8:12]))
		if length < 0 || length > 16<<20 {
			return packets, fmt.Errorf("invalid pcap packet length %d", length)
		}
		if len(d.buffer) < 16+length {
			break
		}
		if packet, ok := decodeEthernet(d.buffer[16 : 16+length]); ok {
			packets = append(packets, Packet{Key: packet})
		}
		d.buffer = d.buffer[16+length:]
	}
	return packets, nil
}

func (d *PacketDecoder) decodePCAPNG() ([]Packet, error) {
	var packets []Packet
	for len(d.buffer) >= 12 {
		length := int(binary.LittleEndian.Uint32(d.buffer[4:8]))
		if length < 12 || length > 16<<20 {
			return packets, fmt.Errorf("invalid pcapng block length %d", length)
		}
		if len(d.buffer) < length {
			break
		}
		block := d.buffer[:length]
		switch binary.LittleEndian.Uint32(block[:4]) {
		case 0x00000006:
			if length >= 32 {
				captured := int(binary.LittleEndian.Uint32(block[20:24]))
				if captured >= 0 && 28+captured <= length-4 {
					if packet, ok := decodeEthernet(block[28 : 28+captured]); ok {
						packets = append(packets, Packet{Key: packet})
					}
				}
			}
		case 0x00000003:
			if length >= 16 {
				captured := length - 16
				if packet, ok := decodeEthernet(block[12 : 12+captured]); ok {
					packets = append(packets, Packet{Key: packet})
				}
			}
		}
		d.buffer = d.buffer[length:]
	}
	return packets, nil
}

func decodeEthernet(frame []byte) (PacketKey, bool) {
	if len(frame) < 14 {
		return PacketKey{}, false
	}
	offset := 14
	sourceMAC := net.HardwareAddr(frame[6:12]).String()
	destinationMAC := net.HardwareAddr(frame[0:6]).String()
	etherType := binary.BigEndian.Uint16(frame[12:14])
	for etherType == 0x8100 || etherType == 0x88a8 {
		if len(frame) < offset+4 {
			return PacketKey{}, false
		}
		etherType = binary.BigEndian.Uint16(frame[offset+2 : offset+4])
		offset += 4
	}
	switch etherType {
	case 0x0800:
		key, ok := decodeIPv4(frame[offset:])
		key.SourceMAC, key.DestinationMAC = sourceMAC, destinationMAC
		return key, ok
	case 0x86dd:
		key, ok := decodeIPv6(frame[offset:])
		key.SourceMAC, key.DestinationMAC = sourceMAC, destinationMAC
		return key, ok
	case 0x0806:
		return PacketKey{Protocol: 0, Source: "arp", Destination: "arp", SourceMAC: sourceMAC, DestinationMAC: destinationMAC, Length: len(frame), PayloadPrefix: frame[offset:]}, true
	default:
		return PacketKey{}, false
	}
}

func decodeIPv4(packet []byte) (PacketKey, bool) {
	if len(packet) < 20 || packet[0]>>4 != 4 {
		return PacketKey{}, false
	}
	headerLength := int(packet[0]&0x0f) * 4
	if headerLength < 20 || len(packet) < headerLength {
		return PacketKey{}, false
	}
	key := PacketKey{Protocol: packet[9], Source: netip.AddrFrom4([4]byte(packet[12:16])).String(), Destination: netip.AddrFrom4([4]byte(packet[16:20])).String(), Length: len(packet)}
	decodeTransport(&key, packet[headerLength:])
	return key, true
}

func decodeIPv6(packet []byte) (PacketKey, bool) {
	if len(packet) < 40 || packet[0]>>4 != 6 {
		return PacketKey{}, false
	}
	var source, destination [16]byte
	copy(source[:], packet[8:24])
	copy(destination[:], packet[24:40])
	key := PacketKey{Protocol: packet[6], Source: netip.AddrFrom16(source).String(), Destination: netip.AddrFrom16(destination).String(), Length: len(packet)}
	decodeTransport(&key, packet[40:])
	return key, true
}

func decodeTransport(key *PacketKey, payload []byte) {
	if (key.Protocol == 1 || key.Protocol == 58) && len(payload) > 0 {
		key.ICMPType = payload[0]
	}
	if (key.Protocol == 6 || key.Protocol == 17) && len(payload) >= 4 {
		key.SourcePort = binary.BigEndian.Uint16(payload[:2])
		key.DestinationPort = binary.BigEndian.Uint16(payload[2:4])
	}
	if len(payload) > 8 && (key.Protocol == 6 || key.Protocol == 17) {
		payload = payload[8:]
	}
	key.PayloadPrefix = append([]byte(nil), payload...)
}

func Matches(match Match, packet PacketKey) bool {
	if len(match.And) > 0 {
		for _, child := range match.And {
			if !Matches(child, packet) {
				return false
			}
		}
		return true
	}
	if len(match.Or) > 0 {
		for _, child := range match.Or {
			if Matches(child, packet) {
				return true
			}
		}
		return false
	}
	if match.Not != nil {
		return !Matches(*match.Not, packet)
	}
	if match.SourceAddress != "" && !addressMatches(match.SourceAddress, packet.Source) {
		return false
	}
	if match.DestinationAddress != "" && !addressMatches(match.DestinationAddress, packet.Destination) {
		return false
	}
	protocols := map[string]uint8{"tcp": 6, "udp": 17, "icmp": 1, "icmp6": 58}
	if protocol := strings.ToLower(match.Protocol); protocol != "" {
		if expected, ok := protocols[protocol]; ok && packet.Protocol != expected {
			return false
		}
		if protocol == "ip" && strings.Contains(packet.Source, ":") {
			return false
		}
		if protocol == "ip6" && !strings.Contains(packet.Source, ":") {
			return false
		}
		if protocol == "arp" && packet.Source != "arp" {
			return false
		}
	}
	return (match.SourcePort == 0 || uint16(match.SourcePort) == packet.SourcePort) && (match.DestinationPort == 0 || uint16(match.DestinationPort) == packet.DestinationPort)
}

func addressMatches(expression, address string) bool {
	value, err := netip.ParseAddr(address)
	if err != nil {
		return false
	}
	if prefix, prefixErr := netip.ParsePrefix(expression); prefixErr == nil {
		return prefix.Contains(value)
	}
	expected, expectedErr := netip.ParseAddr(expression)
	return expectedErr == nil && expected == value
}
