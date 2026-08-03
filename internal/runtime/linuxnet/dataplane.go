package linuxnet

import (
	"context"
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
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", bridge, "alias", "netlab:"+string(link.ID))
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
	bridge := LinkBridgeName(link.ID)
	if err := d.executor.Run(ctx, d.ip, "link", "add", bridge, "type", "bridge"); err != nil {
		if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", bridge); inspectErr != nil {
			return err
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", bridge, "alias", "netlab:"+string(link.ID))
	if err := d.executor.Run(ctx, d.ip, "link", "set", bridge, "up"); err != nil {
		return err
	}
	if err := d.ensureNetworkObjectLinkEndpoint(ctx, link, "a", link.PortAName, objectA, bridge); err != nil {
		_ = d.DeleteLink(ctx, link.ID)
		return err
	}
	if err := d.ensureNetworkObjectLinkEndpoint(ctx, link, "b", link.PortBName, objectB, bridge); err != nil {
		_ = d.DeleteLink(ctx, link.ID)
		return err
	}
	return nil
}

func (d *DataPlane) ensureNetworkObjectLinkEndpoint(ctx context.Context, link domain.NetworkObjectLink, side, portName string, object domain.NetworkObject, bridge string) error {
	namespace, err := networkObjectNamespace(object)
	if err != nil {
		return err
	}
	endpointID := domain.ID(string(link.ID) + "-" + side)
	host := ownership.Name("nlh", endpointID, 15)
	peer := ownership.Name("nlp", endpointID, 15)
	if err = d.executor.Run(ctx, d.ip, "link", "add", host, "type", "veth", "peer", "name", peer); err != nil {
		if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", host); inspectErr != nil {
			return err
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", host, "alias", "netlab:"+string(link.ID))
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", peer, "alias", "netlab:"+string(link.ID))
	if err = d.executor.Run(ctx, d.ip, "link", "set", host, "master", bridge); err != nil {
		return err
	}
	if err = d.executor.Run(ctx, d.ip, "link", "set", host, "up"); err != nil {
		return err
	}
	if err = d.executor.Run(ctx, d.ip, "link", "set", peer, "netns", namespace); err != nil {
		if !strings.Contains(err.Error(), "Cannot find device") {
			return err
		}
	} else if err = d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", peer, "name", portName); err != nil {
		return err
	}
	if object.Kind == domain.NetworkSwitchL2 {
		if err = d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", portName, "master", "br0"); err != nil {
			return err
		}
		if err = d.configureSwitchL2Port(ctx, namespace, portName, object); err != nil {
			return err
		}
	}
	if err = d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", portName, "up"); err != nil {
		return err
	}
	return d.configureNamespacePort(ctx, namespace, portName, object)
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
	if err := d.executor.Run(ctx, d.ip, "link", "add", bridge, "type", "bridge"); err != nil {
		if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", bridge); inspectErr != nil {
			return err
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", bridge, "alias", "netlab:"+string(attachment.ID))
	if err := d.executor.Run(ctx, d.ip, "link", "set", bridge, "up"); err != nil {
		return err
	}
	if err := d.executor.Run(ctx, d.ip, "link", "add", host, "type", "veth", "peer", "name", peer); err != nil {
		if _, inspectErr := d.executor.Output(ctx, d.ip, "link", "show", host); inspectErr != nil {
			_ = d.executor.Run(ctx, d.ip, "link", "delete", bridge)
			return err
		}
	}
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", host, "alias", "netlab:"+string(attachment.ID))
	_ = d.executor.Run(ctx, d.ip, "link", "set", "dev", peer, "alias", "netlab:"+string(attachment.ID))
	if err := d.executor.Run(ctx, d.ip, "link", "set", host, "master", bridge); err != nil {
		return err
	}
	if err := d.executor.Run(ctx, d.ip, "link", "set", HostInterfaceName(iface.ID), "master", bridge); err != nil {
		return err
	}
	if err := d.executor.Run(ctx, d.ip, "link", "set", host, "up"); err != nil {
		return err
	}
	if err := d.executor.Run(ctx, d.ip, "link", "set", peer, "netns", namespace); err != nil && !strings.Contains(err.Error(), "Cannot find device") {
		return err
	}
	portName := attachment.PortName
	if portName == "" {
		portName = "eth0"
	}
	_ = d.executor.Run(ctx, d.ip, "-n", namespace, "link", "set", peer, "name", portName)
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

func (d *DataPlane) DeleteNetworkObjectLink(ctx context.Context, id domain.ID) error {
	for _, side := range []string{"a", "b"} {
		endpointID := domain.ID(string(id) + "-" + side)
		host := ownership.Name("nlh", endpointID, 15)
		if err := d.executor.Run(ctx, d.ip, "link", "delete", host); err != nil && !missingLinkError(err) {
			return err
		}
	}
	return d.DeleteLink(ctx, id)
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
		}
	}
	return nil
}

func (d *DataPlane) DeleteAttachment(ctx context.Context, attachment domain.NetworkAttachment) error {
	err := d.executor.Run(ctx, d.ip, "link", "delete", ownership.Name("nla", attachment.ID, 15))
	if err != nil && (strings.Contains(err.Error(), "Cannot find device") || strings.Contains(err.Error(), "does not exist")) {
		return nil
	}
	return err
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
