package reconcile

import (
	"context"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

type dataPlaneStoreFake struct {
	lab               domain.Laboratory
	snapshot          domain.TopologySnapshot
	linkState         string
	interfaceStates   map[domain.ID]string
	deleted           bool
	attachmentState   string
	attachmentError   *domain.Problem
	objectLinkState   string
	objectLinkDeleted bool
}

func (s *dataPlaneStoreFake) ListLaboratories(context.Context) ([]domain.Laboratory, error) {
	return []domain.Laboratory{s.lab}, nil
}
func (s *dataPlaneStoreFake) Snapshot(context.Context, domain.ID) (domain.TopologySnapshot, error) {
	return s.snapshot, nil
}
func (s *dataPlaneStoreFake) ListNetworkAttachments(context.Context, domain.ID) ([]domain.NetworkAttachment, error) {
	return nil, nil
}
func (s *dataPlaneStoreFake) SetLinkObservedState(_ context.Context, _ domain.ID, state string) error {
	s.linkState = state
	return nil
}
func (s *dataPlaneStoreFake) SetInterfaceOperationalState(_ context.Context, id domain.ID, state string) error {
	s.interfaceStates[id] = state
	return nil
}
func (s *dataPlaneStoreFake) DeleteLink(context.Context, domain.ID) error {
	s.deleted = true
	return nil
}
func (s *dataPlaneStoreFake) SetNetworkAttachmentState(_ context.Context, _ domain.ID, state string, problem *domain.Problem) error {
	s.attachmentState, s.attachmentError = state, problem
	return nil
}
func (s *dataPlaneStoreFake) SetNetworkObjectLinkState(_ context.Context, _ domain.ID, state string, _ *domain.Problem) error {
	s.objectLinkState = state
	return nil
}
func (s *dataPlaneStoreFake) DeleteNetworkObjectLink(context.Context, domain.ID) error {
	s.objectLinkDeleted = true
	return nil
}

type dataPlaneRuntimeFake struct {
	ensureError           error
	ensureCalls           int
	deleted               bool
	objectLinkEnsureCalls int
	objectLinkDeleted     bool
}

func (r *dataPlaneRuntimeFake) EnsureLink(context.Context, domain.Link, domain.Interface, domain.Interface) error {
	r.ensureCalls++
	return r.ensureError
}
func (r *dataPlaneRuntimeFake) DeleteLink(context.Context, domain.ID) error {
	r.deleted = true
	return nil
}
func (r *dataPlaneRuntimeFake) Attach(context.Context, domain.Interface, domain.NetworkObject) error {
	return nil
}
func (r *dataPlaneRuntimeFake) EnsureNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error {
	r.objectLinkEnsureCalls++
	return nil
}
func (r *dataPlaneRuntimeFake) DeleteNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error {
	r.objectLinkDeleted = true
	return nil
}

func TestDataPlaneReconcilesOperationalStateAndDisconnect(t *testing.T) {
	store := &dataPlaneStoreFake{lab: domain.Laboratory{ID: "lab", LifecycleState: "active"}, interfaceStates: map[domain.ID]string{}}
	store.snapshot = domain.TopologySnapshot{Nodes: []domain.Node{{ID: "node-a", ObservedState: domain.ObservedRunning}, {ID: "node-b", ObservedState: domain.ObservedRunning}}, Interfaces: []domain.Interface{{ID: "a", NodeID: "node-a"}, {ID: "b", NodeID: "node-b"}}, Links: []domain.Link{{ID: "link", EndpointAID: "a", EndpointBID: "b", DesiredState: "connected"}}}
	runtime := &dataPlaneRuntimeFake{}
	if err := NewDataPlaneReconciler(store, runtime).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.linkState != "connected" || store.interfaceStates["a"] != "up" {
		t.Fatalf("state=%s interfaces=%v", store.linkState, store.interfaceStates)
	}
	store.snapshot.Links[0].DesiredState = "disconnected"
	if err := NewDataPlaneReconciler(store, runtime).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !runtime.deleted || !store.deleted {
		t.Fatal("disconnect did not clean runtime and database")
	}
}

func TestDataPlaneKeepsStoppedNodeLinksPending(t *testing.T) {
	store := &dataPlaneStoreFake{lab: domain.Laboratory{ID: "lab", LifecycleState: "active"}, interfaceStates: map[domain.ID]string{}}
	store.snapshot = domain.TopologySnapshot{Nodes: []domain.Node{{ID: "node-a", ObservedState: domain.ObservedStopped}, {ID: "node-b", ObservedState: domain.ObservedStopped}}, Interfaces: []domain.Interface{{ID: "a", NodeID: "node-a"}, {ID: "b", NodeID: "node-b"}}, Links: []domain.Link{{ID: "link", EndpointAID: "a", EndpointBID: "b", DesiredState: "connected"}}}
	runtime := &dataPlaneRuntimeFake{}
	if err := NewDataPlaneReconciler(store, runtime).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.linkState != "pending" || runtime.ensureCalls != 0 {
		t.Fatalf("state=%s ensure_calls=%d", store.linkState, runtime.ensureCalls)
	}
}

func TestDataPlaneReconcilesNetworkObjectLinksWhileObjectsRemainActive(t *testing.T) {
	store := &dataPlaneStoreFake{lab: domain.Laboratory{ID: "lab", LifecycleState: "active"}, interfaceStates: map[domain.ID]string{}}
	store.snapshot = domain.TopologySnapshot{
		NetworkObjects:     []domain.NetworkObject{{ID: "a", ObservedState: "active"}, {ID: "b", ObservedState: "active"}},
		NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "object-link", ObjectAID: "a", ObjectBID: "b", DesiredState: "connected"}},
	}
	runtime := &dataPlaneRuntimeFake{}
	if err := NewDataPlaneReconciler(store, runtime).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.objectLinkState != "connected" || runtime.objectLinkEnsureCalls != 1 {
		t.Fatalf("state=%s ensure_calls=%d", store.objectLinkState, runtime.objectLinkEnsureCalls)
	}
	store.snapshot.NetworkObjectLinks[0].DesiredState = "disconnected"
	if err := NewDataPlaneReconciler(store, runtime).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !runtime.objectLinkDeleted || !store.objectLinkDeleted {
		t.Fatal("network object link was not deleted live")
	}
}
