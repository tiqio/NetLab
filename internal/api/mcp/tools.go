package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/app/query"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
)

type Services struct {
	Labs         *command.LaboratoryService
	LabQueries   *query.LaboratoryService
	Templates    *query.TemplateService
	Nodes        *command.NodeService
	Links        *command.LinkService
	TopologyOps  *command.TopologyTaskService
	LabOps       *command.LaboratoryTaskService
	Interfaces   *command.InterfaceService
	Guest        *command.GuestCommandService
	Mappings     *command.PortMappingService
	Tasks        *query.TaskService
	Exporter     *command.ExportService
	Importer     *command.ImportService
	Automation   *command.AutomationTaskService
	Captures     *reconcile.CaptureManager
	Filters      *reconcile.TrafficFilterManager
	CaptureOps   *reconcile.CaptureTaskService
	Capabilities *query.RuntimeCapabilityService
	ConsoleIdle  time.Duration
}

func Tools(services Services) []Tool {
	return []Tool{
		NodeCapabilityTool(services.Capabilities),
		{Name: "netlab.capabilities", Description: "Discover NetLab capabilities and trusted-network deployment boundary.", InputSchema: requiredObject(map[string]any{}), Handler: func(_ *gin.Context, _ map[string]any) (any, error) {
			return map[string]any{"single_host": true, "authentication": false, "runtimes": []string{"qemu", "docker", "namespace"}, "capture_returns": "opaque HTTP stream or artifact handle"}, nil
		}},
		{Name: "netlab.templates.list", Description: "List versioned device templates and immutable image variants.", InputSchema: requiredObject(map[string]any{}), Handler: func(c *gin.Context, _ map[string]any) (any, error) {
			if services.Templates == nil {
				return unavailable("template catalog")
			}
			templates, err := services.Templates.List(c)
			if err != nil {
				return nil, err
			}
			images, err := services.Templates.Images(c)
			if err != nil {
				return nil, err
			}
			return map[string]any{"templates": templates, "images": images}, nil
		}},
		{Name: "netlab.labs.list", Description: "List shared laboratories.", InputSchema: requiredObject(map[string]any{}), Handler: func(c *gin.Context, _ map[string]any) (any, error) { return services.LabQueries.List(c) }},
		{Name: "netlab.labs.get", Description: "Get a complete shared topology snapshot.", InputSchema: requiredObject(map[string]any{"lab_id": stringProperty("Laboratory ID")}, "lab_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "lab_id")
			if err != nil {
				return nil, err
			}
			return services.LabQueries.Snapshot(c, domain.ID(id))
		}},
		{Name: "netlab.labs.create", Description: "Create a shared laboratory.", InputSchema: requiredObject(map[string]any{"name": stringProperty("Laboratory name"), "description": stringProperty("Description"), "recovery_policy": enumProperty("auto_restore", "remain_stopped")}, "name"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			name, err := argumentString(args, "name")
			if err != nil {
				return nil, err
			}
			return services.Labs.Create(c, name, optionalString(args, "description"), domain.RecoveryPolicy(optionalString(args, "recovery_policy")))
		}},
		{Name: "netlab.labs.delete", Description: "Mark a laboratory for terminal owned-resource cleanup.", InputSchema: mutationSchema(map[string]any{"lab_id": stringProperty("Laboratory ID")}, "lab_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.LabOps == nil {
				return unavailable("laboratory deletion")
			}
			id, err := argumentString(args, "lab_id")
			if err != nil {
				return nil, err
			}
			value, err := services.LabOps.Delete(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": value}, nil
		}},
		{Name: "netlab.labs.export", Description: "Create a redacted export artifact.", InputSchema: mutationSchema(map[string]any{"lab_id": stringProperty("Laboratory ID")}, "lab_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Automation == nil {
				return unavailable("laboratory export")
			}
			id, err := argumentString(args, "lab_id")
			if err != nil {
				return nil, err
			}
			taskValue, err := services.Automation.Export(c, domain.ID(id), 24*time.Hour, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": taskValue}, nil
		}},
		{Name: "netlab.labs.import", Description: "Import a validated, redacted laboratory bundle.", InputSchema: mutationSchema(map[string]any{"bundle": map[string]any{"type": "object"}}, "bundle"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Automation == nil {
				return unavailable("laboratory import")
			}
			var bundle command.LaboratoryExport
			if err := decodeArgument(args["bundle"], &bundle); err != nil {
				return nil, err
			}
			taskValue, err := services.Automation.Import(c, bundle, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": taskValue}, nil
		}},
		{Name: "netlab.labs.duplicate", Description: "Duplicate a laboratory through a redacted durable export/import task.", InputSchema: mutationSchema(map[string]any{"lab_id": stringProperty("Laboratory ID"), "name": stringProperty("New laboratory name")}, "lab_id", "name"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Automation == nil {
				return unavailable("laboratory duplicate")
			}
			id, err := argumentString(args, "lab_id")
			if err != nil {
				return nil, err
			}
			name, err := argumentString(args, "name")
			if err != nil {
				return nil, err
			}
			taskValue, err := services.Automation.Duplicate(c, domain.ID(id), name, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": taskValue}, nil
		}},
		{Name: "netlab.nodes.create", Description: "Create a template-pinned QEMU/Docker or lightweight node.", InputSchema: mutationSchema(map[string]any{"lab_id": stringProperty("Laboratory ID"), "name": stringProperty("Node name"), "kind": stringProperty("Runtime kind"), "template_version_id": stringProperty("Template version ID"), "image_version_id": stringProperty("Image version ID"), "cpu_count": integerProperty(0), "cpu_quota_micros": integerProperty(0), "memory_mib": integerProperty(0), "interface_count": integerProperty(0), "config": map[string]any{"type": "object"}, "bootstrap": map[string]any{"type": "object"}}, "lab_id", "name"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			labID, err := argumentString(args, "lab_id")
			if err != nil {
				return nil, err
			}
			name, err := argumentString(args, "name")
			if err != nil {
				return nil, err
			}
			request := command.CreateNodeRequest{Name: name, Kind: optionalString(args, "kind"), TemplateVersionID: domain.ID(optionalString(args, "template_version_id")), ImageVersionID: domain.ID(optionalString(args, "image_version_id")), CPUCount: intArgument(args, "cpu_count"), CPUQuotaMicros: int64(intArgument(args, "cpu_quota_micros")), MemoryMiB: intArgument(args, "memory_mib"), InterfaceCount: intArgument(args, "interface_count")}
			if config, ok := args["config"].(map[string]any); ok {
				request.Config = config
			}
			_ = decodeArgument(args["bootstrap"], &request.Bootstrap)
			node, interfaces, err := services.Nodes.CreateConfigured(c, domain.ID(labID), request)
			if err != nil {
				return nil, err
			}
			return map[string]any{"node": node, "interfaces": interfaces}, nil
		}},
		{Name: "netlab.nodes.set_state", Description: "Set desired running or stopped state.", InputSchema: mutationSchema(map[string]any{"node_id": stringProperty("Node ID"), "desired_state": enumProperty("running", "stopped")}, "node_id", "desired_state", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "node_id")
			if err != nil {
				return nil, err
			}
			state, err := argumentString(args, "desired_state")
			if err != nil {
				return nil, err
			}
			if services.TopologyOps != nil {
				taskValue, taskErr := services.TopologyOps.SetNodeState(c, domain.ID(id), revisionArgument(args), domain.DesiredState(state), optionalString(args, "idempotency_key"))
				if taskErr != nil {
					return nil, taskErr
				}
				return map[string]any{"task": taskValue}, nil
			}
			return services.Nodes.SetState(c, domain.ID(id), revisionArgument(args), domain.DesiredState(state))
		}},
		{Name: "netlab.nodes.delete", Description: "Delete a stopped node.", InputSchema: mutationSchema(map[string]any{"node_id": stringProperty("Node ID")}, "node_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "node_id")
			if err != nil {
				return nil, err
			}
			if services.TopologyOps != nil {
				taskValue, taskErr := services.TopologyOps.DeleteNode(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
				if taskErr != nil {
					return nil, taskErr
				}
				return map[string]any{"task": taskValue}, nil
			}
			if err = services.Nodes.Delete(c, domain.ID(id), revisionArgument(args)); err != nil {
				return nil, err
			}
			return map[string]any{"node_id": id, "deleted": true}, nil
		}},
		{Name: "netlab.nodes.exec", Description: "Execute a bounded QEMU guest-agent command.", InputSchema: mutationSchema(map[string]any{"node_id": stringProperty("Node ID"), "argv": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1}, "timeout_seconds": integerProperty(1), "output_limit": integerProperty(1)}, "node_id", "argv"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Guest == nil {
				return unavailable("guest command")
			}
			id, err := argumentString(args, "node_id")
			if err != nil {
				return nil, err
			}
			argv := stringSlice(args["argv"])
			taskValue, taskErr := services.Guest.Execute(c, domain.ID(id), argv, time.Duration(intArgument(args, "timeout_seconds"))*time.Second, intArgument(args, "output_limit"), optionalString(args, "idempotency_key"))
			if taskErr != nil {
				return nil, taskErr
			}
			return map[string]any{"task": taskValue}, nil
		}},
		{Name: "netlab.interfaces.add", Description: "Add a NIC and hot-plug it when the QEMU node is running.", InputSchema: mutationSchema(map[string]any{"node_id": stringProperty("Node ID"), "driver": stringProperty("NIC driver")}, "node_id", "driver"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Interfaces == nil {
				return unavailable("interface hot-plug")
			}
			id, err := argumentString(args, "node_id")
			if err != nil {
				return nil, err
			}
			driver, err := argumentString(args, "driver")
			if err != nil {
				return nil, err
			}
			iface, task, err := services.Interfaces.Add(c, domain.ID(id), driver, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			result := map[string]any{"interface": iface}
			if task != nil {
				result["task"] = task
			}
			return result, nil
		}},
		{Name: "netlab.interfaces.remove", Description: "Remove a NIC and hot-unplug it when required.", InputSchema: mutationSchema(map[string]any{"interface_id": stringProperty("Interface ID")}, "interface_id", "expected_revision"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Interfaces == nil {
				return unavailable("interface hot-plug")
			}
			id, err := argumentString(args, "interface_id")
			if err != nil {
				return nil, err
			}
			taskValue, taskErr := services.Interfaces.Remove(c, domain.ID(id), revisionArgument(args), optionalString(args, "idempotency_key"))
			if taskErr != nil {
				return nil, taskErr
			}
			if taskValue.ID == "" {
				return map[string]any{"interface_id": id, "removed": true}, nil
			}
			return map[string]any{"task": taskValue}, nil
		}},
		{Name: "netlab.links.connect", Description: "Connect two interfaces without stopping nodes.", InputSchema: mutationSchema(map[string]any{"lab_id": stringProperty("Laboratory ID"), "endpoint_a_id": stringProperty("First interface ID"), "endpoint_b_id": stringProperty("Second interface ID")}, "lab_id", "endpoint_a_id", "endpoint_b_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			labID, err := argumentString(args, "lab_id")
			if err != nil {
				return nil, err
			}
			a, err := argumentString(args, "endpoint_a_id")
			if err != nil {
				return nil, err
			}
			b, err := argumentString(args, "endpoint_b_id")
			if err != nil {
				return nil, err
			}
			if services.TopologyOps != nil {
				link, taskValue, taskErr := services.TopologyOps.ConnectLink(c, domain.ID(labID), domain.ID(a), domain.ID(b), optionalString(args, "idempotency_key"))
				if taskErr != nil {
					return nil, taskErr
				}
				return map[string]any{"link": link, "task": taskValue}, nil
			}
			return services.Links.Connect(c, domain.ID(labID), domain.ID(a), domain.ID(b))
		}},
		{Name: "netlab.links.disconnect", Description: "Disconnect and clean a live link.", InputSchema: mutationSchema(map[string]any{"link_id": stringProperty("Link ID")}, "link_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "link_id")
			if err != nil {
				return nil, err
			}
			if services.TopologyOps != nil {
				taskValue, taskErr := services.TopologyOps.DisconnectLink(c, domain.ID(id), optionalString(args, "idempotency_key"))
				if taskErr != nil {
					return nil, taskErr
				}
				return map[string]any{"task": taskValue}, nil
			}
			if err = services.Links.Disconnect(c, domain.ID(id)); err != nil {
				return nil, err
			}
			return map[string]any{"link_id": id, "state": "disconnecting"}, nil
		}},
		{Name: "netlab.port_mappings.create", Description: "Publish a node port through the host.", InputSchema: mutationSchema(map[string]any{"node_id": stringProperty("Node ID"), "protocol": enumProperty("tcp", "udp"), "host_address": stringProperty("Host address"), "host_port": integerProperty(1), "guest_address": stringProperty("Guest address"), "guest_port": integerProperty(1)}, "node_id", "protocol", "host_port", "guest_address", "guest_port"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Mappings == nil {
				return unavailable("port mapping")
			}
			id, err := argumentString(args, "node_id")
			if err != nil {
				return nil, err
			}
			mapping := domain.PortMapping{NodeID: domain.ID(id), Protocol: optionalString(args, "protocol"), HostAddress: optionalString(args, "host_address"), HostPort: intArgument(args, "host_port"), GuestAddress: optionalString(args, "guest_address"), GuestPort: intArgument(args, "guest_port")}
			value, task, err := services.Mappings.Create(c, mapping, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"port_mapping": value, "task": task}, nil
		}},
		{Name: "netlab.port_mappings.delete", Description: "Remove a host port mapping.", InputSchema: mutationSchema(map[string]any{"mapping_id": stringProperty("Mapping ID")}, "mapping_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.Mappings == nil {
				return unavailable("port mapping")
			}
			id, err := argumentString(args, "mapping_id")
			if err != nil {
				return nil, err
			}
			taskValue, taskErr := services.Mappings.Delete(c, domain.ID(id), optionalString(args, "idempotency_key"))
			if taskErr != nil {
				return nil, taskErr
			}
			return map[string]any{"task": taskValue}, nil
		}},
		{Name: "netlab.consoles.get", Description: "Return reconnect-safe Telnet/VNC WebSocket descriptors.", InputSchema: requiredObject(map[string]any{"node_id": stringProperty("Node ID"), "mode": enumProperty("telnet", "vnc")}, "node_id"), Handler: func(_ *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "node_id")
			if err != nil {
				return nil, err
			}
			mode := optionalString(args, "mode")
			idle := services.ConsoleIdle
			if idle <= 0 {
				idle = 30 * time.Minute
			}
			descriptor := func(value string) map[string]any {
				return map[string]any{"mode": value, "stream_url": "/api/v1/nodes/" + id + "/consoles/" + value + "/stream", "idle_seconds": int(idle.Seconds()), "reconnectable": true}
			}
			if mode != "" {
				return descriptor(mode), nil
			}
			return []map[string]any{descriptor("telnet"), descriptor("vnc")}, nil
		}},
		{Name: "netlab.captures.start", Description: "Start a live or retained packet capture.", InputSchema: mutationSchema(map[string]any{"laboratory_id": stringProperty("Laboratory ID"), "source_type": enumProperty("interface", "link", "network_object_link"), "source_id": stringProperty("Source ID"), "filter": stringProperty("BPF filter"), "format": enumProperty("pcap", "pcapng"), "retain": map[string]any{"type": "boolean"}, "max_bytes": integerProperty(1), "duration_seconds": integerProperty(0)}, "laboratory_id", "source_type", "source_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.CaptureOps == nil {
				return unavailable("packet capture")
			}
			request := reconcile.CaptureRequest{LaboratoryID: domain.ID(optionalString(args, "laboratory_id")), SourceType: optionalString(args, "source_type"), SourceID: domain.ID(optionalString(args, "source_id")), Interface: optionalString(args, "interface"), Filter: optionalString(args, "filter"), Format: optionalString(args, "format"), Retain: boolArgument(args, "retain"), MaxBytes: int64(intArgument(args, "max_bytes")), Duration: time.Duration(intArgument(args, "duration_seconds")) * time.Second}
			value, taskValue, err := services.CaptureOps.StartCapture(c, request, optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			envelope := captureEnvelope(value)
			envelope["task"] = taskValue
			return envelope, nil
		}},
		{Name: "netlab.captures.get", Description: "Read bounded capture metadata and stream/artifact handles.", InputSchema: requiredObject(map[string]any{"capture_id": stringProperty("Capture ID")}, "capture_id"), Handler: func(_ *gin.Context, args map[string]any) (any, error) {
			if services.Captures == nil {
				return unavailable("packet capture")
			}
			id, err := argumentString(args, "capture_id")
			if err != nil {
				return nil, err
			}
			value, err := services.Captures.Get(domain.ID(id))
			if err != nil {
				return nil, err
			}
			return captureEnvelope(value), nil
		}},
		{Name: "netlab.captures.stop", Description: "Stop a running packet capture.", InputSchema: mutationSchema(map[string]any{"capture_id": stringProperty("Capture ID")}, "capture_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.CaptureOps == nil {
				return unavailable("packet capture")
			}
			id, err := argumentString(args, "capture_id")
			if err != nil {
				return nil, err
			}
			value, err := services.CaptureOps.StopCapture(c, domain.ID(id), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": value}, nil
		}},
		{Name: "netlab.traffic_filters.start", Description: "Observe matching packet fingerprints across selected interfaces and links.", InputSchema: mutationSchema(map[string]any{"laboratory_id": stringProperty("Laboratory ID"), "match": map[string]any{"type": "object"}, "interface_ids": stringArrayProperty(), "link_ids": stringArrayProperty(), "network_object_link_ids": stringArrayProperty(), "max_observations": integerProperty(1)}, "laboratory_id", "match"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.CaptureOps == nil {
				return unavailable("traffic filter")
			}
			var match captureRuntime.Match
			if err := decodeArgument(args["match"], &match); err != nil {
				return nil, err
			}
			value, taskValue, err := services.CaptureOps.StartFilterWithObjectLinks(c, domain.ID(optionalString(args, "laboratory_id")), match, intArgument(args, "max_observations"), idSlice(args["interface_ids"]), idSlice(args["link_ids"]), idSlice(args["network_object_link_ids"]), optionalString(args, "color"), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"traffic_filter": value, "task": taskValue}, nil
		}},
		{Name: "netlab.traffic_filters.get", Description: "Read correlated path observations and ambiguity state.", InputSchema: requiredObject(map[string]any{"filter_id": stringProperty("Traffic filter ID")}, "filter_id"), Handler: func(_ *gin.Context, args map[string]any) (any, error) {
			if services.Filters == nil {
				return unavailable("traffic filter")
			}
			id, err := argumentString(args, "filter_id")
			if err != nil {
				return nil, err
			}
			value, ambiguous, err := services.Filters.Get(domain.ID(id))
			if err != nil {
				return nil, err
			}
			return map[string]any{"traffic_filter": value, "ambiguous": ambiguous}, nil
		}},
		{Name: "netlab.traffic_filters.stop", Description: "Stop packet path observation.", InputSchema: mutationSchema(map[string]any{"filter_id": stringProperty("Traffic filter ID")}, "filter_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			if services.CaptureOps == nil {
				return unavailable("traffic filter")
			}
			id, err := argumentString(args, "filter_id")
			if err != nil {
				return nil, err
			}
			value, err := services.CaptureOps.StopFilter(c, domain.ID(id), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": value}, nil
		}},
		{Name: "netlab.tasks.get", Description: "Read durable operation status.", InputSchema: requiredObject(map[string]any{"task_id": stringProperty("Task ID")}, "task_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "task_id")
			if err != nil {
				return nil, err
			}
			return services.Tasks.Get(c, domain.ID(id))
		}},
		{Name: "netlab.tasks.cancel", Description: "Request cancellation of a durable operation.", InputSchema: requiredObject(map[string]any{"task_id": stringProperty("Task ID")}, "task_id"), Handler: func(c *gin.Context, args map[string]any) (any, error) {
			id, err := argumentString(args, "task_id")
			if err != nil {
				return nil, err
			}
			return services.Tasks.Cancel(c, domain.ID(id))
		}},
	}
}

func unavailable(name string) (any, error) {
	return nil, domain.Problem{Code: "capability_unsupported", Message: name + " is unavailable in this runtime configuration"}
}
func enumProperty(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}
func integerProperty(minimum int) map[string]any {
	return map[string]any{"type": "integer", "minimum": minimum}
}
func stringArrayProperty() map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
}
func mutationSchema(properties map[string]any, required ...string) map[string]any {
	properties["idempotency_key"] = stringProperty("Replay-safe idempotency key")
	properties["expected_revision"] = integerProperty(1)
	return requiredObject(properties, required...)
}
func optionalString(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}
func intArgument(args map[string]any, key string) int {
	value, _ := args[key].(float64)
	return int(value)
}
func boolArgument(args map[string]any, key string) bool { value, _ := args[key].(bool); return value }
func revisionArgument(args map[string]any) domain.Revision {
	return domain.Revision(intArgument(args, "expected_revision"))
}
func stringSlice(value any) []string {
	raw, _ := value.([]any)
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
func idSlice(value any) []domain.ID {
	values := stringSlice(value)
	result := make([]domain.ID, len(values))
	for index := range values {
		result[index] = domain.ID(values[index])
	}
	return result
}
func decodeArgument(value any, target any) error {
	if value == nil {
		return nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
func captureEnvelope(value domain.Capture) map[string]any {
	result := map[string]any{"capture": value, "stream_url": "/api/v1/captures/" + string(value.ID) + "/stream"}
	if value.ArtifactURL != "" {
		result["artifact_url"] = value.ArtifactURL
	}
	return result
}

type ContextHandler func(context.Context, map[string]any) (any, error)
