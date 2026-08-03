package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type LaboratoryRepository interface {
	CreateLaboratory(context.Context, domain.Laboratory) error
	GetLaboratory(context.Context, domain.ID) (domain.Laboratory, error)
	UpdateLaboratory(context.Context, domain.ID, domain.Revision, string, string, domain.RecoveryPolicy) (domain.Laboratory, error)
	MarkLaboratoryDeleting(context.Context, domain.ID, domain.Revision) error
}

type LaboratoryService struct{ repository LaboratoryRepository }

func NewLaboratoryService(repository LaboratoryRepository) *LaboratoryService {
	return &LaboratoryService{repository: repository}
}
func (s *LaboratoryService) Create(ctx context.Context, name, description string, policy domain.RecoveryPolicy) (domain.Laboratory, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 {
		return domain.Laboratory{}, fmt.Errorf("laboratory name must be 1-128 characters")
	}
	if policy == "" {
		policy = domain.RecoveryAutoRestore
	}
	if policy != domain.RecoveryAutoRestore && policy != domain.RecoveryRemainStopped {
		return domain.Laboratory{}, fmt.Errorf("invalid recovery policy")
	}
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: name, Description: description, Revision: 1, RecoveryPolicy: policy, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	return lab, s.repository.CreateLaboratory(ctx, lab)
}
func (s *LaboratoryService) Update(ctx context.Context, id domain.ID, revision domain.Revision, name, description string, policy domain.RecoveryPolicy) (domain.Laboratory, error) {
	if _, err := s.repository.GetLaboratory(ctx, id); err != nil {
		return domain.Laboratory{}, err
	}
	return s.repository.UpdateLaboratory(ctx, id, revision, strings.TrimSpace(name), description, policy)
}
func (s *LaboratoryService) Delete(ctx context.Context, id domain.ID, revision domain.Revision) error {
	return s.repository.MarkLaboratoryDeleting(ctx, id, revision)
}
