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
	attachments       []domain.NetworkAttachment
	attachmentState   string
	attachmentError   *domain.Problem
	attachmentDeleted bool
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
	return s.attachments, nil
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
func (s *dataPlaneStoreFake) DeleteTopologyNetworkAttachment(context.Context, domain.ID, domain.Revision, domain.ID) error {
	s.attachmentDeleted = true
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
	attachmentError       error
	objectLinkError       error
	ensureCalls           int
	deleted               bool
	attachmentDeleted     bool
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
	return r.attachmentError
}
func (r *dataPlaneRuntimeFake) AttachNamespace(context.Context, domain.NetworkAttachment, domain.Interface, domain.NetworkObject) error {
	return r.attachmentError
}
func (r *dataPlaneRuntimeFake) DeleteAttachment(context.Context, domain.NetworkAttachment) error {
	r.attachmentDeleted = true
	return nil
}
func (r *dataPlaneRuntimeFake) EnsureNetworkObjectLink(context.Context, domain.NetworkObjectLink, domain.NetworkObject, domain.NetworkObject) error {
	r.objectLinkEnsureCalls++
	return r.objectLinkError
}

type dependentObjectReconcilerFake struct{ ids []domain.ID }

func (r *dependentObjectReconcilerFake) ReconcileObject(_ context.Context, id domain.ID, _ domain.Revision) (domain.NetworkObject, error) {
	r.ids = append(r.ids, id)
	return domain.NetworkObject{ID: id}, nil
}

func TestDataPlaneReconcilesL3AfterLateAttachmentAndObjectLink(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		attachments:     []domain.NetworkAttachment{{ID: "attachment", InterfaceID: "if-1", NetworkObjectID: "l3-a", PortName: "eth0", Revision: 1, ObservedState: "pending"}},
		snapshot: domain.TopologySnapshot{
			Interfaces: []domain.Interface{{ID: "if-1", NodeID: "node"}},
			NetworkObjects: []domain.NetworkObject{
				{ID: "l3-a", Kind: domain.NetworkSwitchL3, Revision: 2, ObservedState: "active"},
				{ID: "l3-b", Kind: domain.NetworkSwitchL3, Revision: 3, ObservedState: "active"},
			},
			NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "object-link", ObjectAID: "l3-a", PortAName: "eth1", ObjectBID: "l3-b", PortBName: "eth0", DesiredState: "connected", ObservedState: "pending"}},
		},
	}
	runtime := &dataPlaneRuntimeFake{}
	objects := &dependentObjectReconcilerFake{}
	reconciler := NewDataPlaneReconciler(store, runtime)
	reconciler.SetNetworkObjectReconciler(objects)
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []domain.ID{"l3-a", "l3-a", "l3-b"}
	if len(objects.ids) != len(want) {
		t.Fatalf("reconciled=%v", objects.ids)
	}
	for index := range want {
		if objects.ids[index] != want[index] {
			t.Fatalf("reconciled=%v want=%v", objects.ids, want)
		}
	}
}

func TestDataPlaneDoesNotReconfigureHealthyDependentObjects(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		attachments:     []domain.NetworkAttachment{{ID: "attachment", InterfaceID: "if-1", NetworkObjectID: "l3-a", PortName: "eth0", Revision: 1, ObservedState: "active"}},
		snapshot: domain.TopologySnapshot{
			Interfaces: []domain.Interface{{ID: "if-1", NodeID: "node"}},
			NetworkObjects: []domain.NetworkObject{
				{ID: "l3-a", Kind: domain.NetworkSwitchL3, Revision: 2, ObservedState: "active"},
				{ID: "l3-b", Kind: domain.NetworkSwitchL3, Revision: 3, ObservedState: "active"},
			},
			NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "object-link", ObjectAID: "l3-a", PortAName: "eth1", ObjectBID: "l3-b", PortBName: "eth0", DesiredState: "connected", ObservedState: "connected"}},
		},
	}
	runtime := &dataPlaneRuntimeFake{}
	objects := &dependentObjectReconcilerFake{}
	reconciler := NewDataPlaneReconciler(store, runtime)
	reconciler.SetNetworkObjectReconciler(objects)
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(objects.ids) != 0 {
		t.Fatalf("healthy dependent objects were reconfigured: %v", objects.ids)
	}
	if runtime.objectLinkEnsureCalls != 1 {
		t.Fatalf("healthy link was not inspected: %d", runtime.objectLinkEnsureCalls)
	}
}

func TestDataPlaneLeavesStoppedNodeAttachmentsPending(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		attachments:     []domain.NetworkAttachment{{ID: "attachment", InterfaceID: "if-1", NetworkObjectID: "l2", PortName: "access0", ObservedState: "active"}},
		snapshot: domain.TopologySnapshot{
			Nodes:          []domain.Node{{ID: "node", DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped}},
			Interfaces:     []domain.Interface{{ID: "if-1", NodeID: "node", OperationalState: "up"}},
			NetworkObjects: []domain.NetworkObject{{ID: "l2", Kind: domain.NetworkSwitchL2, ObservedState: "active"}},
		},
	}
	runtime := &dataPlaneRuntimeFake{}
	if err := NewDataPlaneReconciler(store, runtime).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.attachmentState != "pending" || store.attachmentError != nil {
		t.Fatalf("attachment state=%s problem=%+v", store.attachmentState, store.attachmentError)
	}
	if store.interfaceStates["if-1"] != "down" {
		t.Fatalf("interface state=%q", store.interfaceStates["if-1"])
	}
	if runtime.attachmentDeleted {
		t.Fatal("stopped node attachment runtime was deleted")
	}
}

func TestDataPlaneReconfiguresOnlyPendingEndpointOnHealthyLink(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		snapshot: domain.TopologySnapshot{
			NetworkObjects: []domain.NetworkObject{
				{ID: "l2", Kind: domain.NetworkSwitchL2, Revision: 2, ObservedState: "pending"},
				{ID: "l3", Kind: domain.NetworkSwitchL3, Revision: 3, ObservedState: "active"},
			},
			NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "object-link", ObjectAID: "l2", PortAName: "eth0", ObjectBID: "l3", PortBName: "eth0", DesiredState: "connected", ObservedState: "connected"}},
		},
	}
	objects := &dependentObjectReconcilerFake{}
	reconciler := NewDataPlaneReconciler(store, &dataPlaneRuntimeFake{})
	reconciler.SetNetworkObjectReconciler(objects)
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(objects.ids) != 1 || objects.ids[0] != "l2" {
		t.Fatalf("reconfigured endpoints=%v", objects.ids)
	}
}

func TestDataPlaneReconcilesL2AfterLateAttachment(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		attachments:     []domain.NetworkAttachment{{ID: "attachment", InterfaceID: "if-1", NetworkObjectID: "l2", PortName: "access0", Revision: 1}},
		snapshot: domain.TopologySnapshot{
			Interfaces:     []domain.Interface{{ID: "if-1", NodeID: "node"}},
			NetworkObjects: []domain.NetworkObject{{ID: "l2", Kind: domain.NetworkSwitchL2, Revision: 2, ObservedState: "active"}},
		},
	}
	objects := &dependentObjectReconcilerFake{}
	reconciler := NewDataPlaneReconciler(store, &dataPlaneRuntimeFake{})
	reconciler.SetNetworkObjectReconciler(objects)
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(objects.ids) != 1 || objects.ids[0] != "l2" {
		t.Fatalf("late L2 port did not trigger membership reconcile: %v", objects.ids)
	}
}

func TestDataPlaneConnectsPendingL2ObjectSoLatePortsCanConverge(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		snapshot: domain.TopologySnapshot{
			NetworkObjects: []domain.NetworkObject{
				{ID: "l2", Kind: domain.NetworkSwitchL2, Revision: 2, ObservedState: "pending"},
				{ID: "pc", Kind: domain.NetworkPC, Revision: 1, ObservedState: "active"},
			},
			NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "late-port", ObjectAID: "l2", PortAName: "access0", ObjectBID: "pc", PortBName: "eth0", DesiredState: "connected"}},
		},
	}
	runtime := &dataPlaneRuntimeFake{}
	objects := &dependentObjectReconcilerFake{}
	reconciler := NewDataPlaneReconciler(store, runtime)
	reconciler.SetNetworkObjectReconciler(objects)
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.objectLinkEnsureCalls != 1 {
		t.Fatalf("pending L2 link ensure calls=%d", runtime.objectLinkEnsureCalls)
	}
	if len(objects.ids) != 1 || objects.ids[0] != "l2" {
		t.Fatalf("pending L2 was not reconciled after port arrival: %v", objects.ids)
	}
}

func TestDataPlaneConnectsPendingL3ObjectSoLatePortsCanConverge(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		snapshot: domain.TopologySnapshot{
			NetworkObjects: []domain.NetworkObject{
				{ID: "l3", Kind: domain.NetworkSwitchL3, Revision: 2, ObservedState: "pending"},
				{ID: "pc", Kind: domain.NetworkPC, Revision: 1, ObservedState: "active"},
			},
			NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "late-port", ObjectAID: "l3", PortAName: "eth0", ObjectBID: "pc", PortBName: "eth0", DesiredState: "connected"}},
		},
	}
	runtime := &dataPlaneRuntimeFake{}
	objects := &dependentObjectReconcilerFake{}
	reconciler := NewDataPlaneReconciler(store, runtime)
	reconciler.SetNetworkObjectReconciler(objects)
	if err := reconciler.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.objectLinkEnsureCalls != 1 {
		t.Fatalf("pending L3 link ensure calls=%d", runtime.objectLinkEnsureCalls)
	}
	if len(objects.ids) != 1 || objects.ids[0] != "l3" {
		t.Fatalf("pending L3 was not reconciled after port arrival: %v", objects.ids)
	}
}

func TestDataPlaneDoesNotConnectFailedL2Object(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		snapshot: domain.TopologySnapshot{
			NetworkObjects: []domain.NetworkObject{
				{ID: "l2", Kind: domain.NetworkSwitchL2, ObservedState: "failed"},
				{ID: "pc", Kind: domain.NetworkPC, ObservedState: "active"},
			},
			NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "blocked", ObjectAID: "l2", ObjectBID: "pc", DesiredState: "connected"}},
		},
	}
	runtime := &dataPlaneRuntimeFake{}
	if err := NewDataPlaneReconciler(store, runtime).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.objectLinkEnsureCalls != 0 {
		t.Fatalf("failed L2 unexpectedly connected: %d", runtime.objectLinkEnsureCalls)
	}
}

func TestDataPlaneCompensatesPartialConnectionCreationFailures(t *testing.T) {
	tests := []struct {
		name    string
		store   *dataPlaneStoreFake
		runtime *dataPlaneRuntimeFake
		assert  func(*testing.T, *dataPlaneStoreFake, *dataPlaneRuntimeFake)
	}{
		{
			name: "link",
			store: &dataPlaneStoreFake{
				lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
				interfaceStates: map[domain.ID]string{},
				snapshot: domain.TopologySnapshot{
					Nodes:      []domain.Node{{ID: "node-a", ObservedState: domain.ObservedRunning}, {ID: "node-b", ObservedState: domain.ObservedRunning}},
					Interfaces: []domain.Interface{{ID: "a", NodeID: "node-a"}, {ID: "b", NodeID: "node-b"}},
					Links:      []domain.Link{{ID: "link", EndpointAID: "a", EndpointBID: "b", DesiredState: "connected"}},
				},
			},
			runtime: &dataPlaneRuntimeFake{ensureError: domain.ErrNotFound},
			assert: func(t *testing.T, store *dataPlaneStoreFake, runtime *dataPlaneRuntimeFake) {
				if runtime.deleted || store.deleted {
					t.Fatal("transiently unavailable link was deleted")
				}
				if store.linkState != "pending" {
					t.Fatalf("link state=%s", store.linkState)
				}
			},
		},
		{
			name: "network attachment",
			store: &dataPlaneStoreFake{
				lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
				interfaceStates: map[domain.ID]string{},
				attachments:     []domain.NetworkAttachment{{ID: "attachment", InterfaceID: "interface", NetworkObjectID: "object", Revision: 1}},
				snapshot: domain.TopologySnapshot{
					Interfaces:     []domain.Interface{{ID: "interface", NodeID: "node"}},
					NetworkObjects: []domain.NetworkObject{{ID: "object", Kind: domain.NetworkBridge}},
				},
			},
			runtime: &dataPlaneRuntimeFake{attachmentError: domain.ErrNotFound},
			assert: func(t *testing.T, store *dataPlaneStoreFake, runtime *dataPlaneRuntimeFake) {
				if runtime.attachmentDeleted || store.attachmentDeleted {
					t.Fatalf("runtime_deleted=%t authoritative_deleted=%t", runtime.attachmentDeleted, store.attachmentDeleted)
				}
				if store.attachmentState != "pending" || store.attachmentError == nil || !store.attachmentError.Retryable || store.attachmentError.Code != "attachment_runtime_pending" {
					t.Fatalf("attachment state=%s problem=%+v", store.attachmentState, store.attachmentError)
				}
			},
		},
		{
			name: "network object link",
			store: &dataPlaneStoreFake{
				lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
				interfaceStates: map[domain.ID]string{},
				snapshot: domain.TopologySnapshot{
					NetworkObjects:     []domain.NetworkObject{{ID: "a", ObservedState: "active"}, {ID: "b", ObservedState: "active"}},
					NetworkObjectLinks: []domain.NetworkObjectLink{{ID: "object-link", ObjectAID: "a", ObjectBID: "b", DesiredState: "connected"}},
				},
			},
			runtime: &dataPlaneRuntimeFake{objectLinkError: domain.ErrNotFound},
			assert: func(t *testing.T, store *dataPlaneStoreFake, runtime *dataPlaneRuntimeFake) {
				if !runtime.objectLinkDeleted || !store.objectLinkDeleted {
					t.Fatal("object-link partial resources were not compensated")
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := NewDataPlaneReconciler(test.store, test.runtime).Reconcile(context.Background()); err != nil {
				t.Fatal(err)
			}
			test.assert(t, test.store, test.runtime)
		})
	}
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

func TestDataPlaneMarksAttachedInterfaceUp(t *testing.T) {
	store := &dataPlaneStoreFake{
		lab:             domain.Laboratory{ID: "lab", LifecycleState: "active"},
		interfaceStates: map[domain.ID]string{},
		attachments:     []domain.NetworkAttachment{{ID: "attachment", InterfaceID: "interface", NetworkObjectID: "nat"}},
	}
	store.snapshot = domain.TopologySnapshot{
		Interfaces:     []domain.Interface{{ID: "interface", NodeID: "node"}},
		NetworkObjects: []domain.NetworkObject{{ID: "nat", Kind: domain.NetworkNAT}},
	}
	if err := NewDataPlaneReconciler(store, &dataPlaneRuntimeFake{}).Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.attachmentState != "active" || store.interfaceStates["interface"] != "up" {
		t.Fatalf("attachment=%s interfaces=%v", store.attachmentState, store.interfaceStates)
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

func TestDataPlaneDoesNotResurrectDisconnectingObjectLinkDuringRecovery(t *testing.T) {
	store := &dataPlaneStoreFake{lab: domain.Laboratory{ID: "lab", LifecycleState: "active"}, interfaceStates: map[domain.ID]string{}}
	store.snapshot = domain.TopologySnapshot{
		NetworkObjects: []domain.NetworkObject{{ID: "a", ObservedState: "active"}, {ID: "b", ObservedState: "active"}},
		NetworkObjectLinks: []domain.NetworkObjectLink{{
			ID: "object-link", ObjectAID: "a", ObjectBID: "b", DesiredState: "connected", ObservedState: "disconnecting",
		}},
	}
	runtime := &dataPlaneRuntimeFake{}
	outcomes := []RecoveryResourceOutcome{}
	if err := NewDataPlaneReconciler(store, runtime).ReconcileWithCheckpoints(context.Background(), func(outcome RecoveryResourceOutcome) error {
		outcomes = append(outcomes, outcome)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if runtime.objectLinkEnsureCalls != 0 || runtime.objectLinkDeleted || store.objectLinkDeleted {
		t.Fatalf("interrupted delete was mutated by generic recovery: runtime=%+v store=%+v", runtime, store)
	}
	if len(outcomes) != 1 || outcomes[0].State != "pending_task_recovery" {
		t.Fatalf("outcomes=%+v", outcomes)
	}
}
