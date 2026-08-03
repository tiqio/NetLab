package linuxnet

import (
	"context"
	"os/exec"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type EndpointRuntime struct {
	executor CommandExecutor
	ipPath   string
}

func NewEndpointRuntime() (*EndpointRuntime, error) {
	path, err := exec.LookPath("ip")
	if err != nil {
		return nil, err
	}
	return &EndpointRuntime{executor: SystemExecutor{}, ipPath: path}, nil
}
func (r *EndpointRuntime) namespace(node domain.Node) string {
	return ownership.Name("nl", node.ID, 15)
}
func (r *EndpointRuntime) Inspect(ctx context.Context, node domain.Node) (ports.ActualNode, error) {
	name := r.namespace(node)
	if namespaceReady(ctx, r.executor, r.ipPath, name) {
		return ports.ActualNode{State: domain.ObservedRunning, Owner: map[string]string{"netns": name}}, nil
	}
	return ports.ActualNode{State: domain.ObservedStopped}, nil
}
func (r *EndpointRuntime) Start(ctx context.Context, node domain.Node) error {
	actual, err := r.Inspect(ctx, node)
	if err == nil && actual.State == domain.ObservedRunning {
		return nil
	}
	if err := ensureNamespace(ctx, r.executor, r.ipPath, r.namespace(node)); err != nil {
		return err
	}
	for _, iface := range InterfaceDescriptors(node) {
		host, peer := HostInterfaceName(iface.ID), PeerInterfaceName(iface.ID)
		if err := r.executor.Run(ctx, r.ipPath, "link", "add", host, "type", "veth", "peer", "name", peer); err != nil {
			if _, inspectErr := r.executor.Output(ctx, r.ipPath, "link", "show", host); inspectErr != nil {
				return err
			}
		} else if err := r.executor.Run(ctx, r.ipPath, "link", "set", peer, "netns", r.namespace(node)); err != nil {
			return err
		}
		_ = r.executor.Run(ctx, r.ipPath, "link", "set", host, "alias", "netlab:"+string(node.ID))
		_ = r.executor.Run(ctx, r.ipPath, "link", "set", host, "up")
		if _, err := r.executor.Output(ctx, r.ipPath, "-n", r.namespace(node), "link", "show", iface.Name); err != nil {
			_ = r.executor.Run(ctx, r.ipPath, "-n", r.namespace(node), "link", "set", peer, "name", iface.Name)
		}
		_ = r.executor.Run(ctx, r.ipPath, "-n", r.namespace(node), "link", "set", iface.Name, "up")
	}
	return r.executor.Run(ctx, r.ipPath, "-n", r.namespace(node), "link", "set", "lo", "up")
}
func (r *EndpointRuntime) Stop(ctx context.Context, node domain.Node) error {
	_ = deleteNamespace(ctx, r.executor, r.ipPath, r.namespace(node))
	return nil
}
func (r *EndpointRuntime) Delete(ctx context.Context, node domain.Node) error {
	return r.Stop(ctx, node)
}
func (r *EndpointRuntime) Name() string { return "namespace-endpoint" }

var _ ports.NodeRuntime = (*EndpointRuntime)(nil)
