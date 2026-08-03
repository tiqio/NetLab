package console

import (
	"fmt"
	"net"
)

type VNCDescriptor struct {
	Mode        string `json:"mode"`
	StreamURL   string `json:"stream_url"`
	Subprotocol string `json:"subprotocol"`
}

func DescribeVNC(nodeID string) VNCDescriptor {
	return VNCDescriptor{Mode: "vnc", StreamURL: fmt.Sprintf("/api/v1/nodes/%s/consoles/vnc/stream", nodeID), Subprotocol: "binary"}
}

func DialVNC(socketPath string) (net.Conn, error) { return net.Dial("unix", socketPath) }
