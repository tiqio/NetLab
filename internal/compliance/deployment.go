package compliance

import (
	"bufio"
	"fmt"
	"io"
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
