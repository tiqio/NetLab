package linuxnet

import (
	"fmt"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type InterfaceDescriptor struct {
	ID         domain.ID
	Slot       int
	Name       string
	Driver     string
	MACAddress string
}

func InterfaceDescriptors(node domain.Node) []InterfaceDescriptor {
	values, _ := node.Config["interfaces"].([]any)
	if direct, ok := node.Config["interfaces"].([]map[string]any); ok {
		values = make([]any, len(direct))
		for index := range direct {
			values[index] = direct[index]
		}
	}
	result := make([]InterfaceDescriptor, 0, len(values))
	for index, raw := range values {
		value, _ := raw.(map[string]any)
		id, _ := value["id"].(string)
		name, _ := value["name"].(string)
		driver, _ := value["driver"].(string)
		mac, _ := value["mac_address"].(string)
		slot := index
		switch configured := value["slot"].(type) {
		case int:
			slot = configured
		case float64:
			slot = int(configured)
		}
		if id != "" && name != "" {
			result = append(result, InterfaceDescriptor{ID: domain.ID(id), Slot: slot, Name: name, Driver: driver, MACAddress: mac})
		}
	}
	return result
}

func HostInterfaceName(id domain.ID) string { return ownership.Name("nli", id, 15) }
func PeerInterfaceName(id domain.ID) string { return ownership.Name("nlp", id, 15) }
func LinkBridgeName(id domain.ID) string    { return ownership.Name("nll", id, 15) }

func SwitchL2NamespaceName(id domain.ID) string { return ownership.Name("n2sw", id, 15) }
func SwitchL3NamespaceName(id domain.ID) string { return ownership.Name("n2r", id, 15) }

func NetworkBridgeName(object domain.NetworkObject) (string, error) {
	switch object.Kind {
	case domain.NetworkBridge:
		return ownership.Name("nlbr", object.ID, 15), nil
	case domain.NetworkNAT:
		return ownership.Name("nlnat", object.ID, 15), nil
	default:
		return "", fmt.Errorf("network object %s is not an attachable bridge", object.Kind)
	}
}
