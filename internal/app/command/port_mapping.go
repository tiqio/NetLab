package command

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type PortMappingRepository interface {
	CreatePortMapping(context.Context, domain.PortMapping) error
	GetPortMapping(context.Context, domain.ID) (domain.PortMapping, error)
	ListNodePortMappings(context.Context, domain.ID) ([]domain.PortMapping, error)
	ListAllPortMappings(context.Context) ([]domain.PortMapping, error)
	SetPortMappingState(context.Context, domain.ID, string, *domain.Problem) error
	DeletePortMapping(context.Context, domain.ID) error
}

type PortMappingRuntime interface {
	Apply(context.Context, domain.PortMapping) error
	Delete(context.Context, domain.ID) error
	CheckHostPort(string, string, int) error
}

type PortMappingNodeReader interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
}

type PortMappingAddressResolver interface {
	ResolveAddresses(context.Context, domain.Node) ([]string, error)
}

type PortMappingService struct {
	repository PortMappingRepository
	runner     *task.Runner
	runtime    PortMappingRuntime
	nodes      PortMappingNodeReader
	addresses  PortMappingAddressResolver
}

func (s *PortMappingService) SetAutoResolver(nodes PortMappingNodeReader, addresses PortMappingAddressResolver) {
	s.nodes = nodes
	s.addresses = addresses
}

func NewPortMappingService(repository PortMappingRepository, runner *task.Runner, runtime PortMappingRuntime) *PortMappingService {
	service := &PortMappingService{repository: repository, runner: runner, runtime: runtime}
	runner.Register("port_mapping.create", service.handleCreate)
	runner.Register("port_mapping.delete", service.handleDelete)
	return service
}

func (s *PortMappingService) Create(ctx context.Context, value domain.PortMapping, idempotencyKey string) (domain.PortMapping, domain.OperationTask, error) {
	if value.Protocol != "tcp" && value.Protocol != "udp" {
		return value, domain.OperationTask{}, domain.Problem{Code: "validation_failed", Message: "protocol must be tcp or udp", ResourceType: "node", ResourceID: value.NodeID}
	}
	if value.GuestPort < 1 || value.GuestPort > 65535 {
		return value, domain.OperationTask{}, domain.Problem{Code: "validation_failed", Message: "guest port must be between 1 and 65535", ResourceType: "node", ResourceID: value.NodeID}
	}
	if value.HostAddress == "" {
		value.HostAddress = "0.0.0.0"
	}
	if net.ParseIP(value.HostAddress) == nil {
		return value, domain.OperationTask{}, domain.Problem{Code: "validation_failed", Message: "host address must be an IP address", ResourceType: "node", ResourceID: value.NodeID}
	}
	if value.GuestAddress == "" {
		if s.nodes == nil || s.addresses == nil {
			return value, domain.OperationTask{}, domain.Problem{Code: "guest_address_unavailable", Message: "automatic guest address resolution is unavailable", ResourceType: "node", ResourceID: value.NodeID}
		}
		node, err := s.nodes.GetNode(ctx, value.NodeID)
		if err != nil {
			return value, domain.OperationTask{}, err
		}
		addresses, err := s.addresses.ResolveAddresses(ctx, node)
		if err != nil {
			return value, domain.OperationTask{}, domain.Problem{Code: "guest_address_unavailable", Message: err.Error(), ResourceType: "node", ResourceID: value.NodeID, OperatorHint: "attach a DHCP-enabled interface to a running NAT bridge or configure a static address"}
		}
		value.GuestAddress = matchingAddressFamily(value.HostAddress, addresses)
		if value.GuestAddress == "" {
			return value, domain.OperationTask{}, domain.Problem{Code: "guest_address_unavailable", Message: "no guest address matches the host address family", ResourceType: "node", ResourceID: value.NodeID}
		}
	}
	if value.HostPort == 0 {
		port, err := s.allocateHostPort(ctx, value.HostAddress, value.Protocol)
		if err != nil {
			return value, domain.OperationTask{}, err
		}
		value.HostPort = port
	}
	if value.HostPort < 1 || value.HostPort > 65535 {
		return value, domain.OperationTask{}, domain.Problem{Code: "validation_failed", Message: "host port must be zero for automatic allocation or between 1 and 65535", ResourceType: "node", ResourceID: value.NodeID}
	}
	value.ID = domain.NewID()
	value.Revision = 1
	value.ObservedState = "pending"
	value.CreatedAt = time.Now().UTC()
	if err := s.repository.CreatePortMapping(ctx, value); err != nil {
		return value, domain.OperationTask{}, err
	}
	taskValue := domain.OperationTask{ID: domain.NewID(), Kind: "port_mapping.create", ResourceType: "port_mapping", ResourceID: value.ID, IdempotencyKey: idempotencyKey, ProgressTotal: 1, CreatedAt: time.Now().UTC()}
	if err := s.runner.Enqueue(ctx, taskValue); err != nil {
		_ = s.repository.DeletePortMapping(ctx, value.ID)
		return value, taskValue, err
	}
	return value, taskValue, nil
}

func (s *PortMappingService) allocateHostPort(ctx context.Context, address, protocol string) (int, error) {
	values, err := s.repository.ListAllPortMappings(ctx)
	if err != nil {
		return 0, err
	}
	reserved := map[int]bool{}
	for _, value := range values {
		if value.Protocol == protocol && value.HostAddress == address {
			reserved[value.HostPort] = true
		}
	}
	for port := 20000; port <= 39999; port++ {
		if reserved[port] || s.runtime.CheckHostPort(address, protocol, port) != nil {
			continue
		}
		return port, nil
	}
	return 0, domain.Problem{Code: "port_range_exhausted", Message: fmt.Sprintf("no available %s port in automatic range 20000-39999", protocol), ResourceType: "port_mapping"}
}

func matchingAddressFamily(hostAddress string, addresses []string) string {
	hostIPv4 := net.ParseIP(hostAddress).To4() != nil
	for _, address := range addresses {
		parsed := net.ParseIP(address)
		if parsed != nil && (parsed.To4() != nil) == hostIPv4 {
			return address
		}
	}
	return ""
}

func (s *PortMappingService) ListNode(ctx context.Context, nodeID domain.ID) ([]domain.PortMapping, error) {
	return s.repository.ListNodePortMappings(ctx, nodeID)
}

func (s *PortMappingService) Delete(ctx context.Context, id domain.ID, idempotencyKey string) (domain.OperationTask, error) {
	if _, err := s.repository.GetPortMapping(ctx, id); err != nil {
		return domain.OperationTask{}, err
	}
	value := domain.OperationTask{ID: domain.NewID(), Kind: "port_mapping.delete", ResourceType: "port_mapping", ResourceID: id, IdempotencyKey: idempotencyKey, ProgressTotal: 2, CreatedAt: time.Now().UTC()}
	return value, s.runner.Enqueue(ctx, value)
}

func (s *PortMappingService) handleCreate(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	mapping, err := s.repository.GetPortMapping(ctx, value.ResourceID)
	if err != nil {
		return nil, err
	}
	if err = s.runtime.Apply(ctx, mapping); err != nil {
		problem := &domain.Problem{Code: "port_mapping_failed", Message: err.Error()}
		_ = s.repository.SetPortMappingState(context.Background(), mapping.ID, "failed", problem)
		return nil, err
	}
	value.ProgressCurrent = 1
	_ = s.repository.SetPortMappingState(context.Background(), mapping.ID, "active", nil)
	return map[string]any{"mapping_id": mapping.ID}, nil
}

func (s *PortMappingService) handleDelete(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	if err := s.runtime.Delete(ctx, value.ResourceID); err != nil {
		return nil, err
	}
	value.ProgressCurrent = 1
	if err := s.repository.DeletePortMapping(ctx, value.ResourceID); err != nil {
		return nil, err
	}
	value.ProgressCurrent = 2
	return map[string]any{"mapping_id": value.ResourceID}, nil
}
