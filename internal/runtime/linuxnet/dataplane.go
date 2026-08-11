package linuxnet

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type DataPlane struct {
	executor CommandExecutor
	ip       string
}

func NewDataPlane(executor CommandExecutor) (*DataPlane, error) {
	if executor != nil {
		return &DataPlane{executor: executor, ip: "ip"}, nil
	}
	path, err := lookup("ip")
	if err != nil {
		return nil, err
	}
	return &DataPlane{executor: SystemExecutor{}, ip: path}, nil
}

func (d *DataPlane) EnsureLink(ctx context.Context, link domain.Link, endpointA, endpointB domain.Interface) error {
	bridge := LinkBridgeName(link.ID)
	if endpointA.ID == "" || endpointB.ID == "" {
		return fmt.Errorf("link endpoints are unavailable")
	}
	if err := d.executor.Run(ctx, d.ip, "link", "add", bridge, "type", "bridge"); err != nil {
		if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", bridge); inspectErr != nil {
			return err
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", bridge, "alias", ownership.Marker("netlab", string(link.ID)))
	if err := d.executor.Run(ctx, d.ip, "link", "set", bridge, "up"); err != nil {
		return err
	}
	for _, iface := range []domain.Interface{endpointA, endpointB} {
		if err := d.executor.Run(ctx, d.ip, "link", "set", HostInterfaceName(iface.ID), "master", bridge); err != nil {
			_ = d.executor.Run(ctx, d.ip, "link", "delete", bridge)
			return err
		}
	}
	return nil
}

func (d *DataPlane) DeleteLink(ctx context.Context, id domain.ID) error {
	err := d.executor.Run(ctx, d.ip, "link", "delete", LinkBridgeName(id))
	if err != nil && (strings.Contains(err.Error(), "Cannot find device") || strings.Contains(err.Error(), "does not exist")) {
		return nil
	}
	return err
}

func (d *DataPlane) EnsureNetworkObjectLink(ctx context.Context, link domain.NetworkObjectLink, objectA, objectB domain.NetworkObject) error {
	if link.ObjectAID == "" || link.ObjectBID == "" || link.PortAName == "" || link.PortBName == "" {
		return fmt.Errorf("network object link endpoints are incomplete")
	}
	endA := ownership.Name("nva", link.ID, 15)
	endB := ownership.Name("nvb", link.ID, 15)
	endpointA, err := networkObjectLinkEndpoint(objectA, link.PortAName, endA)
	if err != nil {
		return err
	}
	endpointB, err := networkObjectLinkEndpoint(objectB, link.PortBName, endB)
	if err != nil {
		return err
	}
	_, errA := d.executor.Output(ctx, d.ip, endpointA.inspectArgs()...)
	_, errB := d.executor.Output(ctx, d.ip, endpointB.inspectArgs()...)
	if errA == nil && errB == nil {
		if err := d.prepareNetworkObjectLinkEndpoint(ctx, endpointA); err != nil {
			return err
		}
		if err := d.prepareNetworkObjectLinkEndpoint(ctx, endpointB); err != nil {
			return err
		}
		if err := d.configureNetworkObjectLinkEndpoint(ctx, endpointA); err != nil {
			return err
		}
		return d.configureNetworkObjectLinkEndpoint(ctx, endpointB)
	}
	if errA == nil {
		if err := d.executor.Run(ctx, d.ip, endpointA.deleteArgs()...); err != nil && !missingLinkError(err) {
			return err
		}
	} else if errB == nil {
		if err := d.executor.Run(ctx, d.ip, endpointB.deleteArgs()...); err != nil && !missingLinkError(err) {
			return err
		}
	}
	_, hostAErr := d.executor.Output(ctx, d.ip, "link", "show", endA)
	_, hostBErr := d.executor.Output(ctx, d.ip, "link", "show", endB)
	if (hostAErr == nil) != (hostBErr == nil) {
		stale := endA
		if hostAErr != nil {
			stale = endB
		}
		if err := d.executor.Run(ctx, d.ip, "link", "delete", stale); err != nil && !missingLinkError(err) {
			return err
		}
	}
	if err = d.executor.Run(ctx, d.ip, "link", "add", endA, "type", "veth", "peer", "name", endB); err != nil {
		if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", endA); inspectErr != nil {
			return err
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", endA, "alias", ownership.Marker("netlab", string(link.ID)+":a"))
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", endB, "alias", ownership.Marker("netlab", string(link.ID)+":b"))
	if err = d.placeNetworkObjectLinkEndpoint(ctx, endpointA); err != nil {
		return err
	}
	if err = d.placeNetworkObjectLinkEndpoint(ctx, endpointB); err != nil {
		return err
	}
	if err = d.prepareNetworkObjectLinkEndpoint(ctx, endpointA); err != nil {
		return err
	}
	if err = d.prepareNetworkObjectLinkEndpoint(ctx, endpointB); err != nil {
		return err
	}
	if err = d.configureNetworkObjectLinkEndpoint(ctx, endpointA); err != nil {
		return err
	}
	return d.configureNetworkObjectLinkEndpoint(ctx, endpointB)
}

type networkObjectLinkEndpointSpec struct {
	object      domain.NetworkObject
	portName    string
	runtimeName string
	namespace   string
	hostBridge  string
}

func networkObjectLinkEndpoint(object domain.NetworkObject, portName, runtimeName string) (networkObjectLinkEndpointSpec, error) {
	value := networkObjectLinkEndpointSpec{object: object, portName: portName, runtimeName: runtimeName}
	switch object.Kind {
	case domain.NetworkBridge, domain.NetworkNAT:
		bridge, err := NetworkBridgeName(object)
		if err != nil {
			return networkObjectLinkEndpointSpec{}, err
		}
		value.hostBridge = bridge
	default:
		namespace, err := networkObjectNamespace(object)
		if err != nil {
			return networkObjectLinkEndpointSpec{}, err
		}
		value.namespace = namespace
	}
	return value, nil
}

func (e networkObjectLinkEndpointSpec) inspectArgs() []string {
	if e.namespace == "" {
		return []string{"link", "show", e.runtimeName}
	}
	return []string{"-n", e.namespace, "link", "show", e.portName}
}

func (e networkObjectLinkEndpointSpec) deleteArgs() []string {
	if e.namespace == "" {
		return []string{"link", "delete", e.runtimeName}
	}
	return []string{"-n", e.namespace, "link", "delete", e.portName}
}

func (d *DataPlane) placeNetworkObjectLinkEndpoint(ctx context.Context, endpoint networkObjectLinkEndpointSpec) error {
	if endpoint.namespace == "" {
		return nil
	}
	if err := d.executor.Run(ctx, d.ip, "link", "set", endpoint.runtimeName, "netns", endpoint.namespace); err != nil {
		return err
	}
	return d.executor.Run(ctx, d.ip, "-n", endpoint.namespace, "link", "set", endpoint.runtimeName, "name", endpoint.portName)
}

func (d *DataPlane) prepareNetworkObjectLinkEndpoint(ctx context.Context, endpoint networkObjectLinkEndpointSpec) error {
	if endpoint.namespace == "" {
		if err := d.executor.Run(ctx, d.ip, "link", "set", endpoint.runtimeName, "master", endpoint.hostBridge); err != nil {
			return err
		}
		return d.executor.Run(ctx, d.ip, "link", "set", endpoint.runtimeName, "up")
	}
	return d.prepareNetworkObjectLinkPort(ctx, endpoint.namespace, endpoint.portName, endpoint.object)
}

func (d *DataPlane) configureNetworkObjectLinkEndpoint(ctx context.Context, endpoint networkObjectLinkEndpointSpec) error {
	if endpoint.namespace == "" {
		return nil
	}
	return d.configureNamespacePort(ctx, endpoint.namespace, endpoint.portName, endpoint.object)
}

func (d *DataPlane) prepareNetworkObjectLinkPort(ctx context.Context, namespace, portName string, object domain.NetworkObject) error {
	if object.Kind == domain.NetworkSwitchL2 {
		if err := d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", portName, "master", "br0"); err != nil {
			return err
		}
		if err := d.configureSwitchL2Port(ctx, namespace, portName, object); err != nil {
			return err
		}
	}
	if err := d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", portName, "up"); err != nil {
		return err
	}
	return nil
}

func networkObjectNamespace(object domain.NetworkObject) (string, error) {
	switch object.Kind {
	case domain.NetworkPC:
		return ownership.Name("nlpc", object.ID, 15), nil
	case domain.NetworkSwitchL2:
		return SwitchL2NamespaceName(object.ID), nil
	case domain.NetworkSwitchL3:
		return SwitchL3NamespaceName(object.ID), nil
	default:
		return "", fmt.Errorf("network object %s is not namespace-backed", object.Kind)
	}
}

func NetworkObjectNamespaceName(object domain.NetworkObject) (string, error) {
	return networkObjectNamespace(object)
}

func (d *DataPlane) configureSwitchL2Port(ctx context.Context, namespace, portName string, object domain.NetworkObject) error {
	var config domain.SwitchL2Config
	if err := decodeConfig(object.Config, &config); err != nil {
		return err
	}
	for _, port := range config.Ports {
		if port.Name != portName {
			continue
		}
		if port.PVID > 0 {
			if err := d.executor.Run(ctx, "bridge", "-n", namespace, "vlan", "add", "dev", portName, "vid", fmt.Sprint(port.PVID), "pvid", "untagged"); err != nil {
				return err
			}
		}
		for _, vlan := range port.Tagged {
			if err := d.executor.Run(ctx, "bridge", "-n", namespace, "vlan", "add", "dev", portName, "vid", fmt.Sprint(vlan)); err != nil {
				return err
			}
		}
		break
	}
	return nil
}

func (d *DataPlane) Attach(ctx context.Context, iface domain.Interface, object domain.NetworkObject) error {
	bridge, err := NetworkBridgeName(object)
	if err != nil {
		return err
	}
	return d.executor.Run(ctx, d.ip, "link", "set", HostInterfaceName(iface.ID), "master", bridge)
}

func (d *DataPlane) AttachNamespace(ctx context.Context, attachment domain.NetworkAttachment, iface domain.Interface, object domain.NetworkObject) error {
	namespace, err := networkObjectNamespace(object)
	if err != nil {
		return err
	}
	bridge := ownership.Name("nla", attachment.ID, 15)
	host := ownership.Name("nah", attachment.ID, 15)
	peer := ownership.Name("nap", attachment.ID, 15)
	portName := attachment.PortName
	if portName == "" {
		portName = "eth0"
	}
	_, hostErr := d.executor.Output(ctx, d.ip, "link", "show", host)
	_, portErr := d.executor.Output(ctx, d.ip, "-n", namespace, "link", "show", portName)
	if (hostErr == nil) != (portErr == nil) {
		if hostErr == nil {
			if err := d.executor.Run(ctx, d.ip, "link", "delete", host); err != nil && !missingLinkError(err) {
				return err
			}
		} else if err := d.executor.Run(ctx, d.ip, "-n", namespace, "link", "delete", portName); err != nil && !missingLinkError(err) {
			return err
		}
		hostErr, portErr = errors.New("missing"), errors.New("missing")
	}
	if err := d.executor.Run(ctx, d.ip, "link", "add", bridge, "type", "bridge"); err != nil {
		if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", bridge); inspectErr != nil {
			return err
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", bridge, "alias", ownership.Marker("netlab", string(attachment.ID)))
	if err := d.executor.Run(ctx, d.ip, "link", "set", bridge, "up"); err != nil {
		return err
	}
	if hostErr != nil {
		if err := d.executor.Run(ctx, d.ip, "link", "add", host, "type", "veth", "peer", "name", peer); err != nil {
			if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", host); inspectErr != nil {
				_ = d.executor.Run(ctx, d.ip, "link", "delete", bridge)
				return err
			}
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", host, "alias", ownership.Marker("netlab", string(attachment.ID)))
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", peer, "alias", ownership.Marker("netlab", string(attachment.ID)))
	if err := d.executor.Run(ctx, d.ip, "link", "set", host, "master", bridge); err != nil {
		return err
	}
	if err := d.executor.Run(ctx, d.ip, "link", "set", HostInterfaceName(iface.ID), "master", bridge); err != nil {
		return err
	}
	if err := d.executor.Run(ctx, d.ip, "link", "set", host, "up"); err != nil {
		return err
	}
	if portErr != nil {
		if err := d.executor.Run(ctx, d.ip, "link", "set", peer, "netns", namespace); err != nil {
			return err
		}
		if err := d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", peer, "name", portName); err != nil {
			return err
		}
	}
	if object.Kind == domain.NetworkSwitchL2 {
		if err := d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", portName, "master", "br0"); err != nil {
			return err
		}
		if pvid := attachmentVLAN(attachment.Config, "pvid"); pvid > 0 {
			if err := d.executor.Run(ctx, "bridge", "-n", namespace, "vlan", "add", "dev", portName, "vid", fmt.Sprint(pvid), "pvid", "untagged"); err != nil {
				return err
			}
		}
		for _, vlan := range attachmentVLANs(attachment.Config["tagged"]) {
			if err := d.executor.Run(ctx, "bridge", "-n", namespace, "vlan", "add", "dev", portName, "vid", fmt.Sprint(vlan)); err != nil {
				return err
			}
		}
	}
	if err := d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", portName, "up"); err != nil {
		return err
	}
	return d.configureNamespacePort(ctx, namespace, portName, object)
}

func (d *DataPlane) DeleteNetworkObjectLink(ctx context.Context, link domain.NetworkObjectLink, objectA, objectB domain.NetworkObject) error {
	type endpoint struct {
		object      domain.NetworkObject
		port        string
		runtimeName string
	}
	var cleanupErrors []error
	attempted := false
	for _, value := range []endpoint{{object: objectA, port: link.PortAName, runtimeName: ownership.Name("nva", link.ID, 15)}, {object: objectB, port: link.PortBName, runtimeName: ownership.Name("nvb", link.ID, 15)}} {
		namespace, err := networkObjectNamespace(value.object)
		if err != nil {
			if value.object.Kind == domain.NetworkBridge || value.object.Kind == domain.NetworkNAT {
				attempted = true
				if err = d.executor.Run(ctx, d.ip, "link", "delete", value.runtimeName); err == nil {
					return nil
				} else if !missingLinkError(err) {
					cleanupErrors = append(cleanupErrors, err)
				}
				continue
			}
			cleanupErrors = append(cleanupErrors, err)
			continue
		}
		attempted = true
		if err = d.executor.Run(ctx, d.ip, "-n", namespace, "link", "delete", value.port); err == nil {
			return nil
		} else if !missingLinkError(err) {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	if attempted && len(cleanupErrors) == 0 {
		return nil
	}
	return errors.Join(cleanupErrors...)
}

func missingLinkError(err error) bool {
	return strings.Contains(err.Error(), "Cannot find device") || strings.Contains(err.Error(), "does not exist")
}

func (d *DataPlane) configureNamespacePort(ctx context.Context, namespace, portName string, object domain.NetworkObject) error {
	switch object.Kind {
	case domain.NetworkPC:
		var config domain.PCConfig
		if err := decodeConfig(object.Config, &config); err != nil {
			return err
		}
		for _, iface := range config.Interfaces {
			if iface.Name == portName {
				pc, err := NewPCRuntime(d.executor)
				if err != nil {
					return err
				}
				pc.ip = d.ip
				return pc.configureInterface(ctx, namespace, object.ID, iface)
			}
		}
	case domain.NetworkSwitchL3:
		var config domain.SwitchL3Config
		if err := decodeConfig(object.Config, &config); err != nil {
			return err
		}
		for _, iface := range config.Interfaces {
			if iface.Name != portName {
				continue
			}
			for _, address := range iface.Addresses {
				if err := d.executor.Run(ctx, d.ip, "-n", namespace, "address", "replace", address, "dev", portName); err != nil {
					return err
				}
			}
			forwardIPv4 := "0"
			if config.ForwardIPv4 {
				forwardIPv4 = "1"
			}
			if err := d.executor.Run(ctx, d.ip, "netns", "exec", namespace, "sysctl", "-qw", "net/ipv4/conf/"+portName+"/forwarding="+forwardIPv4); err != nil {
				return err
			}
			forwardIPv6 := "0"
			if config.ForwardIPv6 {
				forwardIPv6 = "1"
			}
			if err := d.executor.Run(ctx, d.ip, "netns", "exec", namespace, "sysctl", "-qw", "net/ipv6/conf/"+portName+"/forwarding="+forwardIPv6); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DataPlane) DeleteAttachment(ctx context.Context, attachment domain.NetworkAttachment) error {
	var cleanupErrors []error
	for _, name := range []string{ownership.Name("nah", attachment.ID, 15), ownership.Name("nla", attachment.ID, 15)} {
		if err := d.executor.Run(ctx, d.ip, "link", "delete", name); err != nil && !missingLinkError(err) {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func attachmentVLAN(config map[string]any, key string) int {
	switch value := config[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

func attachmentVLANs(value any) []int {
	var result []int
	switch values := value.(type) {
	case []int:
		return append(result, values...)
	case []any:
		for _, item := range values {
			switch vlan := item.(type) {
			case int:
				result = append(result, vlan)
			case float64:
				result = append(result, int(vlan))
			}
		}
	}
	return result
}
