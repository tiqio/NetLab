package capture

import (
	"encoding/binary"
	"testing"
)

func TestPacketDecoderStreamsAndMatchesIPv4TCP(t *testing.T) {
	frame := make([]byte, 14+20+20)
	copy(frame[0:6], []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x02})
	copy(frame[6:12], []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x01})
	binary.BigEndian.PutUint16(frame[12:14], 0x0800)
	ip := frame[14:]
	ip[0], ip[9] = 0x45, 6
	copy(ip[12:16], []byte{192, 0, 2, 1})
	copy(ip[16:20], []byte{198, 51, 100, 2})
	binary.BigEndian.PutUint16(ip[20:22], 12345)
	binary.BigEndian.PutUint16(ip[22:24], 443)
	pcap := make([]byte, 24+16+len(frame))
	binary.LittleEndian.PutUint32(pcap[:4], 0xa1b2c3d4)
	binary.LittleEndian.PutUint32(pcap[24+8:24+12], uint32(len(frame)))
	copy(pcap[40:], frame)
	decoder := NewPacketDecoder("pcap")
	packets, err := decoder.Add(pcap[:30])
	if err != nil || len(packets) != 0 {
		t.Fatalf("partial packets=%d err=%v", len(packets), err)
	}
	packets, err = decoder.Add(pcap[30:])
	if err != nil || len(packets) != 1 {
		t.Fatalf("packets=%d err=%v", len(packets), err)
	}
	key := packets[0].Key
	if key.SourceMAC != "02:00:00:00:00:01" || key.DestinationMAC != "02:00:00:00:00:02" {
		t.Fatalf("unexpected ethernet direction: %+v", key)
	}
	if !Matches(Match{SourceAddress: "192.0.2.0/24", Protocol: "tcp", DestinationPort: 443}, key) {
		t.Fatalf("packet did not match: %+v", key)
	}
	if Matches(Match{Protocol: "udp"}, key) {
		t.Fatal("tcp packet matched udp")
	}
	if !Matches(Match{Or: []Match{{Protocol: "tcp", SourcePort: 12345}, {Protocol: "tcp", DestinationPort: 12345}}}, key) {
		t.Fatal("recursive or expression did not match")
	}
	if Matches(Match{Not: &Match{Protocol: "tcp"}}, key) {
		t.Fatal("recursive not expression matched")
	}
}

func TestICMPEchoRoles(t *testing.T) {
	for _, test := range []struct {
		protocol uint8
		typeCode uint8
		role     string
	}{
		{protocol: 1, typeCode: 8, role: "request"},
		{protocol: 1, typeCode: 0, role: "reply"},
		{protocol: 58, typeCode: 128, role: "request"},
		{protocol: 58, typeCode: 129, role: "reply"},
	} {
		key := PacketKey{Protocol: test.protocol}
		decodeTransport(&key, []byte{test.typeCode, 0, 0, 0})
		if key.ICMPType != test.typeCode || packetRole(key) != test.role {
			t.Fatalf("key=%+v role=%q", key, packetRole(key))
		}
	}
}
