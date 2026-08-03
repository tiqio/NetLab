package command

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/netlab/netlab/internal/app/ports"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
)

type topologyTaskStore struct {
	mu    sync.Mutex
	tasks map[domain.ID]domain.OperationTask
}

func newTopologyTaskStore() *topologyTaskStore {
	return &topologyTaskStore{tasks: map[domain.ID]domain.OperationTask{}}
}

func (s *topologyTaskStore) CreateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[value.ID]; exists {
		return errors.New("duplicate task")
	}
	s.tasks[value.ID] = value
	return nil
}

func (s *topologyTaskStore) UpdateTask(_ context.Context, value domain.OperationTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[value.ID] = value
	return nil
}

func (s *topologyTaskStore) GetTask(_ context.Context, id domain.ID) (domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.tasks[id]
	if !exists {
		return value, errors.New("task not found")
	}
	return value, nil
}

func (s *topologyTaskStore) ListRecoverableTasks(_ context.Context, limit int) ([]domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := make([]domain.OperationTask, 0, len(s.tasks))
	for _, value := range s.tasks {
		if value.State == domain.TaskQueued || value.State == domain.TaskRunning || value.State == domain.TaskCancelling {
			values = append(values, value)
			if len(values) == limit {
				break
			}
		}
	}
	return values, nil
}

func (s *topologyTaskStore) GetTaskByIdempotency(_ context.Context, kind, key string) (domain.OperationTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, value := range s.tasks {
		if value.Kind == kind && value.IdempotencyKey == key {
			return value, nil
		}
	}
	return domain.OperationTask{}, errors.New("task not found")
}

type topologyTaskRepositoryFake struct {
	mu                 sync.Mutex
	node               domain.Node
	links              map[domain.ID]domain.Link
	linkEndpointsReady bool
}

func (r *topologyTaskRepositoryFake) SetNodeObservedState(_ context.Context, id domain.ID, state domain.ObservedState, problem *domain.Problem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.node.ID != id {
		return errors.New("node not found")
	}
	if err := domain.ValidateNodeTransition(r.node.ObservedState, state); err != nil {
		return err
	}
	r.node.ObservedState = state
	r.node.LastError = problem
	return nil
}

func newTopologyTaskRepositoryFake() *topologyTaskRepositoryFake {
	return &topologyTaskRepositoryFake{node: domain.Node{ID: "node", Revision: 1, DesiredState: domain.DesiredStopped, ObservedState: domain.ObservedStopped}, links: map[domain.ID]domain.Link{}, linkEndpointsReady: true}
}

func (r *topologyTaskRepositoryFake) GetNode(_ context.Context, id domain.ID) (domain.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.node.ID != id {
		return domain.Node{}, errors.New("node not found")
	}
	return r.node, nil
}

func (r *topologyTaskRepositoryFake) ListNodeLinks(_ context.Context, _ domain.ID) ([]domain.Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make([]domain.Link, 0, len(r.links))
	for _, link := range r.links {
		values = append(values, link)
	}
	return values, nil
}

func (r *topologyTaskRepositoryFake) SetNodeDesiredState(_ context.Context, id domain.ID, revision domain.Revision, state domain.DesiredState) (domain.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.node.ID != id {
		return domain.Node{}, errors.New("node not found")
	}
	if r.node.Revision != revision {
		return domain.Node{}, errors.New("revision conflict")
	}
	r.node.DesiredState = state
	r.node.Revision++
	return r.node, nil
}

func (r *topologyTaskRepositoryFake) DeleteNode(_ context.Context, id domain.ID, revision domain.Revision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.node.ID != id {
		return errors.New("node not found")
	}
	if r.node.Revision != revision {
		return errors.New("revision conflict")
	}
	r.node = domain.Node{}
	return nil
}

func (r *topologyTaskRepositoryFake) CreateLink(_ context.Context, value domain.Link) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.links[value.ID]; exists {
		return errors.New("duplicate link")
	}
	r.links[value.ID] = value
	return nil
}

func (r *topologyTaskRepositoryFake) GetLink(_ context.Context, id domain.ID) (domain.Link, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, exists := r.links[id]
	if !exists {
		return value, errors.New("link not found")
	}
	return value, nil
}

func (r *topologyTaskRepositoryFake) LinkEndpointsReady(context.Context, domain.Link) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.linkEndpointsReady, nil
}

func (r *topologyTaskRepositoryFake) MarkLinkDisconnected(_ context.Context, id domain.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, exists := r.links[id]
	if !exists {
		return errors.New("link not found")
	}
	value.DesiredState = "disconnected"
	value.ObservedState = "disconnecting"
	r.links[id] = value
	return nil
}

func (r *topologyTaskRepositoryFake) MarkLinkConnected(_ context.Context, id domain.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, exists := r.links[id]
	if !exists {
		return errors.New("link not found")
	}
	value.DesiredState = "connected"
	value.ObservedState = "pending"
	r.links[id] = value
	return nil
}

func (r *topologyTaskRepositoryFake) DeleteLink(_ context.Context, id domain.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.links, id)
	return nil
}

func (r *topologyTaskRepositoryFake) setNodeObserved(state domain.ObservedState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.node.ObservedState = state
}

type topologyDeleteRuntime struct {
	deleteErr error
	deleted   int
}

type topologyDeleteLinkRuntime struct {
	deleted []domain.ID
}

func (r *topologyDeleteLinkRuntime) DeleteLink(_ context.Context, id domain.ID) error {
	r.deleted = append(r.deleted, id)
	return nil
}

func (r *topologyDeleteRuntime) Inspect(context.Context, domain.Node) (ports.ActualNode, error) {
	return ports.ActualNode{State: domain.ObservedStopped}, nil
}
func (r *topologyDeleteRuntime) Start(context.Context, domain.Node) error { return nil }
func (r *topologyDeleteRuntime) Stop(context.Context, domain.Node) error  { return nil }
func (r *topologyDeleteRuntime) Delete(context.Context, domain.Node) error {
	r.deleted++
	return r.deleteErr
}

func (r *topologyTaskRepositoryFake) setLinkObserved(id domain.ID, state string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value := r.links[id]
	value.ObservedState = state
	r.links[id] = value
}

func TestTopologyTaskNodeLifecyclePersistsProgress(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.poll = 5 * time.Millisecond
	service.timeout = time.Second
	value, err := service.SetNodeState(context.Background(), "node", 1, domain.DesiredRunning, "state-key")
	if err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool {
		return current.State == domain.TaskRunning && current.ProgressCurrent >= 1
	})
	repository.setNodeObserved(domain.ObservedRunning)
	current := waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
	if current.ProgressCurrent != current.ProgressTotal || current.IdempotencyKey != "state-key" {
		t.Fatalf("task=%+v", current)
	}
}

func TestTopologyTaskNodeDeleteFailureRetainsNodeAndDiagnostics(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	repository.node.Kind = "qemu"
	runtime := &topologyDeleteRuntime{deleteErr: errors.New("owned runtime directory busy")}
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.SetNodeDeletionRuntime("qemu", runtime)
	value, err := service.DeleteNode(context.Background(), "node", 1, "delete-failure")
	if err != nil {
		t.Fatal(err)
	}
	current := waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskFailed })
	if current.Error == nil || current.Error.Code != "node_delete_runtime_failed" || current.Error.Phase != "deleting" || current.Error.Cleanup == "" || current.Error.OperatorHint == "" {
		t.Fatalf("task=%+v", current)
	}
	node, getErr := repository.GetNode(context.Background(), "node")
	if getErr != nil || node.ObservedState != domain.ObservedFailed || node.LastError == nil || runtime.deleted != 1 {
		t.Fatalf("node=%+v get_err=%v runtime=%+v", node, getErr, runtime)
	}
}

func TestTopologyTaskNodeDeleteCleansRuntimeBeforeRow(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	repository.node.Kind = "qemu"
	runtime := &topologyDeleteRuntime{}
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.SetNodeDeletionRuntime("qemu", runtime)
	value, err := service.DeleteNode(context.Background(), "node", 1, "delete-success")
	if err != nil {
		t.Fatal(err)
	}
	current := waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
	if current.ProgressCurrent != 3 || runtime.deleted != 1 {
		t.Fatalf("task=%+v runtime=%+v", current, runtime)
	}
	if _, getErr := repository.GetNode(context.Background(), "node"); getErr == nil {
		t.Fatal("node row remained after successful runtime cleanup")
	}
}

func TestTopologyTaskNodeDeleteStopsRunningNodeAndCleansConnectedLinks(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	repository.node.Kind = "qemu"
	repository.node.DesiredState = domain.DesiredRunning
	repository.node.ObservedState = domain.ObservedRunning
	repository.links["link"] = domain.Link{ID: "link", Revision: 1, DesiredState: "connected", ObservedState: "connected"}
	nodeRuntime := &topologyDeleteRuntime{}
	linkRuntime := &topologyDeleteLinkRuntime{}
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.SetNodeDeletionRuntime("qemu", nodeRuntime)
	service.SetNodeDeletionLinkRuntime(linkRuntime)
	value, err := service.DeleteNode(context.Background(), "node", 1, "delete-running")
	if err != nil {
		t.Fatal(err)
	}
	current := waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
	if current.ProgressCurrent != current.ProgressTotal || nodeRuntime.deleted != 1 || len(linkRuntime.deleted) != 1 || linkRuntime.deleted[0] != "link" {
		t.Fatalf("task=%+v node_runtime=%+v link_runtime=%+v", current, nodeRuntime, linkRuntime)
	}
	if _, getErr := repository.GetNode(context.Background(), "node"); getErr == nil {
		t.Fatal("running node remained after successful deletion")
	}
}

func TestTopologyTaskCancellationCompensatesNodeDesiredState(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.poll = 5 * time.Millisecond
	service.timeout = time.Second
	value, err := service.SetNodeState(context.Background(), "node", 1, domain.DesiredRunning, "cancel-key")
	if err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool {
		return current.State == domain.TaskRunning && current.ProgressCurrent >= 1
	})
	if err = runner.Cancel(context.Background(), value.ID); err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskCancelled })
	node, err := repository.GetNode(context.Background(), "node")
	if err != nil || node.DesiredState != domain.DesiredStopped {
		t.Fatalf("node=%+v err=%v", node, err)
	}
}

func TestTopologyTaskRecoveryResumesOriginalTask(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	repository.node.DesiredState = domain.DesiredRunning
	repository.node.ObservedState = domain.ObservedRunning
	value := domain.OperationTask{ID: "recover-task", Kind: "node.set_state", ResourceType: "node", ResourceID: "node", State: domain.TaskRunning, ProgressTotal: 3, Input: map[string]any{"revision": int64(1), "desired_state": "running", "previous_desired_state": "stopped"}, CreatedAt: time.Now().UTC()}
	store.tasks[value.ID] = value
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.poll = 5 * time.Millisecond
	service.timeout = time.Second
	if err := runner.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	current := waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
	if current.ID != value.ID || current.Kind != value.Kind {
		t.Fatalf("task identity changed: %+v", current)
	}
}

func TestTopologyTaskIdempotencyReplaysAndRejectsConflicts(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.poll = 5 * time.Millisecond
	service.timeout = time.Second
	first, err := service.SetNodeState(context.Background(), "node", 1, domain.DesiredRunning, "same-key")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.SetNodeState(context.Background(), "node", 1, domain.DesiredRunning, "same-key")
	if err != nil || second.ID != first.ID {
		t.Fatalf("first=%s second=%s err=%v", first.ID, second.ID, err)
	}
	if _, err = service.SetNodeState(context.Background(), "node", 1, domain.DesiredStopped, "same-key"); err == nil {
		t.Fatal("expected idempotency conflict")
	} else if problem, ok := domain.ProblemFromError(err); !ok || problem.Code != "idempotency_conflict" || problem.TaskID != first.ID {
		t.Fatalf("problem=%+v ok=%t", problem, ok)
	}
	if err = runner.Cancel(context.Background(), first.ID); err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, first.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskCancelled })
}

func TestTopologyTaskLinkConnectAndCancellationCompensation(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)
	service.poll = 5 * time.Millisecond
	service.timeout = time.Second
	link, value, err := service.ConnectLink(context.Background(), "lab", "a", "b", "link-key")
	if err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool {
		return current.State == domain.TaskRunning && current.ProgressCurrent >= 1
	})
	if err = runner.Cancel(context.Background(), value.ID); err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskCancelled })
	current, err := repository.GetLink(context.Background(), link.ID)
	if err != nil || current.DesiredState != "disconnected" {
		t.Fatalf("link=%+v err=%v", current, err)
	}

	link, value, err = service.ConnectLink(context.Background(), "lab", "c", "d", "link-success")
	if err != nil {
		t.Fatal(err)
	}
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool {
		return current.State == domain.TaskRunning && current.ProgressCurrent >= 1
	})
	repository.setLinkObserved(link.ID, "connected")
	waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool { return current.State == domain.TaskSucceeded })
}

func TestTopologyTaskLinkConnectSucceedsPendingWhenEndpointsAreStopped(t *testing.T) {
	store := newTopologyTaskStore()
	repository := newTopologyTaskRepositoryFake()
	repository.linkEndpointsReady = false
	runner := task.NewRunner(store, 1, 8)
	defer runner.Close()
	service := NewTopologyTaskService(repository, runner)

	link, value, err := service.ConnectLink(context.Background(), "lab", "a", "b", "link-pending")
	if err != nil {
		t.Fatal(err)
	}
	completed := waitForTopologyTask(t, store, value.ID, func(current domain.OperationTask) bool {
		return current.State == domain.TaskSucceeded
	})
	if completed.Result["convergence"] != "pending" {
		t.Fatalf("result=%v", completed.Result)
	}
	current, err := repository.GetLink(context.Background(), link.ID)
	if err != nil || current.ObservedState != "pending" || current.DesiredState != "connected" {
		t.Fatalf("link=%+v err=%v", current, err)
	}
}

func waitForTopologyTask(t *testing.T, store *topologyTaskStore, id domain.ID, condition func(domain.OperationTask) bool) domain.OperationTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := store.GetTask(context.Background(), id)
		if err == nil && condition(value) {
			return value
		}
		time.Sleep(5 * time.Millisecond)
	}
	value, _ := store.GetTask(context.Background(), id)
	t.Fatalf("task did not reach expected state: %+v", value)
	return domain.OperationTask{}
}
