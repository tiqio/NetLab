package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/reconcile"
	"github.com/netlab/netlab/internal/app/task"
	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/linuxnet"
	storesqlite "github.com/netlab/netlab/internal/store/sqlite"
)

type vlanHTTPExecutor struct{}

func (vlanHTTPExecutor) Run(context.Context, string, ...string) error { return nil }
func (vlanHTTPExecutor) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	command := name + " " + strings.Join(args, " ")
	if strings.Contains(command, "vlan show dev eth0") {
		return []byte(`[{"ifname":"eth0","vlans":[{"vlan":1,"flags":["PVID","Egress Untagged"]}]}]`), nil
	}
	return []byte("usable"), nil
}

type ipv6DiagnosticsHTTPExecutor struct{}

func (ipv6DiagnosticsHTTPExecutor) Run(context.Context, string, ...string) error { return nil }
func (ipv6DiagnosticsHTTPExecutor) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	switch {
	case strings.Contains(command, "-j address show"):
		return []byte(`[{"ifname":"eth0","addr_info":[{"local":"fd10::1","prefixlen":64,"scope":"global"}]}]`), nil
	case strings.Contains(command, "-j route show"):
		return []byte(`[{"dst":"fd20::/64","gateway":"fd10::fe"}]`), nil
	case strings.Contains(command, "-j neigh show"):
		return []byte(`[{"dst":"fd10::fe","dev":"eth0","state":["FAILED"]}]`), nil
	case strings.Contains(command, "net.ipv6.conf.all.forwarding"):
		return []byte("1\n"), nil
	case strings.Contains(command, "net.ipv4.ip_forward"):
		return []byte("0\n"), nil
	default:
		return nil, nil
	}
}

func TestNetworkVLANHTTPValidationAndObservedDiagnostics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:http-vlan?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "http vlan", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: lab.ID, Name: "switch", Kind: domain.NetworkSwitchL2, Revision: 1, DesiredState: "active", ObservedState: "pending", Config: map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{20}}}}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	runtime, err := linuxnet.NewSwitchL2Runtime(vlanHTTPExecutor{})
	if err != nil {
		t.Fatal(err)
	}
	runner := task.NewRunner(repositories, 1, 8)
	defer runner.Close()
	service := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{SwitchL2: runtime})
	engine := gin.New()
	NewNetworkHandlers(service, reconcile.NewNetworkObjectTaskService(service, runner), nil, runtime).Register(engine)

	body, _ := json.Marshal(map[string]any{"name": "switch", "config": map[string]any{"vlan_filtering": true, "ports": []any{map[string]any{"name": "eth0", "pvid": 10, "tagged": []any{10}}}}})
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/network-objects/"+string(object.ID), bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", "1")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code < 400 || !strings.Contains(response.Body.String(), "cannot also be tagged") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/network-objects/"+string(object.ID)+"/diagnostics", nil)
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"pvid":1`) || !strings.Contains(response.Body.String(), "VLAN membership differs") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestNetworkL3HTTPDiagnosticsExposeIPv6PathStop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	database, err := storesqlite.Open(ctx, "file:http-ipv6-diagnostics?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	repositories := storesqlite.NewRepositories(database)
	topology := storesqlite.NewTopologyRepository(database)
	now := time.Now().UTC()
	lab := domain.Laboratory{ID: domain.NewID(), Name: "http ipv6 diagnostics", Revision: 1, RecoveryPolicy: domain.RecoveryRemainStopped, LifecycleState: "active", CreatedAt: now, UpdatedAt: now}
	if err = topology.CreateLaboratory(ctx, lab); err != nil {
		t.Fatal(err)
	}
	object := domain.NetworkObject{ID: domain.NewID(), LaboratoryID: lab.ID, Name: "router", Kind: domain.NetworkSwitchL3, Revision: 1, DesiredState: "active", ObservedState: "active", Config: map[string]any{
		"interfaces": []any{map[string]any{"name": "eth0", "addresses": []any{"fd10::1/64"}}}, "routes": []any{map[string]any{"destination": "fd20::/64", "gateway": "fd10::fe"}}, "forward_ipv6": true,
	}, CreatedAt: now, UpdatedAt: now}
	if err = repositories.CreateNetworkObject(ctx, object); err != nil {
		t.Fatal(err)
	}
	runtime, err := linuxnet.NewSwitchL3Runtime(ipv6DiagnosticsHTTPExecutor{})
	if err != nil {
		t.Fatal(err)
	}
	service := reconcile.NewNetworkObjectService(repositories, reconcile.NetworkRuntimeDispatch{SwitchL3: runtime})
	engine := gin.New()
	NewNetworkHandlers(service, nil, nil, runtime).Register(engine)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/network-objects/"+string(object.ID)+"/diagnostics", nil)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"stop_at":"neighbor_discovery"`) || !strings.Contains(response.Body.String(), `"ipv6_neighbors"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
