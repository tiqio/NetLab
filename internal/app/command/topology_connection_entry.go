package command

import (
	"context"
	"strings"
)

type topologyConnectionEntryPointKey struct{}

const TopologyConnectionEntryPointContextKey = "netlab.topology_connection.entry_point"

func WithTopologyConnectionEntryPoint(ctx context.Context, entryPoint string) context.Context {
	return context.WithValue(ctx, topologyConnectionEntryPointKey{}, normalizeTopologyConnectionEntryPoint(entryPoint, "http"))
}

func TopologyConnectionEntryPoint(ctx context.Context, fallback string) string {
	if ctx != nil {
		if value, ok := ctx.Value(topologyConnectionEntryPointKey{}).(string); ok && value != "" {
			return value
		}
		if value, ok := ctx.Value(TopologyConnectionEntryPointContextKey).(string); ok && value != "" {
			return normalizeTopologyConnectionEntryPoint(value, fallback)
		}
	}
	return normalizeTopologyConnectionEntryPoint(fallback, "compatibility")
}

func normalizeTopologyConnectionEntryPoint(value, fallback string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "port_click", "port_drag", "resource_plus", "keyboard", "http", "mcp", "compatibility_http", "compatibility_mcp":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return fallback
	}
}
