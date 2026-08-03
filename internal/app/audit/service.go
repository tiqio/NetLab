package audit

import (
	"context"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type Repository interface {
	CreateAuditEvent(context.Context, domain.AuditEvent) error
	ListAuditEvents(context.Context, int) ([]domain.AuditEvent, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Record(ctx context.Context, actorClass, action, resourceType string, resourceID, taskID domain.ID, outcome, correlationID string, details map[string]any) (domain.AuditEvent, error) {
	event := domain.AuditEvent{ID: domain.NewID(), ActorClass: actorClass, Action: action, ResourceType: resourceType, ResourceID: resourceID, TaskID: taskID, Outcome: outcome, CorrelationID: correlationID, Details: Redact(details), OccurredAt: time.Now().UTC()}
	return event, s.repository.CreateAuditEvent(ctx, event)
}

func (s *Service) List(ctx context.Context, limit int) ([]domain.AuditEvent, error) {
	return s.repository.ListAuditEvents(ctx, limit)
}

func Redact(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	return redactMap(input)
}

func redactMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		if sensitiveKey(key) {
			result[key] = "[REDACTED]"
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			result[key] = redactMap(typed)
		case []any:
			items := make([]any, len(typed))
			for index, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					items[index] = redactMap(nested)
				} else {
					items[index] = item
				}
			}
			result[key] = items
		default:
			result[key] = value
		}
	}
	return result
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
	for _, marker := range []string{"password", "passwd", "secret", "token", "credential", "private_key", "user_data", "cloud_init", "argv", "command", "script", "stdin"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
