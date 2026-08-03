package integration_test

import (
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestPCAutomaticAddressingMatrixAcceptsDeclaredModes(t *testing.T) {
	config := domain.PCConfig{Interfaces: []domain.PCInterfaceConfig{{Name: "eth0", Modes: []domain.AddressMode{domain.AddressDHCPv4, domain.AddressDHCPv6, domain.AddressSLAAC}, DNS: []string{"1.1.1.1", "2606:4700:4700::1111"}}}}
	if err := domain.ValidatePCConfig(config); err != nil {
		t.Fatal(err)
	}
}
