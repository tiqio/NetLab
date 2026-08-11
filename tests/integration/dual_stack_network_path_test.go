package integration

import (
	"context"
	"testing"

	"github.com/netlab/netlab/tests/testsupport"
)

type deterministicDualStackProber struct{ calls []testsupport.DualStackProbe }

func (p *deterministicDualStackProber) Probe(_ context.Context, probe testsupport.DualStackProbe) (int, error) {
	p.calls = append(p.calls, probe)
	return 99, nil
}

func TestDualStackComponentMatrixCoversServiceTransitCoreDMZAndManagement(t *testing.T) {
	fixture := testsupport.ComponentMatrixDualStackFixture()
	prober := &deterministicDualStackProber{}
	if err := fixture.Run(context.Background(), prober); err != nil {
		t.Fatal(err)
	}
	if len(prober.calls) != 10 {
		t.Fatalf("probes=%d", len(prober.calls))
	}
}
