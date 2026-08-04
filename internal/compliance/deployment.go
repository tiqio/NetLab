package compliance

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
)

type Listener struct {
	Address string
	Process string
}

func ValidateDeploymentAuthority(authority DeploymentAuthority) error {
	externalAuthoritative := 0
	for _, instance := range authority.Instances {
		if instance.Role == "authoritative" && instance.ExternallyReachable {
			externalAuthoritative++
		}
		if instance.ExternallyReachable && instance.Role != "authoritative" {
			return fmt.Errorf("deployment authority invariant: instance %s is externally reachable with role %s", instance.ID, instance.Role)
		}
	}
	if externalAuthoritative != 1 {
		return fmt.Errorf("deployment authority invariant: expected one external authoritative instance, got %d", externalAuthoritative)
	}
	return nil
}

func ParseSocketListeners(reader io.Reader) []Listener {
	listeners := []Listener{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "State") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		listeners = append(listeners, Listener{Address: fields[3], Process: strings.Join(fields[5:], " ")})
	}
	return listeners
}

func ValidateRuntimeAuthority(listeners []Listener, authoritativePort string) error {
	external := make([]Listener, 0, 1)
	for _, listener := range listeners {
		if !strings.Contains(strings.ToLower(listener.Process), "netlabd") || isLoopbackListener(listener.Address) {
			continue
		}
		external = append(external, listener)
	}
	if len(external) != 1 {
		return fmt.Errorf("runtime authority invariant: expected one external netlabd listener, got %d", len(external))
	}
	if listenerPort(external[0].Address) != authoritativePort {
		return fmt.Errorf("runtime authority invariant: external netlabd listens on %s instead of port %s", external[0].Address, authoritativePort)
	}
	return nil
}

func isLoopbackListener(address string) bool {
	host := strings.TrimSpace(strings.TrimSuffix(address, ":"+listenerPort(address)))
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func listenerPort(address string) string {
	index := strings.LastIndex(address, ":")
	if index < 0 || index == len(address)-1 {
		return ""
	}
	return address[index+1:]
}
