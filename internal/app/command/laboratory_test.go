package command

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type labMemory struct{ lab domain.Laboratory }

func (m *labMemory) CreateLaboratory(_ context.Context, lab domain.Laboratory) error {
	m.lab = lab
	return nil
}
func (m *labMemory) GetLaboratory(_ context.Context, _ domain.ID) (domain.Laboratory, error) {
	return m.lab, nil
}
func (m *labMemory) UpdateLaboratory(_ context.Context, id domain.ID, revision domain.Revision, name, description string, policy domain.RecoveryPolicy) (domain.Laboratory, error) {
	if m.lab.Revision != revision {
		return domain.Laboratory{}, domain.Problem{Code: "revision_conflict", Message: "conflict"}
	}
	m.lab.ID = id
	m.lab.Name = name
	m.lab.Description = description
	m.lab.RecoveryPolicy = policy
	m.lab.Revision++
	return m.lab, nil
}
func (m *labMemory) MarkLaboratoryDeleting(_ context.Context, _ domain.ID, revision domain.Revision) error {
	if m.lab.Revision != revision {
		return domain.Problem{Code: "revision_conflict", Message: "conflict"}
	}
	m.lab.LifecycleState = "deleting"
	return nil
}
func TestLaboratoryCommands(t *testing.T) {
	repo := &labMemory{}
	service := NewLaboratoryService(repo)
	lab, err := service.Create(context.Background(), " lab ", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if lab.Name != "lab" || lab.RecoveryPolicy != domain.RecoveryAutoRestore {
		t.Fatal("defaults not applied")
	}
	if _, err = service.Update(context.Background(), lab.ID, 99, "x", "", domain.RecoveryAutoRestore); err == nil {
		t.Fatal("expected conflict")
	}
}
