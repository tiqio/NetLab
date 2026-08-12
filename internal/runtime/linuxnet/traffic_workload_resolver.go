package linuxnet

import (
	"context"
	"fmt"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

type TrafficWorkloadNodeRepository interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
}

type TrafficWorkloadObjectRepository interface {
	GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error)
}

type TrafficWorkloadTargetResolver struct {
	nodes   TrafficWorkloadNodeRepository
	objects TrafficWorkloadObjectRepository
}

func NewTrafficWorkloadTargetResolver(nodes TrafficWorkloadNodeRepository, objects TrafficWorkloadObjectRepository) *TrafficWorkloadTargetResolver {
	return &TrafficWorkloadTargetResolver{nodes: nodes, objects: objects}
}

func (r *TrafficWorkloadTargetResolver) ResolveTrafficWorkloadTarget(ctx context.Context, workload domain.TrafficWorkload) (ports.TrafficWorkloadTarget, error) {
	switch workload.Source.Kind {
	case "node":
		node, err := r.nodes.GetNode(ctx, workload.Source.ResourceID)
		if err != nil {
			return ports.TrafficWorkloadTarget{}, err
		}
		switch node.Kind {
		case string(domain.RuntimeDocker):
			return ports.TrafficWorkloadTarget{Kind: "docker", Container: "netlab-" + string(node.ID), Node: node}, nil
		case string(domain.RuntimeQEMU):
			return ports.TrafficWorkloadTarget{Kind: "qga", Node: node}, nil
		default:
			return ports.TrafficWorkloadTarget{}, trafficSourceProblem(workload, "node runtime does not support traffic execution")
		}
	case "network_object":
		object, err := r.objects.GetNetworkObject(ctx, workload.Source.ResourceID)
		if err != nil {
			return ports.TrafficWorkloadTarget{}, err
		}
		namespace, err := NetworkObjectNamespaceName(object)
		if err != nil {
			return ports.TrafficWorkloadTarget{}, trafficSourceProblem(workload, err.Error())
		}
		return ports.TrafficWorkloadTarget{Kind: "namespace", Namespace: namespace}, nil
	default:
		return ports.TrafficWorkloadTarget{}, trafficSourceProblem(workload, fmt.Sprintf("unsupported source kind %q", workload.Source.Kind))
	}
}

func trafficSourceProblem(workload domain.TrafficWorkload, message string) domain.Problem {
	return domain.Problem{Code: domain.ProblemCodeCapabilityUnavailable, Message: message, Retryable: true, ResourceType: "traffic_workload", ResourceID: workload.ID, Phase: "workload_source_resolution", Cleanup: "no command executed", OperatorHint: "choose a running namespace, Docker, or QGA-capable source"}
}
