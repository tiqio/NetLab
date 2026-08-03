package reconcile

import (
	"context"

	"github.com/netlab/netlab/internal/domain"
)

type DataPlaneStore interface {
	ListLaboratories(context.Context) ([]domain.Laboratory, error)
	Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error)
	ListNetworkAttachments(context.Context, domain.ID) ([]domain.NetworkAttachment, error)
	SetLinkObservedState(context.Context, domain.ID, string) error
	SetInterfaceOperationalState(context.Context, domain.ID, string) error
	SetNetworkAttachmentState(context.Context, domain.ID, string, *domain.Problem) error
	SetNetworkObjectLinkState(context.Context, domain.ID, string, *domain.Problem) error
	DeleteLink(context.Context, domain.ID) error
	DeleteNetworkObjectLink(context.Context, domain.ID) error
}

type DataPlaneRuntime interface {
	EnsureLink(context.Context, domain.Link, domain.Interface, domain.Interface) error
	DeleteLink(context.Context, domain.ID) error
	Attach(context.Context, domain.Interface, domain.NetworkObject) error
	EnsureNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error
	DeleteNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error
}

type DataPlaneReconciler struct {
	store   DataPlaneStore
	runtime DataPlaneRuntime
}

func NewDataPlaneReconciler(store DataPlaneStore, runtime DataPlaneRuntime) *DataPlaneReconciler {
	return &DataPlaneReconciler{store: store, runtime: runtime}
}

func (r *DataPlaneReconciler) Name() string { return "data-plane" }

func (r *DataPlaneReconciler) Reconcile(ctx context.Context) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("reconciler", domain.ID(r.Name()), "data_plane_reconcile"))
	labs, err := r.store.ListLaboratories(ctx)
	if err != nil {
		return err
	}
	for _, laboratory := range labs {
		if laboratory.LifecycleState != "active" {
			continue
		}
		snapshot, snapshotErr := r.store.Snapshot(ctx, laboratory.ID)
		if snapshotErr != nil {
			return snapshotErr
		}
		interfaces := make(map[domain.ID]domain.Interface, len(snapshot.Interfaces))
		nodes := make(map[domain.ID]domain.Node, len(snapshot.Nodes))
		objects := make(map[domain.ID]domain.NetworkObject, len(snapshot.NetworkObjects))
		for _, node := range snapshot.Nodes {
			nodes[node.ID] = node
		}
		for _, iface := range snapshot.Interfaces {
			interfaces[iface.ID] = iface
		}
		for _, object := range snapshot.NetworkObjects {
			objects[object.ID] = object
		}
		for _, link := range snapshot.Links {
			if link.DesiredState == "disconnected" {
				_ = r.runtime.DeleteLink(ctx, link.ID)
				if err = r.store.DeleteLink(ctx, link.ID); err != nil {
					return err
				}
				continue
			}
			endpointA, endpointAExists := interfaces[link.EndpointAID]
			endpointB, endpointBExists := interfaces[link.EndpointBID]
			if !endpointAExists || !endpointBExists || nodes[endpointA.NodeID].ObservedState != domain.ObservedRunning || nodes[endpointB.NodeID].ObservedState != domain.ObservedRunning {
				_ = r.store.SetLinkObservedState(ctx, link.ID, "pending")
				continue
			}
			if err = r.runtime.EnsureLink(ctx, link, endpointA, endpointB); err != nil {
				_ = r.store.SetLinkObservedState(ctx, link.ID, "failed")
				continue
			}
			_ = r.store.SetLinkObservedState(ctx, link.ID, "connected")
			_ = r.store.SetInterfaceOperationalState(ctx, link.EndpointAID, "up")
			_ = r.store.SetInterfaceOperationalState(ctx, link.EndpointBID, "up")
		}
		attachments, attachmentErr := r.store.ListNetworkAttachments(ctx, laboratory.ID)
		if attachmentErr != nil {
			return attachmentErr
		}
		for _, attachment := range attachments {
			if iface, ok := interfaces[attachment.InterfaceID]; ok {
				object, exists := objects[attachment.NetworkObjectID]
				if !exists {
					problem := structuredProblem(nil, domain.Problem{Code: "attachment_target_missing", Message: "network object is unavailable", ResourceType: "network_attachment", ResourceID: attachment.ID, Phase: "attachment_reconcile", Cleanup: "attachment remains detached", OperatorHint: "restore the target network object or delete the attachment"})
					_ = r.store.SetNetworkAttachmentState(ctx, attachment.ID, "failed", problem)
					continue
				}
				var attachErr error
				if object.Kind == domain.NetworkBridge || object.Kind == domain.NetworkNAT {
					attachErr = r.runtime.Attach(ctx, iface, object)
				} else if runtime, supported := r.runtime.(interface {
					AttachNamespace(context.Context, domain.NetworkAttachment, domain.Interface, domain.NetworkObject) error
				}); supported {
					attachErr = runtime.AttachNamespace(ctx, attachment, iface, object)
				} else {
					attachErr = domain.Problem{Code: "capability_unsupported", Message: "namespace attachment runtime unavailable"}
				}
				if attachErr != nil {
					problem := structuredProblem(attachErr, domain.Problem{Code: "attachment_failed", Retryable: true, ResourceType: "network_attachment", ResourceID: attachment.ID, Phase: "attachment_reconcile", Cleanup: "attachment remains detached", OperatorHint: "inspect the interface and network object then retry", RetryAfterSeconds: 2})
					_ = r.store.SetNetworkAttachmentState(ctx, attachment.ID, "failed", problem)
					continue
				}
				_ = r.store.SetNetworkAttachmentState(ctx, attachment.ID, "active", nil)
			}
		}
		for _, link := range snapshot.NetworkObjectLinks {
			if link.ObservedState == "disconnecting" {
				continue
			}
			if link.DesiredState == "disconnected" {
				objectA, aExists := objects[link.ObjectAID]
				objectB, bExists := objects[link.ObjectBID]
				if aExists && bExists {
					_ = r.runtime.DeleteNetworkObjectLink(ctx, link, objectA, objectB)
				}
				if err = r.store.DeleteNetworkObjectLink(ctx, link.ID); err != nil {
					return err
				}
				continue
			}
			objectA, aExists := objects[link.ObjectAID]
			objectB, bExists := objects[link.ObjectBID]
			if !aExists || !bExists || objectA.ObservedState != "active" || objectB.ObservedState != "active" {
				_ = r.store.SetNetworkObjectLinkState(ctx, link.ID, "pending", nil)
				continue
			}
			if linkErr := r.runtime.EnsureNetworkObjectLink(ctx, link, objectA, objectB); linkErr != nil {
				problem := structuredProblem(linkErr, domain.Problem{Code: "network_object_link_failed", Retryable: true, ResourceType: "network_object_link", ResourceID: link.ID, Phase: "link_reconcile", Cleanup: "link remains disconnected", OperatorHint: "inspect both namespace ports then retry", RetryAfterSeconds: 2})
				_ = r.store.SetNetworkObjectLinkState(ctx, link.ID, "failed", problem)
				continue
			}
			_ = r.store.SetNetworkObjectLinkState(ctx, link.ID, "connected", nil)
		}
	}
	return nil
}

func (r *DataPlaneReconciler) ReconcileWithCheckpoints(ctx context.Context, checkpoint func(RecoveryResourceOutcome) error) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("reconciler", domain.ID(r.Name()), "data_plane_recovery"))
	if err := r.Reconcile(ctx); err != nil {
		return err
	}
	labs, err := r.store.ListLaboratories(ctx)
	if err != nil {
		return err
	}
	for _, laboratory := range labs {
		if laboratory.LifecycleState != "active" {
			continue
		}
		snapshot, snapshotErr := r.store.Snapshot(ctx, laboratory.ID)
		if snapshotErr != nil {
			return snapshotErr
		}
		for _, link := range snapshot.Links {
			state := "recovered"
			message := ""
			if link.ObservedState == "failed" {
				state, message = "failed", "link reconciliation failed"
			}
			if err = checkpoint(RecoveryResourceOutcome{ResourceType: "link", ResourceID: link.ID, State: state, Error: message}); err != nil {
				return err
			}
		}
		attachments, attachmentErr := r.store.ListNetworkAttachments(ctx, laboratory.ID)
		if attachmentErr != nil {
			return attachmentErr
		}
		for _, attachment := range attachments {
			state := "recovered"
			message := ""
			if attachment.ObservedState == "failed" {
				state, message = "failed", "attachment reconciliation failed"
			}
			if err = checkpoint(RecoveryResourceOutcome{ResourceType: "network_attachment", ResourceID: attachment.ID, State: state, Error: message}); err != nil {
				return err
			}
		}
		for _, link := range snapshot.NetworkObjectLinks {
			state := "recovered"
			message := ""
			if link.ObservedState == "disconnecting" {
				state, message = "pending_task_recovery", "durable delete task owns interrupted cleanup"
			} else if link.ObservedState == "failed" {
				state, message = "failed", "network object link reconciliation failed"
			}
			if err = checkpoint(RecoveryResourceOutcome{ResourceType: "network_object_link", ResourceID: link.ID, State: state, Error: message}); err != nil {
				return err
			}
		}
	}
	return nil
}
