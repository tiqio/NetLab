package ports

import (
	"context"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type TrafficWorkloadTarget struct {
	Kind      string
	Namespace string
	Container string
	Node      domain.Node
}

type TrafficWorkloadExecution struct {
	ExitCode     int
	MatchedBytes int64
	Truncated    bool
}

type TrafficWorkloadExecutor interface {
	ExecuteTrafficWorkload(context.Context, domain.TrafficWorkload, TrafficWorkloadTarget) (TrafficWorkloadExecution, error)
}

type TrafficWorkloadTargetResolver interface {
	ResolveTrafficWorkloadTarget(context.Context, domain.TrafficWorkload) (TrafficWorkloadTarget, error)
}

type TrafficWorkloadFilterCorrelator interface {
	CorrelateSuccessfulWorkload(domain.TrafficWorkload, time.Time) []domain.ID
}
