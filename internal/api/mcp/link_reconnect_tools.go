package mcp

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/domain"
)

type LinkReconnectCommand interface {
	Reconnect(context.Context, domain.ID, domain.Revision, domain.ID, domain.ID, string) (domain.OperationTask, error)
}

func LinkReconnectTools(service LinkReconnectCommand) []Tool {
	if service == nil {
		return nil
	}
	return []Tool{{
		Name: "netlab.links.reconnect", Description: "Atomically replace one link endpoint and restore the original link on failure.",
		InputSchema: mutationSchema(map[string]any{
			"link_id":                 stringProperty("Link ID"),
			"retained_endpoint_id":    stringProperty("Existing endpoint to retain"),
			"replacement_endpoint_id": stringProperty("Available replacement interface"),
		}, "link_id", "expected_revision", "retained_endpoint_id", "replacement_endpoint_id", "idempotency_key"),
		Handler: func(c *gin.Context, args map[string]any) (any, error) {
			linkID, err := argumentString(args, "link_id")
			if err != nil {
				return nil, err
			}
			retained, err := argumentString(args, "retained_endpoint_id")
			if err != nil {
				return nil, err
			}
			replacement, err := argumentString(args, "replacement_endpoint_id")
			if err != nil {
				return nil, err
			}
			value, err := service.Reconnect(c, domain.ID(linkID), revisionArgument(args), domain.ID(retained), domain.ID(replacement), optionalString(args, "idempotency_key"))
			if err != nil {
				return nil, err
			}
			return map[string]any{"task": value}, nil
		},
	}}
}
