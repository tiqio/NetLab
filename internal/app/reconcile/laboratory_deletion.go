package reconcile

import (
	"context"
	"os"
	"path/filepath"

	"github.com/netlab/netlab/internal/domain"
)

type LaboratoryDeletionStore interface {
	ListLaboratories(context.Context) ([]domain.Laboratory, error)
	Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error)
	FinalizeLaboratoryDeletion(context.Context, domain.ID) error
	MarkLaboratoryDeleteFailed(context.Context, domain.ID, domain.Problem) error
	ListLaboratoryPortMappings(context.Context, domain.ID) ([]domain.PortMapping, error)
	ListLaboratoryArtifacts(context.Context, domain.ID, []domain.ID) ([]domain.Artifact, error)
	DeleteArtifacts(context.Context, []domain.ID) error
}

type MappingCleanup interface {
	Delete(context.Context, domain.ID) error
}
type ResourceCleanup interface{ Cleanup(domain.ID) error }
type InterfaceCleanup interface {
	Delete(context.Context, string) error
}

type LaboratoryDeletionReconciler struct {
	store      LaboratoryDeletionStore
	nodes      RuntimeDispatch
	networks   *NetworkObjectService
	links      DataPlaneRuntime
	captures   *CaptureManager
	mappings   MappingCleanup
	resources  ResourceCleanup
	interfaces InterfaceCleanup
}

func NewLaboratoryDeletionReconciler(store LaboratoryDeletionStore, nodes RuntimeDispatch, networks *NetworkObjectService, links DataPlaneRuntime, captures *CaptureManager, cleanup ...any) *LaboratoryDeletionReconciler {
	value := &LaboratoryDeletionReconciler{store: store, nodes: nodes, networks: networks, links: links, captures: captures}
	for _, item := range cleanup {
		if runtime, ok := item.(MappingCleanup); ok {
			value.mappings = runtime
		}
		if runtime, ok := item.(ResourceCleanup); ok {
			value.resources = runtime
		}
		if runtime, ok := item.(InterfaceCleanup); ok {
			value.interfaces = runtime
		}
	}
	return value
}

func (r *LaboratoryDeletionReconciler) Name() string { return "laboratory-deletion" }

func (r *LaboratoryDeletionReconciler) Reconcile(ctx context.Context) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("reconciler", domain.ID(r.Name()), "laboratory_deletion"))
	laboratories, err := r.store.ListLaboratories(ctx)
	if err != nil {
		return err
	}
	for _, laboratory := range laboratories {
		if laboratory.LifecycleState != "deleting" && laboratory.LifecycleState != "delete_failed" {
			continue
		}
		snapshot, snapshotErr := r.store.Snapshot(ctx, laboratory.ID)
		if snapshotErr != nil {
			return snapshotErr
		}
		artifactIDs := []domain.ID{}
		if r.captures != nil {
			artifactIDs = r.captures.PurgeLaboratory(laboratory.ID)
		}
		fail := func(cleanupErr error) {
			problem := structuredProblem(cleanupErr, domain.Problem{Code: "deletion_cleanup_failed", Retryable: true, ResourceType: "laboratory", ResourceID: laboratory.ID, Phase: "deleting", Cleanup: "completed cleanup steps are retained", OperatorHint: "inspect remaining owned resources and retry deletion", RetryAfterSeconds: 5})
			_ = r.store.MarkLaboratoryDeleteFailed(context.Background(), laboratory.ID, *problem)
		}
		cleanupFailed := false
		mappings, mappingErr := r.store.ListLaboratoryPortMappings(ctx, laboratory.ID)
		if mappingErr != nil {
			fail(mappingErr)
			continue
		}
		for _, mapping := range mappings {
			if r.mappings != nil {
				if cleanupErr := r.mappings.Delete(ctx, mapping.ID); cleanupErr != nil {
					fail(cleanupErr)
					cleanupFailed = true
				}
			}
		}
		if cleanupFailed {
			continue
		}
		for _, link := range snapshot.Links {
			if r.links != nil {
				if cleanupErr := r.links.DeleteLink(ctx, link.ID); cleanupErr != nil {
					fail(cleanupErr)
					cleanupFailed = true
				}
			}
		}
		if cleanupFailed {
			continue
		}
		for _, node := range snapshot.Nodes {
			if runtime, runtimeErr := r.nodes.For(node); runtimeErr == nil && runtime != nil {
				if cleanupErr := runtime.Delete(ctx, node); cleanupErr != nil {
					fail(cleanupErr)
					cleanupFailed = true
				}
			}
			if r.resources != nil {
				if cleanupErr := r.resources.Cleanup(node.ID); cleanupErr != nil {
					fail(cleanupErr)
					cleanupFailed = true
				}
			}
			if seedPath, ok := node.Config["seed_iso"].(string); ok && seedPath != "" {
				_ = os.RemoveAll(filepath.Dir(seedPath))
			}
		}
		if r.interfaces != nil {
			for _, iface := range snapshot.Interfaces {
				_ = r.interfaces.Delete(ctx, hotAddTapName(iface.ID))
			}
		}
		if cleanupFailed {
			continue
		}
		artifacts, artifactErr := r.store.ListLaboratoryArtifacts(ctx, laboratory.ID, artifactIDs)
		if artifactErr != nil {
			fail(artifactErr)
			continue
		}
		for _, artifact := range artifacts {
			if cleanupErr := os.Remove(artifact.Path); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
				fail(cleanupErr)
				cleanupFailed = true
			}
		}
		if cleanupFailed {
			continue
		}
		ids := make([]domain.ID, len(artifacts))
		for index := range artifacts {
			ids[index] = artifacts[index].ID
		}
		if err = r.store.DeleteArtifacts(ctx, ids); err != nil {
			fail(err)
			continue
		}
		if r.networks != nil {
			for _, object := range snapshot.NetworkObjects {
				if cleanupErr := r.networks.DeleteOwned(ctx, object); cleanupErr != nil {
					fail(cleanupErr)
					cleanupFailed = true
				}
			}
		}
		if cleanupFailed {
			continue
		}
		if err = r.store.FinalizeLaboratoryDeletion(ctx, laboratory.ID); err != nil {
			return err
		}
	}
	return nil
}

func hotAddTapName(id domain.ID) string {
	value := "nlt" + string(id)
	if len(value) > 15 {
		value = value[:15]
	}
	return value
}
