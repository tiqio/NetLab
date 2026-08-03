package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

type CaptureRequest struct {
	LaboratoryID domain.ID
	SourceType   string
	SourceID     domain.ID
	Purpose      string
	ParentID     domain.ID
	Interface    string
	Namespace    string
	Filter       string
	Format       string
	Retain       bool
	MaxBytes     int64
	Duration     time.Duration
	Direction    string
}

type managedCapture struct {
	metadata   domain.Capture
	worker     *captureRuntime.Worker
	path       string
	request    CaptureRequest
	stopReason string
}

type captureRecord struct {
	Metadata domain.Capture `json:"metadata"`
	Path     string         `json:"path,omitempty"`
	Request  CaptureRequest `json:"request"`
}

type CaptureArtifactService interface {
	Create(context.Context, string, string, string, domain.ID, []byte, time.Duration) (domain.Artifact, error)
}

type CaptureObserver func(domain.ID, domain.ID, domain.ID, domain.ID, domain.ID, domain.ID, string, string, []byte, time.Time)

type CaptureManager struct {
	directory      string
	concurrent     int
	globalMaxBytes int64
	retention      time.Duration
	mu             sync.RWMutex
	values         map[domain.ID]*managedCapture
	artifacts      CaptureArtifactService
	observer       CaptureObserver
	networkObjects interface {
		GetNetworkObjectLink(context.Context, domain.ID) (domain.NetworkObjectLink, error)
		GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error)
	}
}

func NewCaptureManager(stateDir string, concurrent int, globalMaxBytes int64, retention time.Duration, artifacts ...CaptureArtifactService) *CaptureManager {
	if concurrent < 1 {
		concurrent = 4
	}
	if globalMaxBytes < 1 {
		globalMaxBytes = 10 << 30
	}
	if retention <= 0 {
		retention = 24 * time.Hour
	}
	manager := &CaptureManager{directory: filepath.Join(stateDir, "captures"), concurrent: concurrent, globalMaxBytes: globalMaxBytes, retention: retention, values: map[domain.ID]*managedCapture{}}
	if len(artifacts) > 0 {
		manager.artifacts = artifacts[0]
	}
	manager.load()
	return manager
}

func (m *CaptureManager) SetObserver(observer CaptureObserver) { m.observer = observer }

func (m *CaptureManager) SetNetworkObjectRepository(repository interface {
	GetNetworkObjectLink(context.Context, domain.ID) (domain.NetworkObjectLink, error)
	GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error)
}) {
	m.networkObjects = repository
}

func (m *CaptureManager) AvailableBytes() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	available := m.globalMaxBytes - m.reservedBytesLocked()
	if available < 0 {
		return 0
	}
	return available
}

func (m *CaptureManager) reservedBytesLocked() int64 {
	var reserved int64
	for _, value := range m.values {
		switch value.metadata.State {
		case "starting", "running", "stopping":
			budget := value.metadata.MaxBytes
			if budget < value.metadata.BytesWritten {
				budget = value.metadata.BytesWritten
			}
			reserved += budget
		default:
			if value.metadata.Retain {
				reserved += value.metadata.BytesWritten
			}
		}
	}
	return reserved
}

func (m *CaptureManager) Start(ctx context.Context, request CaptureRequest) (domain.Capture, error) {
	return m.StartAs(ctx, domain.NewID(), request)
}

func (m *CaptureManager) StartAs(ctx context.Context, id domain.ID, request CaptureRequest) (domain.Capture, error) {
	var err error
	if request.Interface, request.Namespace, err = m.captureLocator(ctx, request.SourceType, request.SourceID); err != nil {
		return domain.Capture{}, *structuredProblem(err, domain.Problem{Code: "invalid_capture_source", ResourceType: request.SourceType, ResourceID: request.SourceID, Phase: "capture_admission", Cleanup: "no capture resources created", OperatorHint: "select an existing interface or link"})
	}
	m.mu.Lock()
	if existing := m.values[id]; existing != nil {
		if existing.metadata.State == "starting" || existing.metadata.State == "running" || existing.metadata.State == "stopping" {
			metadata := existing.metadata
			m.mu.Unlock()
			return metadata, nil
		}
		delete(m.values, id)
	}
	active := 0
	for _, value := range m.values {
		if value.metadata.State == "running" || value.metadata.State == "starting" {
			active++
		}
	}
	if active >= m.concurrent {
		m.mu.Unlock()
		return domain.Capture{}, domain.Problem{Code: "resource_exhausted", Message: "capture concurrency limit reached", Retryable: true, ResourceType: "capture", ResourceID: id, Phase: "capture_admission", Cleanup: "no capture resources created", OperatorHint: "stop an active capture and retry", RetryAfterSeconds: 2}
	}
	if request.MaxBytes <= 0 {
		request.MaxBytes = 256 << 20
	}
	if request.Format == "" {
		request.Format = "pcap"
	}
	if m.reservedBytesLocked()+request.MaxBytes > m.globalMaxBytes {
		m.mu.Unlock()
		return domain.Capture{}, domain.Problem{Code: "resource_exhausted", Message: "capture global byte ceiling reached", Retryable: true, ResourceType: "capture", ResourceID: id, Phase: "capture_admission", Cleanup: "no capture resources created", OperatorHint: "delete retained captures or reduce max_bytes", RetryAfterSeconds: 5}
	}
	if err := os.MkdirAll(m.directory, 0o700); err != nil {
		m.mu.Unlock()
		return domain.Capture{}, *structuredProblem(err, domain.Problem{Code: "capture_prepare_failed", Retryable: true, ResourceType: "capture", ResourceID: id, Phase: "capture_prepare", Cleanup: "no capture worker started", OperatorHint: "inspect capture directory permissions and retry", RetryAfterSeconds: 2})
	}
	path := ""
	if request.Retain {
		path = filepath.Join(m.directory, string(id)+"."+request.Format)
	}
	worker, err := captureRuntime.NewWorker(captureRuntime.WorkerConfig{OwnershipID: id, Interface: request.Interface, Namespace: request.Namespace, Filter: request.Filter, Format: request.Format, MaxBytes: request.MaxBytes, Duration: request.Duration, RetainPath: path, Direction: request.Direction})
	if err != nil {
		m.mu.Unlock()
		return domain.Capture{}, *structuredProblem(err, domain.Problem{Code: "capture_prepare_failed", Retryable: true, ResourceType: "capture", ResourceID: id, Phase: "capture_prepare", Cleanup: "capture worker was not started", OperatorHint: "verify tcpdump and the selected source interface", RetryAfterSeconds: 2})
	}
	now := time.Now().UTC()
	metadata := domain.Capture{ID: id, LaboratoryID: request.LaboratoryID, SourceType: request.SourceType, SourceID: request.SourceID, Purpose: request.Purpose, ParentResourceID: request.ParentID, Filter: request.Filter, Format: request.Format, State: "starting", Retain: request.Retain, MaxBytes: request.MaxBytes, CreatedAt: now, StartedAt: &now}
	managed := &managedCapture{metadata: metadata, worker: worker, path: path, request: request}
	m.values[id] = managed
	m.persistLocked()
	m.mu.Unlock()
	if err = worker.Start(ctx); err != nil {
		m.mu.Lock()
		delete(m.values, id)
		m.persistLocked()
		m.mu.Unlock()
		return domain.Capture{}, err
	}
	m.mu.Lock()
	managed.metadata.State = "running"
	metadata = managed.metadata
	m.persistLocked()
	m.mu.Unlock()
	go m.watch(id, managed)
	if m.observer != nil {
		stream, cancel := worker.Subscribe()
		go func() {
			defer cancel()
			for chunk := range stream {
				interfaceID, linkID, objectLinkID := request.SourceID, domain.ID(""), domain.ID("")
				if request.SourceType == "link" {
					interfaceID, linkID = "", request.SourceID
				} else if request.SourceType == "network_object_link" {
					interfaceID, objectLinkID = "", request.SourceID
				}
				direction := request.Direction
				if direction == "" {
					direction = "observed"
				}
				m.observer(id, request.ParentID, request.LaboratoryID, interfaceID, linkID, objectLinkID, direction, request.Format, chunk, time.Now().UTC())
			}
		}()
	}
	return metadata, nil
}

func (m *CaptureManager) captureLocator(ctx context.Context, sourceType string, sourceID domain.ID) (string, string, error) {
	if sourceID == "" {
		return "", "", domain.Problem{Code: "invalid_capture_source", Message: "source_id is required", Phase: "capture_admission", Cleanup: "no capture resources created", OperatorHint: "provide an interface or link source_id"}
	}
	switch sourceType {
	case "interface":
		value, err := captureInterface(sourceType, sourceID)
		return value, "", err
	case "link":
		value, err := captureInterface(sourceType, sourceID)
		return value, "", err
	case "network_object_link":
		if m.networkObjects == nil {
			return "", "", domain.Problem{Code: "capture_source_unavailable", Message: "network object link resolver is unavailable"}
		}
		link, err := m.networkObjects.GetNetworkObjectLink(ctx, sourceID)
		if err != nil {
			return "", "", err
		}
		object, err := m.networkObjects.GetNetworkObject(ctx, link.ObjectAID)
		if err != nil {
			return "", "", err
		}
		namespace, err := linuxnet.NetworkObjectNamespaceName(object)
		if err != nil {
			return "", "", err
		}
		return link.PortAName, namespace, nil
	default:
		return "", "", domain.Problem{Code: "invalid_capture_source", Message: "source_type must be interface, link, or network_object_link", ResourceType: sourceType, ResourceID: sourceID, Phase: "capture_admission", Cleanup: "no capture resources created"}
	}
}

func captureInterface(sourceType string, sourceID domain.ID) (string, error) {
	if sourceID == "" {
		return "", domain.Problem{Code: "invalid_capture_source", Message: "source_id is required"}
	}
	switch sourceType {
	case "interface":
		return linuxnet.HostInterfaceName(sourceID), nil
	case "link":
		return linuxnet.LinkBridgeName(sourceID), nil
	default:
		return "", domain.Problem{Code: "invalid_capture_source", Message: "source_type must be interface or link"}
	}
}

func (m *CaptureManager) watch(id domain.ID, managed *managedCapture) {
	<-managed.worker.Done()
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.values[id]
	if current == nil {
		return
	}
	current.metadata.BytesWritten = managed.worker.Bytes()
	current.metadata.Packets = managed.worker.Packets()
	current.metadata.Truncated = managed.worker.Truncated()
	current.metadata.FinishedAt = &now
	if err := managed.worker.Error(); err != nil {
		current.metadata.State = "failed"
		current.metadata.CompletionReason = "failed"
		current.metadata.LastError = structuredProblem(err, domain.Problem{Code: "capture_failed", Retryable: true, ResourceType: "capture", ResourceID: current.metadata.ID, Phase: "capturing", Cleanup: "capture worker exited and partial output is retained according to policy", OperatorHint: "inspect capture source availability and retry", RetryAfterSeconds: 2})
	} else if managed.worker.Stopping() {
		current.metadata.State = "cancelled"
		current.metadata.CompletionReason = managed.stopReason
		if current.metadata.CompletionReason == "" {
			current.metadata.CompletionReason = "cancelled"
		}
	} else if managed.worker.Truncated() {
		current.metadata.State = "truncated"
		current.metadata.CompletionReason = "size_limit"
	} else {
		current.metadata.State = "completed"
		current.metadata.CompletionReason = "completed"
	}
	if current.metadata.Retain && current.path != "" && m.artifacts != nil {
		body, readErr := os.ReadFile(current.path)
		if readErr == nil {
			mediaType := "application/vnd.tcpdump.pcap"
			if current.metadata.Format == "pcapng" {
				mediaType = "application/x-pcapng"
			}
			artifact, artifactErr := m.artifacts.Create(context.Background(), "packet_capture", mediaType, "capture", current.metadata.ID, body, m.retention)
			if artifactErr == nil {
				current.metadata.ArtifactID = artifact.ID
				current.metadata.ArtifactURL = "/api/v1/artifacts/" + string(artifact.ID)
				current.metadata.ExpiresAt = artifact.ExpiresAt
				_ = os.Remove(current.path)
				current.path = ""
			}
		}
	}
	m.persistLocked()
}

func (m *CaptureManager) Get(id domain.ID) (domain.Capture, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value := m.values[id]
	if value == nil {
		return domain.Capture{}, fmt.Errorf("capture not found")
	}
	metadata := value.metadata
	if value.worker != nil {
		metadata.BytesWritten = value.worker.Bytes()
		metadata.Packets = value.worker.Packets()
		metadata.Truncated = value.worker.Truncated()
	}
	return metadata, nil
}

func (m *CaptureManager) Request(id domain.ID) (CaptureRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value := m.values[id]
	if value == nil {
		return CaptureRequest{}, fmt.Errorf("capture not found")
	}
	return value.request, nil
}

func (m *CaptureManager) List() []domain.Capture {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]domain.Capture, 0, len(m.values))
	for _, value := range m.values {
		values = append(values, value.metadata)
	}
	return values
}

func (m *CaptureManager) ListLaboratory(laboratoryID domain.ID) []domain.Capture {
	values := m.List()
	if laboratoryID == "" {
		return values
	}
	filtered := make([]domain.Capture, 0, len(values))
	for _, value := range values {
		if value.LaboratoryID == laboratoryID {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func (m *CaptureManager) AvailableSlots() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	active := 0
	for _, value := range m.values {
		if value.metadata.State == "running" || value.metadata.State == "starting" {
			active++
		}
	}
	if active >= m.concurrent {
		return 0
	}
	return m.concurrent - active
}

func (m *CaptureManager) Stop(id domain.ID) (domain.Capture, error) {
	return m.StopWithReason(id, "cancelled")
}

func (m *CaptureManager) StopWithReason(id domain.ID, reason string) (domain.Capture, error) {
	m.mu.Lock()
	value := m.values[id]
	if value == nil {
		m.mu.Unlock()
		return domain.Capture{}, fmt.Errorf("capture not found")
	}
	if value.worker == nil {
		metadata := value.metadata
		m.mu.Unlock()
		return metadata, nil
	}
	value.stopReason = reason
	if value.metadata.State == "running" || value.metadata.State == "starting" {
		value.metadata.State = "stopping"
	}
	m.persistLocked()
	worker := value.worker
	m.mu.Unlock()
	worker.Stop()
	select {
	case <-worker.Done():
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			metadata, getErr := m.Get(id)
			if getErr != nil || metadata.State != "stopping" {
				return metadata, getErr
			}
			time.Sleep(time.Millisecond)
		}
	case <-time.After(2 * time.Second):
	}
	return m.Get(id)
}

func (m *CaptureManager) StopNetworkObjectLink(linkID domain.ID) {
	m.mu.RLock()
	ids := make([]domain.ID, 0)
	for id, value := range m.values {
		if value.metadata.SourceType == "network_object_link" && value.metadata.SourceID == linkID && value.worker != nil {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	for _, id := range ids {
		_, _ = m.StopWithReason(id, "link_deleted")
	}
}

func (m *CaptureManager) Subscribe(id domain.ID) (<-chan []byte, func(), error) {
	m.mu.RLock()
	value := m.values[id]
	m.mu.RUnlock()
	if value == nil {
		return nil, nil, fmt.Errorf("capture not found")
	}
	if value.worker == nil {
		return nil, nil, domain.Problem{Code: "capture_not_streamable", Message: "capture process is no longer active", ResourceType: "capture", ResourceID: id, Phase: "streaming", Cleanup: "capture metadata and retained artifact remain available", OperatorHint: "download the retained artifact or start a new live capture"}
	}
	channel, cancel := value.worker.Subscribe()
	return channel, cancel, nil
}

func (m *CaptureManager) StartCleanup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				_ = m.Cleanup(now.UTC())
			}
		}
	}()
}

func (m *CaptureManager) StopLaboratory(laboratoryID domain.ID) {
	m.mu.RLock()
	values := make([]*managedCapture, 0)
	for _, value := range m.values {
		if value.metadata.LaboratoryID == laboratoryID {
			values = append(values, value)
		}
	}
	m.mu.RUnlock()
	for _, value := range values {
		if value.worker != nil {
			value.worker.Stop()
		}
	}
}

func (m *CaptureManager) PurgeLaboratory(laboratoryID domain.ID) []domain.ID {
	m.StopLaboratory(laboratoryID)
	m.mu.Lock()
	defer m.mu.Unlock()
	var artifacts []domain.ID
	for id, value := range m.values {
		if value.metadata.LaboratoryID != laboratoryID {
			continue
		}
		if value.metadata.ArtifactID != "" {
			artifacts = append(artifacts, value.metadata.ArtifactID)
		}
		if value.path != "" {
			_ = os.Remove(value.path)
		}
		delete(m.values, id)
	}
	m.persistLocked()
	return artifacts
}

func (m *CaptureManager) Cleanup(now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, value := range m.values {
		if value.metadata.FinishedAt == nil || now.Sub(*value.metadata.FinishedAt) < m.retention {
			continue
		}
		if value.path != "" {
			if err := os.Remove(value.path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		delete(m.values, id)
	}
	m.persistLocked()
	return nil
}

func (m *CaptureManager) metadataPath() string { return filepath.Join(m.directory, "index.json") }

func (m *CaptureManager) persistLocked() {
	if err := os.MkdirAll(m.directory, 0o700); err != nil {
		return
	}
	records := make([]captureRecord, 0, len(m.values))
	for _, value := range m.values {
		records = append(records, captureRecord{Metadata: value.metadata, Path: value.path, Request: value.request})
	}
	body, err := json.Marshal(records)
	if err != nil {
		return
	}
	temporary := m.metadataPath() + ".tmp"
	if err = os.WriteFile(temporary, body, 0o600); err == nil {
		_ = os.Rename(temporary, m.metadataPath())
	}
}

func (m *CaptureManager) load() {
	body, err := os.ReadFile(m.metadataPath())
	if err != nil {
		return
	}
	var records []captureRecord
	if json.Unmarshal(body, &records) != nil {
		return
	}
	now := time.Now().UTC()
	changed := false
	for _, record := range records {
		metadata := record.Metadata
		switch metadata.State {
		case "starting", "running", "stopping":
			metadata.State = "failed"
			metadata.CompletionReason = "service_restart"
			metadata.FinishedAt = &now
			metadata.LastError = structuredProblem(nil, domain.Problem{Code: "capture_abandoned", Message: "capture process ended during service restart", Retryable: true, ResourceType: "capture", ResourceID: metadata.ID, Phase: "recovery", Cleanup: "partial capture metadata and output are retained", OperatorHint: "start a new capture if additional packets are required", RetryAfterSeconds: 1})
			changed = true
		}
		m.values[metadata.ID] = &managedCapture{metadata: metadata, path: record.Path, request: record.Request}
	}
	if changed {
		m.persistLocked()
	}
}
