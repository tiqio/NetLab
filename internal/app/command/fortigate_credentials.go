package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	credentialstore "github.com/netlab/netlab/internal/store/credential"
)

type FortiGateCredentialStore interface {
	Put(context.Context, domain.ID, domain.ID, string, string, string, []byte) (domain.NodeCredentialMetadata, error)
	Get(context.Context, domain.ID, domain.ID, string, string) (domain.NodeCredentialSecret, error)
	Metadata(context.Context, domain.ID, domain.ID, string) (domain.NodeCredentialMetadata, error)
	Mark(context.Context, domain.ID, domain.ID, string, string, string) error
	PromoteStaged(context.Context, domain.ID, domain.ID, string) error
	Delete(context.Context, domain.ID, domain.ID, string) error
}

type FortiGateConsole interface {
	Verify(context.Context, domain.Node, domain.NodeCredentialSecret) error
	RotateInitial(context.Context, domain.Node, domain.NodeCredentialSecret, domain.NodeCredentialSecret) error
}

type fortiGateNodeReader interface {
	GetNode(context.Context, domain.ID) (domain.Node, error)
}

type FortiGateCredentialService struct {
	nodes      fortiGateNodeReader
	store      FortiGateCredentialStore
	storeError error
	console    FortiGateConsole
	runner     *task.Runner
}

func NewFortiGateCredentialService(nodes fortiGateNodeReader, store FortiGateCredentialStore, storeError error, console FortiGateConsole, runner *task.Runner) *FortiGateCredentialService {
	service := &FortiGateCredentialService{nodes: nodes, store: store, storeError: storeError, console: console, runner: runner}
	if runner != nil {
		runner.Register("fortigate.credential.verify", service.handleVerify)
		runner.Register("fortigate.bootstrap", service.handleBootstrap)
	}
	return service
}

func (s *FortiGateCredentialService) Metadata(ctx context.Context, nodeID domain.ID) (domain.NodeCredentialMetadata, error) {
	node, err := s.fortigateNode(ctx, nodeID)
	if err != nil {
		return domain.NodeCredentialMetadata{}, err
	}
	if s.store == nil {
		return domain.NodeCredentialMetadata{NodeID: node.ID, LaboratoryID: node.LaboratoryID, Kind: domain.CredentialKindConsoleAdmin, State: "credential_store_locked"}, nil
	}
	return s.store.Metadata(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin)
}

func (s *FortiGateCredentialService) Put(ctx context.Context, nodeID domain.ID, username string, currentPassword, newPassword []byte) (domain.NodeCredentialMetadata, error) {
	node, err := s.fortigateNode(ctx, nodeID)
	if err != nil {
		return domain.NodeCredentialMetadata{}, err
	}
	if err = s.available(node.ID); err != nil {
		return domain.NodeCredentialMetadata{}, err
	}
	metadata, err := s.store.Put(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin, domain.CredentialSlotActive, username, currentPassword)
	if err != nil {
		return metadata, err
	}
	if len(newPassword) > 0 {
		if _, err = s.store.Put(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin, domain.CredentialSlotStaged, username, newPassword); err != nil {
			return metadata, err
		}
	}
	return s.store.Metadata(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin)
}

func (s *FortiGateCredentialService) Delete(ctx context.Context, nodeID domain.ID) error {
	node, err := s.fortigateNode(ctx, nodeID)
	if err != nil {
		return err
	}
	if err = s.available(node.ID); err != nil {
		return err
	}
	return s.store.Delete(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin)
}

func (s *FortiGateCredentialService) Verify(ctx context.Context, nodeID domain.ID, key string) (domain.OperationTask, error) {
	return s.enqueue(ctx, nodeID, "fortigate.credential.verify", key)
}

func (s *FortiGateCredentialService) Bootstrap(ctx context.Context, nodeID domain.ID, key string) (domain.OperationTask, error) {
	return s.enqueue(ctx, nodeID, "fortigate.bootstrap", key)
}

func (s *FortiGateCredentialService) enqueue(ctx context.Context, nodeID domain.ID, kind, key string) (domain.OperationTask, error) {
	node, err := s.fortigateNode(ctx, nodeID)
	if err != nil {
		return domain.OperationTask{}, err
	}
	if err = s.available(node.ID); err != nil {
		return domain.OperationTask{}, err
	}
	if s.runner == nil || s.console == nil {
		return domain.OperationTask{}, domain.Problem{Code: "operation_unavailable", Message: "FortiGate console operation is unavailable", ResourceType: "node", ResourceID: nodeID}
	}
	metadata, err := s.store.Metadata(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin)
	if err != nil {
		return domain.OperationTask{}, err
	}
	if !metadata.Configured {
		return domain.OperationTask{}, domain.Problem{Code: "credential_missing", Message: "FortiGate console credential is not configured", ResourceType: "node", ResourceID: nodeID}
	}
	fingerprint, _ := json.Marshal(map[string]any{"node_id": nodeID, "kind": kind, "credential_revision": metadata.Revision})
	value := domain.OperationTask{
		ID: domain.NewID(), Kind: kind, ResourceType: "node", ResourceID: nodeID,
		IdempotencyKey: key, RequestFingerprint: RequestFingerprint(fingerprint), State: domain.TaskQueued,
		ProgressTotal: 2, CreatedAt: time.Now().UTC(),
		Input: map[string]any{"credential_ref": fmt.Sprintf("node:%s:%s", nodeID, domain.CredentialKindConsoleAdmin), "action": kind},
	}
	return s.runner.EnqueueOrGet(ctx, value)
}

func (s *FortiGateCredentialService) handleVerify(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	return s.handle(ctx, value, false)
}

func (s *FortiGateCredentialService) handleBootstrap(ctx context.Context, value *domain.OperationTask) (map[string]any, error) {
	return s.handle(ctx, value, true)
}

func (s *FortiGateCredentialService) handle(ctx context.Context, value *domain.OperationTask, bootstrap bool) (map[string]any, error) {
	node, err := s.fortigateNode(ctx, value.ResourceID)
	if err != nil {
		return nil, err
	}
	active, err := s.store.Get(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin, domain.CredentialSlotActive)
	if err != nil {
		return nil, err
	}
	defer active.Clear()
	value.ProgressCurrent = 1
	if err = s.runner.Checkpoint(ctx, value); err != nil {
		return nil, err
	}
	if bootstrap {
		staged, getErr := s.store.Get(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin, domain.CredentialSlotStaged)
		if getErr != nil {
			return nil, domain.Problem{Code: "staged_credential_missing", Message: "a staged FortiGate password is required", ResourceType: "node", ResourceID: node.ID}
		}
		defer staged.Clear()
		err = s.console.RotateInitial(ctx, node, active, staged)
		if err == nil {
			err = s.store.PromoteStaged(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin)
		}
	} else {
		err = s.console.Verify(ctx, node, active)
	}
	if err != nil {
		code := "console_interaction_failed"
		var problem domain.Problem
		if errors.As(err, &problem) {
			code = problem.Code
		}
		_ = s.store.Mark(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin, "verification_failed", code)
		return nil, err
	}
	if err = s.store.Mark(ctx, node.LaboratoryID, node.ID, domain.CredentialKindConsoleAdmin, "authenticated", ""); err != nil {
		return nil, err
	}
	value.ProgressCurrent = value.ProgressTotal
	return map[string]any{"credential_ref": value.Input["credential_ref"], "state": "authenticated", "verified": true}, nil
}

func (s *FortiGateCredentialService) available(nodeID domain.ID) error {
	if s.store != nil {
		return nil
	}
	message := "credential store is locked"
	if s.storeError != nil && !errors.Is(s.storeError, credentialstore.ErrLocked) {
		message = "credential store is unavailable"
	}
	return domain.Problem{Code: "credential_store_locked", Message: message, Retryable: true, ResourceType: "node", ResourceID: nodeID}
}

func (s *FortiGateCredentialService) fortigateNode(ctx context.Context, nodeID domain.ID) (domain.Node, error) {
	node, err := s.nodes.GetNode(ctx, nodeID)
	if err != nil {
		return node, err
	}
	templateKey, _ := node.Config["template_key"].(string)
	if !strings.EqualFold(templateKey, "fortigate") {
		return node, domain.Problem{Code: "capability_unsupported", Message: "credentials are supported only for FortiGate nodes", ResourceType: "node", ResourceID: node.ID}
	}
	return node, nil
}
