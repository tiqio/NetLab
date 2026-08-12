package linuxnet

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
)

const TrafficWorkloadOutputLimit = 64 << 10

type TrafficWorkloadTarget struct {
	Kind      string
	Namespace string
	Container string
	Node      domain.Node
}

type TrafficWorkloadResult struct {
	ExitCode     int
	Output       []byte
	MatchedBytes int64
	Truncated    bool
}

type TrafficWorkloadGuestResult struct {
	ExitCode  int
	Stdout    []byte
	Stderr    []byte
	Truncated bool
}

type TrafficWorkloadGuestExecutor func(context.Context, domain.Node, []string, time.Duration, int) (TrafficWorkloadGuestResult, error)

type TrafficWorkloadRuntime struct {
	executor CommandExecutor
	docker   CommandExecutor
	guest    TrafficWorkloadGuestExecutor
}

func NewTrafficWorkloadRuntime(executor, docker CommandExecutor, guest TrafficWorkloadGuestExecutor) *TrafficWorkloadRuntime {
	if executor == nil {
		executor = SystemExecutor{}
	}
	if docker == nil {
		docker = SystemExecutor{}
	}
	return &TrafficWorkloadRuntime{executor: executor, docker: docker, guest: guest}
}

func (r *TrafficWorkloadRuntime) ExecuteTrafficWorkload(ctx context.Context, workload domain.TrafficWorkload, target ports.TrafficWorkloadTarget) (ports.TrafficWorkloadExecution, error) {
	result, err := r.Execute(ctx, workload, TrafficWorkloadTarget(target))
	return ports.TrafficWorkloadExecution{ExitCode: result.ExitCode, MatchedBytes: result.MatchedBytes, Truncated: result.Truncated}, err
}

func (r *TrafficWorkloadRuntime) Execute(ctx context.Context, workload domain.TrafficWorkload, target TrafficWorkloadTarget) (TrafficWorkloadResult, error) {
	if err := workload.Validate(); err != nil {
		return TrafficWorkloadResult{}, domain.Problem{Code: domain.ProblemCodeInvalidRequest, Message: err.Error(), ResourceType: "traffic_workload", ResourceID: workload.ID, Phase: "workload_runtime_admission", Cleanup: "no command executed"}
	}
	argv, err := trafficWorkloadArgv(workload)
	if err != nil {
		return TrafficWorkloadResult{}, err
	}
	timeout := time.Duration(workload.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var output []byte
	result := TrafficWorkloadResult{}
	switch target.Kind {
	case "namespace":
		if strings.TrimSpace(target.Namespace) == "" {
			return result, workloadRuntimeProblem(workload.ID, "namespace target is required")
		}
		output, err = r.executor.Output(ctx, "ip", append([]string{"netns", "exec", target.Namespace}, argv...)...)
	case "docker":
		if strings.TrimSpace(target.Container) == "" {
			return result, workloadRuntimeProblem(workload.ID, "container target is required")
		}
		output, err = r.docker.Output(ctx, "docker", append([]string{"exec", target.Container}, argv...)...)
	case "qga":
		if r.guest == nil {
			return result, workloadRuntimeProblem(workload.ID, "QEMU guest execution is unavailable")
		}
		guestResult, guestErr := r.guest(ctx, target.Node, argv, timeout, TrafficWorkloadOutputLimit)
		err = guestErr
		result.ExitCode = guestResult.ExitCode
		output = append(append([]byte(nil), guestResult.Stdout...), guestResult.Stderr...)
		result.Truncated = guestResult.Truncated
	default:
		return result, workloadRuntimeProblem(workload.ID, "unsupported workload source capability")
	}
	if err != nil {
		if ctx.Err() != nil {
			return result, domain.Problem{Code: "workload_timeout", Message: ctx.Err().Error(), Retryable: true, ResourceType: "traffic_workload", ResourceID: workload.ID, Phase: "workload_execute", Cleanup: "command context cancelled", OperatorHint: "verify destination reachability or increase timeout within supported limits"}
		}
		return result, domain.Problem{Code: "workload_exchange_failed", Message: err.Error(), Retryable: true, ResourceType: "traffic_workload", ResourceID: workload.ID, Phase: "workload_execute", Cleanup: "command exited without owned background process", OperatorHint: "inspect source capability and destination service"}
	}
	if len(output) > TrafficWorkloadOutputLimit {
		output = append([]byte(nil), output[:TrafficWorkloadOutputLimit]...)
		result.Truncated = true
	}
	result.Output = output
	result.MatchedBytes = trafficWorkloadMatchedBytes(workload, output)
	return result, nil
}

func trafficWorkloadArgv(workload domain.TrafficWorkload) ([]string, error) {
	family := []string{}
	switch workload.AddressFamily {
	case "ipv4":
		family = []string{"-4"}
	case "ipv6":
		family = []string{"-6"}
	case "auto":
	default:
		return nil, workloadRuntimeProblem(workload.ID, "unsupported address family")
	}
	timeout := strconv.Itoa(workload.TimeoutSeconds)
	switch workload.Protocol {
	case "icmp":
		if _, err := netip.ParseAddr(workload.Destination.Address); err != nil {
			return nil, workloadRuntimeProblem(workload.ID, "invalid ICMP destination")
		}
		return append([]string{"ping"}, append(family, "-n", "-c", "1", "-W", timeout, workload.Destination.Address)...), nil
	case "http":
		return append([]string{"curl"}, append(family, "--noproxy", "*", "--fail", "--silent", "--show-error", "--max-time", timeout, "--output", "/dev/null", "--write-out", "%{size_download}", workload.Destination.URL)...), nil
	case "dns":
		database := "ahosts"
		if workload.AddressFamily == "ipv4" {
			database = "ahostsv4"
		} else if workload.AddressFamily == "ipv6" {
			database = "ahostsv6"
		}
		return []string{"getent", database, workload.Destination.Name}, nil
	default:
		return nil, workloadRuntimeProblem(workload.ID, "unsupported workload protocol")
	}
}

func trafficWorkloadMatchedBytes(workload domain.TrafficWorkload, output []byte) int64 {
	if workload.Protocol == "http" {
		value, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err == nil && value > 0 {
			return int64(value)
		}
	}
	if len(output) > 0 {
		return int64(len(output))
	}
	return 1
}

func workloadRuntimeProblem(id domain.ID, message string) domain.Problem {
	return domain.Problem{Code: domain.ProblemCodeInvalidRequest, Message: message, ResourceType: "traffic_workload", ResourceID: id, Phase: "workload_runtime_admission", Cleanup: "no command executed", OperatorHint: "use a supported source and protocol definition"}
}

func (r TrafficWorkloadResult) String() string {
	return fmt.Sprintf("exit=%d bytes=%d truncated=%t", r.ExitCode, r.MatchedBytes, r.Truncated)
}
