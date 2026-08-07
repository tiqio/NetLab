package command

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type portMappingRepositoryStub struct {
	values []domain.PortMapping
}

func (s *portMappingRepositoryStub) CreatePortMapping(context.Context, domain.PortMapping) error {
	return nil
}
func (s *portMappingRepositoryStub) GetPortMapping(context.Context, domain.ID) (domain.PortMapping, error) {
	return domain.PortMapping{}, nil
}
func (s *portMappingRepositoryStub) ListNodePortMappings(context.Context, domain.ID) ([]domain.PortMapping, error) {
	return nil, nil
}
func (s *portMappingRepositoryStub) ListAllPortMappings(context.Context) ([]domain.PortMapping, error) {
	return s.values, nil
}
func (s *portMappingRepositoryStub) SetPortMappingState(context.Context, domain.ID, string, *domain.Problem) error {
	return nil
}
func (s *portMappingRepositoryStub) DeletePortMapping(context.Context, domain.ID) error {
	return nil
}

type portMappingRuntimeStub struct {
	busy             map[int]bool
	addressAvailable bool
}

func (s portMappingRuntimeStub) Apply(context.Context, domain.PortMapping) error { return nil }
func (s portMappingRuntimeStub) Delete(context.Context, domain.ID) error         { return nil }
func (s portMappingRuntimeStub) CheckHostPort(_ string, _ string, port int) error {
	if port == 0 && !s.addressAvailable {
		return context.DeadlineExceeded
	}
	if s.busy[port] {
		return context.DeadlineExceeded
	}
	return nil
}

func TestAutomaticPortMappingAllocationSkipsReservedAndBusyPorts(t *testing.T) {
	service := &PortMappingService{
		repository: &portMappingRepositoryStub{values: []domain.PortMapping{{Protocol: "tcp", HostAddress: "10.72.1.159", HostPort: 20000}}},
		runtime:    portMappingRuntimeStub{busy: map[int]bool{20001: true}, addressAvailable: true},
	}
	port, err := service.allocateHostPort(context.Background(), "10.72.1.159", "tcp")
	if err != nil || port != 20002 {
		t.Fatalf("port=%d err=%v", port, err)
	}
}

func TestAutomaticPortMappingRejectsUnavailableHostAddressBeforeScanning(t *testing.T) {
	service := &PortMappingService{
		repository: &portMappingRepositoryStub{},
		runtime:    portMappingRuntimeStub{},
	}
	_, err := service.allocateHostPort(context.Background(), "10.72.1.159", "tcp")
	problem, ok := err.(domain.Problem)
	if !ok || problem.Code != "host_address_unavailable" {
		t.Fatalf("error=%#v", err)
	}
}

func TestAutomaticPortMappingChoosesMatchingAddressFamily(t *testing.T) {
	if value := matchingAddressFamily("10.72.1.159", []string{"2001:db8::10", "10.10.0.155"}); value != "10.10.0.155" {
		t.Fatalf("IPv4 address=%q", value)
	}
	if value := matchingAddressFamily("::1", []string{"10.10.0.155", "2001:db8::10"}); value != "2001:db8::10" {
		t.Fatalf("IPv6 address=%q", value)
	}
}
