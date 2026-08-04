package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
)

const defaultTrafficFilterColor = "#f59e0b"

var trafficFilterColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type managedFilter struct {
	metadata    domain.TrafficFilter
	correlator  *captureRuntime.Correlator
	match       captureRuntime.Match
	interfaces  map[domain.ID]bool
	links       map[domain.ID]bool
	objectLinks map[domain.ID]bool
	decoders    map[string]*captureRuntime.PacketDecoder
	mu          sync.Mutex
	captureIDs  []domain.ID
}

type TrafficFilterManager struct {
	mu       sync.RWMutex
	values   map[domain.ID]*managedFilter
	pending  map[domain.ID]*managedFilter
	captures *CaptureManager
	path     string
}

type trafficFilterRecord struct {
	Metadata domain.TrafficFilter `json:"metadata"`
	Match    captureRuntime.Match `json:"match"`
}

func NewTrafficFilterManager(captures ...*CaptureManager) *TrafficFilterManager {
	manager := &TrafficFilterManager{values: map[domain.ID]*managedFilter{}, pending: map[domain.ID]*managedFilter{}}
	if len(captures) > 0 && captures[0] != nil {
		manager.captures = captures[0]
		manager.path = filepath.Join(filepath.Dir(captures[0].directory), "traffic-filters.json")
		manager.load()
	}
	return manager
}

func (m *TrafficFilterManager) Start(labID domain.ID, match captureRuntime.Match, maximum int) (domain.TrafficFilter, error) {
	return m.StartScoped(labID, match, maximum, nil, nil)
}

func (m *TrafficFilterManager) StartScoped(labID domain.ID, match captureRuntime.Match, maximum int, interfaceIDs, linkIDs []domain.ID) (domain.TrafficFilter, error) {
	return m.StartScopedAs(domain.NewID(), labID, match, maximum, interfaceIDs, linkIDs)
}

func (m *TrafficFilterManager) StartScopedAs(id, labID domain.ID, match captureRuntime.Match, maximum int, interfaceIDs, linkIDs []domain.ID) (domain.TrafficFilter, error) {
	return m.StartScopedAsWithColor(id, labID, match, maximum, interfaceIDs, linkIDs, defaultTrafficFilterColor)
}

func (m *TrafficFilterManager) StartScopedAsWithColor(id, labID domain.ID, match captureRuntime.Match, maximum int, interfaceIDs, linkIDs []domain.ID, color string) (domain.TrafficFilter, error) {
	return m.StartScopedAsWithObjectLinks(id, labID, match, maximum, interfaceIDs, linkIDs, nil, color)
}

func (m *TrafficFilterManager) StartScopedAsWithObjectLinks(id, labID domain.ID, match captureRuntime.Match, maximum int, interfaceIDs, linkIDs, objectLinkIDs []domain.ID, color string) (domain.TrafficFilter, error) {
	interfaceIDs = uniqueTrafficFilterIDs(interfaceIDs)
	linkIDs = uniqueTrafficFilterIDs(linkIDs)
	objectLinkIDs = uniqueTrafficFilterIDs(objectLinkIDs)
	if len(interfaceIDs)+len(linkIDs)+len(objectLinkIDs) == 0 {
		return domain.TrafficFilter{}, domain.Problem{Code: "invalid_traffic_filter_scope", Message: "select at least one interface or link to observe", ResourceType: "traffic_filter", ResourceID: id, Phase: "traffic_filter_admission", Cleanup: "no capture resources created", OperatorHint: "choose one or more connected interfaces or links and retry"}
	}
	expression, err := captureRuntime.Compile(match)
	if err != nil {
		return domain.TrafficFilter{}, err
	}
	if maximum <= 0 || maximum > 100000 {
		maximum = 10000
	}
	if color == "" {
		color = defaultTrafficFilterColor
	}
	if !trafficFilterColorPattern.MatchString(color) {
		return domain.TrafficFilter{}, domain.Problem{Code: "invalid_traffic_filter_color", Message: "traffic filter color must be a six-digit hex color", ResourceType: "traffic_filter", ResourceID: id, Phase: "traffic_filter_admission"}
	}
	m.mu.RLock()
	existing := m.values[id]
	m.mu.RUnlock()
	if existing != nil && existing.metadata.State == "running" {
		return existing.metadata, nil
	}
	value := domain.TrafficFilter{ID: id, LaboratoryID: labID, Expression: expression, Color: color, State: "running", MaxObservations: maximum, InterfaceIDs: interfaceIDs, LinkIDs: linkIDs, NetworkObjectLinkIDs: objectLinkIDs, Observations: []domain.TrafficObservation{}, CreatedAt: time.Now().UTC()}
	interfaces, links, objectLinks := map[domain.ID]bool{}, map[domain.ID]bool{}, map[domain.ID]bool{}
	for _, id := range interfaceIDs {
		interfaces[id] = true
	}
	for _, id := range linkIDs {
		links[id] = true
	}
	for _, id := range objectLinkIDs {
		objectLinks[id] = true
	}
	managed := &managedFilter{metadata: value, correlator: captureRuntime.NewCorrelator(2*time.Second, maximum), match: match, interfaces: interfaces, links: links, objectLinks: objectLinks, decoders: map[string]*captureRuntime.PacketDecoder{}}
	prepared := false
	prepare := func() {
		m.mu.Lock()
		m.pending[value.ID] = managed
		m.mu.Unlock()
		prepared = true
	}
	commit := func() {
		m.mu.Lock()
		delete(m.pending, value.ID)
		m.values[value.ID] = managed
		m.persistLocked()
		m.mu.Unlock()
	}
	rollback := func() {
		for _, captureID := range managed.captureIDs {
			_, _ = m.captures.Stop(captureID)
		}
		m.mu.Lock()
		delete(m.pending, value.ID)
		m.mu.Unlock()
	}
	if m.captures != nil {
		required := len(interfaceIDs) + len(linkIDs) + 2*len(objectLinkIDs)
		available := m.captures.AvailableSlots()
		if required > available {
			return domain.TrafficFilter{}, domain.Problem{Code: "resource_exhausted", Message: fmt.Sprintf("traffic filter needs %d capture slots but only %d are available", required, available), Retryable: true, ResourceType: "traffic_filter", ResourceID: id, Phase: "traffic_filter_admission", Cleanup: "no capture resources created", OperatorHint: "reduce the selected observation scope or stop active captures and retry", RetryAfterSeconds: 2}
		}
		availableBytes := m.captures.AvailableBytes()
		captureMaxBytes := availableBytes / int64(required)
		if captureMaxBytes > 64<<20 {
			captureMaxBytes = 64 << 20
		}
		if captureMaxBytes < 1<<20 {
			return domain.TrafficFilter{}, domain.Problem{Code: "resource_exhausted", Message: fmt.Sprintf("traffic filter needs at least %d bytes per capture but only %d bytes are available across %d sources", 1<<20, availableBytes, required), Retryable: true, ResourceType: "traffic_filter", ResourceID: id, Phase: "traffic_filter_admission", Cleanup: "no capture resources created", OperatorHint: "stop active captures or remove retained capture artifacts and retry", RetryAfterSeconds: 2}
		}
		prepare()
		for _, id := range interfaceIDs {
			captureValue, captureErr := m.captures.Start(context.Background(), CaptureRequest{LaboratoryID: labID, SourceType: "interface", SourceID: id, Interface: linuxnet.HostInterfaceName(id), Purpose: "traffic_filter", ParentID: value.ID, Filter: expression, Format: "pcap", MaxBytes: captureMaxBytes})
			if captureErr != nil {
				rollback()
				return domain.TrafficFilter{}, captureErr
			}
			managed.captureIDs = append(managed.captureIDs, captureValue.ID)
		}
		for _, id := range linkIDs {
			captureValue, captureErr := m.captures.Start(context.Background(), CaptureRequest{LaboratoryID: labID, SourceType: "link", SourceID: id, Interface: linuxnet.LinkBridgeName(id), Purpose: "traffic_filter", ParentID: value.ID, Filter: expression, Format: "pcap", MaxBytes: captureMaxBytes})
			if captureErr != nil {
				rollback()
				return domain.TrafficFilter{}, captureErr
			}
			managed.captureIDs = append(managed.captureIDs, captureValue.ID)
		}
		for _, id := range objectLinkIDs {
			for _, direction := range []string{"egress", "ingress"} {
				captureValue, captureErr := m.captures.Start(context.Background(), CaptureRequest{LaboratoryID: labID, SourceType: "network_object_link", SourceID: id, Purpose: "traffic_filter", ParentID: value.ID, Filter: expression, Format: "pcap", MaxBytes: captureMaxBytes, Direction: direction})
				if captureErr != nil {
					rollback()
					return domain.TrafficFilter{}, captureErr
				}
				managed.captureIDs = append(managed.captureIDs, captureValue.ID)
			}
		}
	}
	if !prepared {
		prepare()
	}
	commit()
	return value, nil
}

func (m *TrafficFilterManager) Observe(id domain.ID, fingerprint string, interfaceID, linkID domain.ID, direction string, length int, at time.Time) error {
	m.mu.RLock()
	value := m.values[id]
	m.mu.RUnlock()
	if value == nil {
		return fmt.Errorf("traffic filter not found")
	}
	value.correlator.Observe(fingerprint, interfaceID, linkID, direction, length, at)
	return nil
}

func (m *TrafficFilterManager) ObserveCapture(laboratoryID, interfaceID, linkID, objectLinkID domain.ID, direction, format string, chunk []byte, at time.Time) {
	m.mu.RLock()
	values := make([]*managedFilter, 0, len(m.values))
	for _, value := range m.values {
		if value.metadata.LaboratoryID == laboratoryID && value.metadata.State == "running" {
			values = append(values, value)
		}
	}
	m.mu.RUnlock()
	for _, value := range values {
		m.observeCapture(value, "", interfaceID, linkID, objectLinkID, direction, format, chunk, at)
	}
}

func (m *TrafficFilterManager) ObserveManagedCapture(captureID, parentID, laboratoryID, interfaceID, linkID, objectLinkID domain.ID, direction, format string, chunk []byte, at time.Time) {
	if parentID == "" {
		return
	}
	m.mu.RLock()
	value := m.values[parentID]
	if value == nil {
		value = m.pending[parentID]
	}
	m.mu.RUnlock()
	if value == nil || value.metadata.LaboratoryID != laboratoryID || value.metadata.State != "running" {
		return
	}
	m.observeCapture(value, captureID, interfaceID, linkID, objectLinkID, direction, format, chunk, at)
}

func (m *TrafficFilterManager) observeCapture(value *managedFilter, captureID, interfaceID, linkID, objectLinkID domain.ID, direction, format string, chunk []byte, at time.Time) {
	if len(value.interfaces) > 0 || len(value.links) > 0 || len(value.objectLinks) > 0 {
		matchesInterface := interfaceID != "" && value.interfaces[interfaceID]
		matchesLink := linkID != "" && value.links[linkID]
		matchesObjectLink := objectLinkID != "" && value.objectLinks[objectLinkID]
		if !matchesInterface && !matchesLink && !matchesObjectLink {
			return
		}
	}
	value.mu.Lock()
	decoderKey := string(captureID) + ":" + string(interfaceID) + ":" + string(linkID) + ":" + string(objectLinkID) + ":" + direction
	decoder := value.decoders[decoderKey]
	if decoder == nil {
		decoder = captureRuntime.NewPacketDecoder(format)
		value.decoders[decoderKey] = decoder
	}
	packets, err := decoder.Add(chunk)
	value.mu.Unlock()
	if err != nil {
		return
	}
	for _, packet := range packets {
		if captureRuntime.Matches(value.match, packet.Key) {
			if objectLinkID != "" {
				value.correlator.ObserveNetworkObjectLinkPacket(captureRuntime.Fingerprint(packet.Key), objectLinkID, networkObjectLinkDirection(direction), packet.Key, at)
			} else {
				value.correlator.ObservePacket(captureRuntime.Fingerprint(packet.Key), interfaceID, linkID, direction, packet.Key, at)
			}
		}
	}
}

func networkObjectLinkDirection(direction string) string {
	switch direction {
	case "egress", "a_to_b":
		return "a_to_b"
	case "ingress", "b_to_a":
		return "b_to_a"
	default:
		return "ambiguous"
	}
}

func (m *TrafficFilterManager) Get(id domain.ID) (domain.TrafficFilter, bool, error) {
	m.mu.RLock()
	value := m.values[id]
	m.mu.RUnlock()
	if value == nil {
		return domain.TrafficFilter{}, false, fmt.Errorf("traffic filter not found")
	}
	result := value.metadata
	var ambiguous bool
	result.Observations, ambiguous = value.correlator.Snapshot()
	return result, ambiguous, nil
}

func (m *TrafficFilterManager) List(laboratoryID domain.ID) []domain.TrafficFilter {
	m.mu.RLock()
	values := make([]*managedFilter, 0, len(m.values))
	for _, value := range m.values {
		if laboratoryID == "" || value.metadata.LaboratoryID == laboratoryID {
			values = append(values, value)
		}
	}
	m.mu.RUnlock()
	result := make([]domain.TrafficFilter, 0, len(values))
	for _, value := range values {
		metadata := value.metadata
		metadata.Observations, _ = value.correlator.Snapshot()
		result = append(result, metadata)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result
}

func uniqueTrafficFilterIDs(values []domain.ID) []domain.ID {
	seen := make(map[domain.ID]struct{}, len(values))
	result := make([]domain.ID, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (m *TrafficFilterManager) Definition(id domain.ID) (domain.TrafficFilter, captureRuntime.Match, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value := m.values[id]
	if value == nil {
		return domain.TrafficFilter{}, captureRuntime.Match{}, fmt.Errorf("traffic filter not found")
	}
	return value.metadata, value.match, nil
}

func (m *TrafficFilterManager) Stop(id domain.ID) (domain.TrafficFilter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value := m.values[id]
	if value == nil {
		return domain.TrafficFilter{}, fmt.Errorf("traffic filter not found")
	}
	now := time.Now().UTC()
	value.metadata.State = "stopped"
	value.metadata.FinishedAt = &now
	value.metadata.Observations, _ = value.correlator.Snapshot()
	if m.captures != nil {
		for _, captureID := range value.captureIDs {
			_, _ = m.captures.Stop(captureID)
		}
	}
	m.persistLocked()
	return value.metadata, nil
}

func (m *TrafficFilterManager) DeleteHistory(id domain.ID) (domain.TrafficFilter, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value := m.values[id]
	if value == nil {
		return domain.TrafficFilter{}, fmt.Errorf("traffic filter not found")
	}
	if value.metadata.State == "starting" || value.metadata.State == "running" || value.metadata.State == "stopping" {
		return domain.TrafficFilter{}, domain.Problem{Code: "traffic_filter_active", Message: "stop the traffic filter before deleting its history", Retryable: true, ResourceType: "traffic_filter", ResourceID: id, Phase: "history_delete", Cleanup: "active filter and capture workers remain unchanged", OperatorHint: "stop the active session and retry deletion", RetryAfterSeconds: 1}
	}
	metadata := value.metadata
	delete(m.values, id)
	delete(m.pending, id)
	m.persistLocked()
	return metadata, nil
}

func (m *TrafficFilterManager) StopNetworkObjectLink(linkID domain.ID) {
	m.mu.RLock()
	ids := make([]domain.ID, 0)
	for id, value := range m.values {
		if value.objectLinks[linkID] && value.metadata.State == "running" {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()
	for _, id := range ids {
		_, _ = m.Stop(id)
	}
}

func (m *TrafficFilterManager) persistLocked() {
	if m.path == "" {
		return
	}
	records := make([]trafficFilterRecord, 0, len(m.values))
	for _, value := range m.values {
		metadata := value.metadata
		metadata.Observations, _ = value.correlator.Snapshot()
		records = append(records, trafficFilterRecord{Metadata: metadata, Match: value.match})
	}
	body, err := json.Marshal(records)
	if err != nil || os.MkdirAll(filepath.Dir(m.path), 0o700) != nil {
		return
	}
	temporary := m.path + ".tmp"
	if os.WriteFile(temporary, body, 0o600) == nil {
		_ = os.Rename(temporary, m.path)
	}
}

func (m *TrafficFilterManager) load() {
	body, err := os.ReadFile(m.path)
	if err != nil {
		return
	}
	var records []trafficFilterRecord
	if json.Unmarshal(body, &records) != nil {
		return
	}
	now := time.Now().UTC()
	for _, record := range records {
		metadata := record.Metadata
		if metadata.State == "running" || metadata.State == "starting" || metadata.State == "stopping" {
			metadata.State = "failed"
			metadata.FinishedAt = &now
			metadata.LastError = &domain.Problem{Code: "traffic_filter_abandoned", Message: "Traffic Filter observation ended during service restart", Retryable: true, ResourceType: "traffic_filter", ResourceID: metadata.ID, Phase: "recovery", Cleanup: "capture workers were finalized independently", OperatorHint: "start a new Traffic Filter to resume observation", RetryAfterSeconds: 1}
		}
		correlator := captureRuntime.NewCorrelator(2*time.Second, metadata.MaxObservations)
		for _, observation := range metadata.Observations {
			length := int(observation.Bytes / max(observation.Count, 1))
			correlator.ObservePacket(observation.Fingerprint, observation.InterfaceID, observation.LinkID, observation.Direction, captureRuntime.PacketKey{
				Source: observation.SourceAddress, Destination: observation.DestinationAddress,
				SourceMAC: observation.SourceMAC, DestinationMAC: observation.DestinationMAC, Length: length,
			}, observation.FirstSeen)
		}
		interfaces, links, objectLinks := map[domain.ID]bool{}, map[domain.ID]bool{}, map[domain.ID]bool{}
		for _, id := range metadata.InterfaceIDs {
			interfaces[id] = true
		}
		for _, id := range metadata.LinkIDs {
			links[id] = true
		}
		for _, id := range metadata.NetworkObjectLinkIDs {
			objectLinks[id] = true
		}
		m.values[metadata.ID] = &managedFilter{metadata: metadata, correlator: correlator, match: record.Match, interfaces: interfaces, links: links, objectLinks: objectLinks, decoders: map[string]*captureRuntime.PacketDecoder{}}
	}
	m.persistLocked()
}
