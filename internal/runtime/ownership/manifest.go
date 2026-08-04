package ownership

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

type Object struct {
	Kind     string            `json:"kind"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
type Manifest struct {
	ResourceType string    `json:"resource_type"`
	ResourceID   domain.ID `json:"resource_id"`
	Objects      []Object  `json:"objects"`
}

type Record struct {
	ResourceType   string            `json:"resource_type"`
	ResourceID     domain.ID         `json:"resource_id"`
	ObjectKind     string            `json:"object_kind"`
	ObjectName     string            `json:"object_name"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CleanupState   string            `json:"cleanup_state"`
	OwnershipClass string            `json:"ownership_class"`
}

const (
	ClassManaged         = "managed"
	ClassAcceptanceOwned = "acceptance_owned"
	ClassForeignObserved = "foreign_observed"
)

func Classify(resourceType string, metadata map[string]string) string {
	if metadata["acceptance_run_id"] != "" {
		return ClassAcceptanceOwned
	}
	if resourceType == "unknown" || metadata["ownership_class"] == ClassForeignObserved {
		return ClassForeignObserved
	}
	return ClassManaged
}

func DirectVethPairManifest(resourceID domain.ID, namespaceA, portA, namespaceB, portB string) Manifest {
	return Manifest{
		ResourceType: "network_object_link",
		ResourceID:   resourceID,
		Objects: []Object{
			{Kind: "network_object_link_endpoint", Name: namespaceA + ":" + portA, Metadata: map[string]string{"endpoint": "a", "namespace": namespaceA, "port": portA}},
			{Kind: "network_object_link_endpoint", Name: namespaceB + ":" + portB, Metadata: map[string]string{"endpoint": "b", "namespace": namespaceB, "port": portB}},
		},
	}
}

func Name(prefix string, id domain.ID, max int) string {
	sum := sha256.Sum256([]byte(id))
	suffix := hex.EncodeToString(sum[:6])
	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, strings.ToLower(prefix))
	value := clean + "-" + suffix
	if max > 0 && len(value) > max {
		return value[:max]
	}
	return value
}
func (m Manifest) Owns(kind, name string) bool {
	for _, object := range m.Objects {
		if object.Kind == kind && object.Name == name {
			return true
		}
	}
	return false
}
func (m *Manifest) Add(kind, name string, metadata map[string]string) error {
	if kind == "" || name == "" {
		return fmt.Errorf("kind and name required")
	}
	if m.Owns(kind, name) {
		return nil
	}
	m.Objects = append(m.Objects, Object{Kind: kind, Name: name, Metadata: metadata})
	return nil
}
func RequireOwned(manifest Manifest, kind, name string) error {
	if !manifest.Owns(kind, name) {
		return fmt.Errorf("refusing to modify unowned %s %s", kind, name)
	}
	return nil
}
