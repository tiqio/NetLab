package contract_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	httpapi "github.com/netlab/netlab/internal/api/http"
	"github.com/netlab/netlab/internal/api/mcp"
	"github.com/netlab/netlab/internal/api/stream"
	"github.com/netlab/netlab/internal/app/events"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	captureRuntime "github.com/netlab/netlab/internal/runtime/capture"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type observationNetworkRepository struct {
	link   domain.NetworkObjectLink
	object domain.NetworkObject
}

func (r observationNetworkRepository) GetNetworkObjectLink(context.Context, domain.ID) (domain.NetworkObjectLink, error) {
	return r.link, nil
}

func (r observationNetworkRepository) GetNetworkObject(context.Context, domain.ID) (domain.NetworkObject, error) {
	return r.object, nil
}

func TestNetworkObjectLinkCaptureAndObservationHTTPMCPParity(t *testing.T) {
	installObservationCaptureTools(t)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:object-link-observation-contract?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	object := domain.NetworkObject{ID: "object-a", LaboratoryID: "lab", Kind: domain.NetworkSwitchL3}
	captures := reconcile.NewCaptureManager(t.TempDir(), 4, 8<<20, time.Hour)
	captures.SetNetworkObjectRepository(observationNetworkRepository{
		link: domain.NetworkObjectLink{ID: "object-link", LaboratoryID: "lab", ObjectAID: object.ID, PortAName: "swp1", ObjectBID: "object-b", PortBName: "swp2"}, object: object,
	})
	filters := reconcile.NewTrafficFilterManager()
	operations := reconcile.NewCaptureTaskService(captures, filters, runner)
	engine := gin.New()
	httpapi.NewCaptureHandlers(captures, filters, operations).Register(engine)

	body := []byte(`{"laboratory_id":"lab","source_type":"network_object_link","source_id":"object-link","format":"pcap","max_bytes":1048576}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/captures", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "object-link-capture")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var restEnvelope struct {
		Capture   domain.Capture       `json:"capture"`
		Task      domain.OperationTask `json:"task"`
		StreamURL string               `json:"stream_url"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &restEnvelope); err != nil {
		t.Fatal(err)
	}
	if restEnvelope.Capture.SourceType != "network_object_link" || restEnvelope.Capture.SourceID != "object-link" || restEnvelope.StreamURL == "" {
		t.Fatalf("REST envelope=%+v", restEnvelope)
	}

	services := mcp.Services{Captures: captures, Filters: filters, CaptureOps: operations}
	startCapture := findObservationTool(t, mcp.Tools(services), "netlab.captures.start")
	mcpContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	result, err := startCapture.Handler(mcpContext, map[string]any{"laboratory_id": "lab", "source_type": "network_object_link", "source_id": "object-link", "format": "pcap", "max_bytes": float64(1048576), "idempotency_key": "object-link-capture"})
	if err != nil {
		t.Fatal(err)
	}
	mcpEnvelope := result.(map[string]any)
	if mcpEnvelope["task"].(domain.OperationTask).ID != restEnvelope.Task.ID || mcpEnvelope["capture"].(domain.Capture).ID != restEnvelope.Capture.ID {
		t.Fatalf("REST=%+v MCP=%+v", restEnvelope, mcpEnvelope)
	}

	filter, err := filters.StartScopedAsWithObjectLinks("filter", "lab", captureRuntime.Match{Protocol: "udp", DestinationPort: 53}, 100, nil, nil, []domain.ID{"object-link"}, "#22c55e")
	if err != nil {
		t.Fatal(err)
	}
	filters.ObserveCapture("lab", "", "", "object-link", "egress", "pcap", observationUDPPcap(), time.Now().UTC())
	getRequest := httptest.NewRequest(http.MethodGet, "/api/v1/traffic-filters/"+string(filter.ID), nil)
	getResponse := httptest.NewRecorder()
	engine.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || !bytes.Contains(getResponse.Body.Bytes(), []byte(`"resource_type":"network_object_link"`)) || !bytes.Contains(getResponse.Body.Bytes(), []byte(`"direction":"a_to_b"`)) {
		t.Fatalf("filter status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}
	getFilter := findObservationTool(t, mcp.Tools(services), "netlab.traffic_filters.get")
	result, err = getFilter.Handler(mcpContext, map[string]any{"filter_id": string(filter.ID)})
	if err != nil {
		t.Fatal(err)
	}
	filterEnvelope := result.(map[string]any)
	observations := filterEnvelope["traffic_filter"].(domain.TrafficFilter).Observations
	if len(observations) != 1 || observations[0].ResourceID != "object-link" || observations[0].Direction != "a_to_b" {
		t.Fatalf("MCP observations=%+v", observations)
	}
}

func TestObjectLinkObservationEventContractUsesDistinctOrderedTypes(t *testing.T) {
	now := time.Now().UTC()
	capture := domain.Capture{ID: "capture", LaboratoryID: "lab", SourceType: "network_object_link", SourceID: "object-link", State: "completed", CompletionReason: "completed", FinishedAt: &now}
	observation := domain.TrafficObservation{Fingerprint: "udp", ResourceType: "network_object_link", ResourceID: "object-link", NetworkObjectLinkID: "object-link", Direction: "a_to_b", FirstSeen: now, LastSeen: now, Count: 1, Bytes: 64}
	captureEvent := events.CaptureEvent(capture, "capture-task")
	trafficEvent := events.TrafficObservationEvent(domain.TrafficFilter{ID: "filter", LaboratoryID: "lab", Color: "#22c55e"}, observation, "filter-task")
	if captureEvent.Type != stream.EventCaptureCompleted || trafficEvent.Type != stream.EventTrafficFilterObservation {
		t.Fatalf("capture=%+v traffic=%+v", captureEvent, trafficEvent)
	}
	if trafficEvent.ResourceType != "network_object_link" || trafficEvent.ResourceID != "object-link" || trafficEvent.Type == stream.EventNetworkObjectLinkStateChanged {
		t.Fatalf("traffic event=%+v", trafficEvent)
	}
}

func findObservationTool(t *testing.T, tools []mcp.Tool, name string) mcp.Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return mcp.Tool{}
}

func installObservationCaptureTools(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "dumpcap"), []byte("#!/bin/sh\nprintf capture-data\nexec sleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "ip"), []byte("#!/bin/sh\nif [ \"$1\" = netns ] && [ \"$2\" = exec ]; then shift 3; exec \"$@\"; fi\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func observationUDPPcap() []byte {
	frame := make([]byte, 14+20+8)
	binary.BigEndian.PutUint16(frame[12:14], 0x0800)
	frame[14], frame[23] = 0x45, 17
	copy(frame[26:30], []byte{192, 0, 2, 1})
	copy(frame[30:34], []byte{192, 0, 2, 53})
	binary.BigEndian.PutUint16(frame[34:36], 1234)
	binary.BigEndian.PutUint16(frame[36:38], 53)
	pcap := make([]byte, 40+len(frame))
	binary.LittleEndian.PutUint32(pcap[:4], 0xa1b2c3d4)
	binary.LittleEndian.PutUint32(pcap[32:36], uint32(len(frame)))
	copy(pcap[40:], frame)
	return pcap
}
