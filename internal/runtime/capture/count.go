package capture

import "encoding/binary"

type packetCounter struct {
	format string
	buffer []byte
	order  binary.ByteOrder
	header bool
}

func newPacketCounter(format string) *packetCounter { return &packetCounter{format: format} }

func (c *packetCounter) Add(chunk []byte) int64 {
	c.buffer = append(c.buffer, chunk...)
	if c.format == "pcapng" {
		return c.countPCAPNG()
	}
	return c.countPCAP()
}

func (c *packetCounter) countPCAP() int64 {
	if !c.header {
		if len(c.buffer) < 24 {
			return 0
		}
		switch binary.LittleEndian.Uint32(c.buffer[:4]) {
		case 0xa1b2c3d4, 0xa1b23c4d:
			c.order = binary.LittleEndian
		default:
			c.order = binary.BigEndian
		}
		c.buffer = c.buffer[24:]
		c.header = true
	}
	var count int64
	for len(c.buffer) >= 16 {
		length := int(c.order.Uint32(c.buffer[8:12]))
		if length < 0 || len(c.buffer) < 16+length {
			break
		}
		c.buffer = c.buffer[16+length:]
		count++
	}
	return count
}

func (c *packetCounter) countPCAPNG() int64 {
	var count int64
	for len(c.buffer) >= 12 {
		blockType := binary.LittleEndian.Uint32(c.buffer[:4])
		length := int(binary.LittleEndian.Uint32(c.buffer[4:8]))
		if length < 12 || len(c.buffer) < length {
			break
		}
		if blockType == 0x00000006 || blockType == 0x00000003 {
			count++
		}
		c.buffer = c.buffer[length:]
	}
	return count
}
