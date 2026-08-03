package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"
	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type Engine interface {
	ContainerCreate(context.Context, dockerclient.ContainerCreateOptions) (dockerclient.ContainerCreateResult, error)
	ContainerStart(context.Context, string, dockerclient.ContainerStartOptions) (dockerclient.ContainerStartResult, error)
	ContainerStop(context.Context, string, dockerclient.ContainerStopOptions) (dockerclient.ContainerStopResult, error)
	ContainerRemove(context.Context, string, dockerclient.ContainerRemoveOptions) (dockerclient.ContainerRemoveResult, error)
	ContainerInspect(context.Context, string, dockerclient.ContainerInspectOptions) (dockerclient.ContainerInspectResult, error)
	ContainerList(context.Context, dockerclient.ContainerListOptions) (dockerclient.ContainerListResult, error)
}
type EndpointRuntime interface {
	Ensure(context.Context, domain.Node, int) error
	Cleanup(context.Context, domain.Node) error
}

type execEngine interface {
	ExecCreate(context.Context, string, dockerclient.ExecCreateOptions) (dockerclient.ExecCreateResult, error)
	ExecAttach(context.Context, string, dockerclient.ExecAttachOptions) (dockerclient.ExecAttachResult, error)
}

type attachedConsole struct {
	response dockerclient.HijackedResponse
}

func (c *attachedConsole) Read(buffer []byte) (int, error) {
	return c.response.Reader.Read(buffer)
}

func (c *attachedConsole) Write(buffer []byte) (int, error) {
	return c.response.Conn.Write(buffer)
}

func (c *attachedConsole) Close() error {
	c.response.Close()
	return nil
}

type Adapter struct {
	engine    Engine
	endpoints EndpointRuntime
}

func NewAdapter() (*Adapter, error) {
	client, err := dockerclient.New(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
		dockerclient.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}
	endpoints, err := linuxnet.NewDockerEndpointRuntime(nil)
	if err != nil {
		return nil, err
	}
	return &Adapter{engine: client, endpoints: endpoints}, nil
}
func NewAdapterWithEngine(engine Engine) *Adapter { return &Adapter{engine: engine} }
func NewAdapterWithRuntime(engine Engine, endpoints EndpointRuntime) *Adapter {
	return &Adapter{engine: engine, endpoints: endpoints}
}

func (a *Adapter) OpenConsole(ctx context.Context, node domain.Node) (io.ReadWriteCloser, error) {
	engine, ok := a.engine.(execEngine)
	if !ok {
		return nil, fmt.Errorf("docker exec console is unavailable")
	}
	containerID, running, err := a.find(ctx, node)
	if err != nil {
		return nil, err
	}
	if containerID == "" || !running {
		return nil, fmt.Errorf("container is not running")
	}
	created, err := engine.ExecCreate(ctx, containerID, dockerclient.ExecCreateOptions{
		TTY:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{"TERM=xterm-256color"},
		Cmd:          []string{"/bin/sh", "-lc", "if command -v bash >/dev/null 2>&1; then exec bash -l; else exec sh; fi"},
	})
	if err != nil {
		return nil, err
	}
	attached, err := engine.ExecAttach(ctx, created.ID, dockerclient.ExecAttachOptions{TTY: true})
	if err != nil {
		return nil, err
	}
	return &attachedConsole{response: attached.HijackedResponse}, nil
}

func (a *Adapter) DiscoverRuntimeOwnership(ctx context.Context) ([]ownership.Record, error) {
	if a == nil || a.engine == nil {
		return nil, nil
	}
	filters := make(dockerclient.Filters).Add("label", "io.netlab.node_id")
	result, err := a.engine.ContainerList(ctx, dockerclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, err
	}
	records := make([]ownership.Record, 0, len(result.Items))
	for _, item := range result.Items {
		nodeID := domain.ID(item.Labels["io.netlab.node_id"])
		if nodeID == "" || item.ID == "" {
			continue
		}
		metadata := map[string]string{"state": string(item.State)}
		if laboratoryID := item.Labels["io.netlab.laboratory_id"]; laboratoryID != "" {
			metadata["laboratory_id"] = laboratoryID
		}
		records = append(records, ownership.Record{ResourceType: "node", ResourceID: nodeID, ObjectKind: "docker_container", ObjectName: item.ID, Metadata: metadata, CleanupState: "active"})
	}
	return records, nil
}
func (a *Adapter) name(id domain.ID) string { return "netlab-" + string(id) }
func (a *Adapter) find(ctx context.Context, node domain.Node) (string, bool, error) {
	filters := make(dockerclient.Filters).Add("label", "io.netlab.node_id="+string(node.ID))
	result, err := a.engine.ContainerList(ctx, dockerclient.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return "", false, err
	}
	if len(result.Items) == 0 {
		return "", false, nil
	}
	return result.Items[0].ID, result.Items[0].State == "running", nil
}
func (a *Adapter) Inspect(ctx context.Context, node domain.Node) (ports.ActualNode, error) {
	id, running, err := a.find(ctx, node)
	if err != nil {
		return ports.ActualNode{}, err
	}
	if id == "" {
		return ports.ActualNode{State: domain.ObservedStopped}, nil
	}
	state := domain.ObservedStopped
	if running {
		state = domain.ObservedRunning
	}
	owner := map[string]string{"container_id": id}
	if running {
		if inspection, inspectErr := a.engine.ContainerInspect(ctx, id, dockerclient.ContainerInspectOptions{}); inspectErr == nil && inspection.Container.State != nil && inspection.Container.State.Pid > 0 {
			owner["pid"] = strconv.Itoa(inspection.Container.State.Pid)
		}
	}
	return ports.ActualNode{State: state, Owner: owner}, nil
}
func (a *Adapter) Start(ctx context.Context, node domain.Node) error {
	id, running, err := a.find(ctx, node)
	if err != nil {
		return err
	}
	if running {
		return a.ensureEndpoints(ctx, node, id)
	}
	created := false
	if id == "" {
		image, ok := node.Config["image"].(string)
		if !ok || image == "" {
			return fmt.Errorf("container image required")
		}
		command, commandErr := containerCommand(node.Config["command"])
		if commandErr != nil {
			return commandErr
		}
		pidsLimit := int64(node.ProcessLimit)
		storage := map[string]string{}
		if node.StorageGiB > 0 {
			storage["size"] = fmt.Sprintf("%dG", node.StorageGiB)
		}
		result, err := a.engine.ContainerCreate(ctx, dockerclient.ContainerCreateOptions{Image: image, Name: a.name(node.ID), Config: &container.Config{Cmd: command, Labels: map[string]string{"io.netlab.node_id": string(node.ID), "io.netlab.laboratory_id": string(node.LaboratoryID)}}, HostConfig: &container.HostConfig{NetworkMode: "none", StorageOpt: storage, Resources: container.Resources{Memory: int64(node.MemoryMiB) << 20, NanoCPUs: quotaToNano(node.CPUQuotaMicros), PidsLimit: &pidsLimit}}})
		if err != nil {
			return err
		}
		id = result.ID
		created = true
	}
	if _, err = a.engine.ContainerStart(ctx, id, dockerclient.ContainerStartOptions{}); err != nil {
		return err
	}
	if err = a.ensureEndpoints(ctx, node, id); err != nil {
		return errors.Join(err, a.compensateStart(node, id, created))
	}
	return nil
}

func (a *Adapter) ensureEndpoints(ctx context.Context, node domain.Node, id string) error {
	if a.endpoints == nil {
		return nil
	}
	inspection, err := a.engine.ContainerInspect(ctx, id, dockerclient.ContainerInspectOptions{})
	if err != nil {
		return err
	}
	if inspection.Container.State == nil || !inspection.Container.State.Running || inspection.Container.State.Pid <= 0 {
		return fmt.Errorf("running container has no network namespace PID")
	}
	return a.endpoints.Ensure(ctx, node, inspection.Container.State.Pid)
}

func (a *Adapter) compensateStart(node domain.Node, id string, created bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var compensationErr error
	if a.endpoints != nil {
		compensationErr = errors.Join(compensationErr, a.endpoints.Cleanup(ctx, node))
	}
	timeout := 10
	_, stopErr := a.engine.ContainerStop(ctx, id, dockerclient.ContainerStopOptions{Timeout: &timeout})
	compensationErr = errors.Join(compensationErr, stopErr)
	if created {
		_, removeErr := a.engine.ContainerRemove(ctx, id, dockerclient.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
		compensationErr = errors.Join(compensationErr, removeErr)
	}
	return compensationErr
}

func containerCommand(value any) ([]string, error) {
	if value == nil {
		return []string{"sleep", "31536000"}, nil
	}
	switch configured := value.(type) {
	case []string:
		if len(configured) == 0 {
			return []string{"sleep", "31536000"}, nil
		}
		return configured, nil
	case []any:
		if len(configured) == 0 {
			return []string{"sleep", "31536000"}, nil
		}
		command := make([]string, len(configured))
		for index, argument := range configured {
			text, ok := argument.(string)
			if !ok || strings.TrimSpace(text) == "" {
				return nil, fmt.Errorf("container command argument %d must be a non-empty string", index)
			}
			command[index] = text
		}
		return command, nil
	default:
		return nil, fmt.Errorf("container command must be an array of strings")
	}
}
func (a *Adapter) Stop(ctx context.Context, node domain.Node) error {
	id, running, err := a.find(ctx, node)
	if err != nil || id == "" || !running {
		return err
	}
	timeout := 10
	_, err = a.engine.ContainerStop(ctx, id, dockerclient.ContainerStopOptions{Timeout: &timeout})
	return err
}
func (a *Adapter) Delete(ctx context.Context, node domain.Node) error {
	if a.endpoints != nil {
		if err := a.endpoints.Cleanup(ctx, node); err != nil {
			return err
		}
	}
	id, _, err := a.find(ctx, node)
	if err != nil || id == "" {
		return err
	}
	_, err = a.engine.ContainerRemove(ctx, id, dockerclient.ContainerRemoveOptions{Force: true, RemoveVolumes: true})
	return err
}
func quotaToNano(micros int64) int64 {
	if micros <= 0 {
		return 0
	}
	return micros * 10000
}

var _ ports.NodeRuntime = (*Adapter)(nil)
var _ = strconv.Itoa
var _ = strings.TrimSpace
