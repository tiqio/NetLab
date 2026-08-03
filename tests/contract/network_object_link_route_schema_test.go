package contract

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNetworkObjectLinkAndRouteContractDelta(t *testing.T) {
	body, err := os.ReadFile("../../specs/005-network-object-links-routes/contracts/openapi-delta.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err = yaml.Unmarshal(body, &document); err != nil {
		t.Fatal(err)
	}
	schemas := object(t, object(t, document, "components"), "schemas")
	paths := object(t, document, "paths")

	requireSchemaProperties(t, schemas, "NetworkObjectLinkTaskEnvelope", "network_object_link", "task")
	requireSchemaProperties(t, schemas, "DockerStaticRoute", "destination", "gateway", "metric")
	requireSchemaProperties(t, schemas, "NodeNetworkInterfaceSettings", "id", "name", "driver", "modes", "addresses", "routes")
	requireSchemaProperties(t, schemas, "NodeSettings", "name", "cpu_count", "cpu_quota_micros", "memory_mib", "interface_limit", "process_limit", "network_interfaces")
	requireSchemaProperties(t, schemas, "NetworkObjectLink", "desired_state", "observed_state")
	requireSchemaProperties(t, schemas, "StartTrafficFilter", "network_object_link_ids")
	requireSchemaProperties(t, schemas, "TrafficObservation", "resource_type", "resource_id", "direction", "first_seen", "last_seen", "count", "bytes")

	for path, methodAndOperation := range map[string][2]string{
		"/api/v1/labs/{labId}/network-object-links": {"post", "createNetworkObjectLink"},
		"/api/v1/network-object-links/{linkId}":     {"delete", "deleteNetworkObjectLink"},
		"/api/v1/nodes/{nodeId}/settings":           {"put", "updateNodeSettings"},
	} {
		operation := object(t, object(t, paths, path), methodAndOperation[0])
		if operation["operationId"] != methodAndOperation[1] {
			t.Errorf("%s %s operationId=%v", methodAndOperation[0], path, operation["operationId"])
		}
	}

	assertSchemaEnumContains(t, schemas, "StartCapture", "source_type", "network_object_link")
	assertSchemaEnumContains(t, schemas, "TrafficObservation", "resource_type", "network_object_link")
	assertSchemaEnumContains(t, schemas, "TrafficObservation", "direction", "a_to_b", "b_to_a", "ambiguous")
}

func TestGeneratedClientIncludesObjectLinkCaptureAndFilterTypes(t *testing.T) {
	body, err := os.ReadFile("../../web/src/api/generated.ts")
	if err != nil {
		t.Fatal(err)
	}
	client := string(body)
	for _, expected := range []string{
		`source_type: "interface" | "link" | "network_object_link"`,
		`network_object_link_ids?: string[]`,
		`network_object_link_id?: string`,
		`resource_id?: string`,
	} {
		if !strings.Contains(client, expected) {
			t.Errorf("generated client missing %q", expected)
		}
	}
}

func assertSchemaEnumContains(t *testing.T, schemas map[string]any, schemaName, propertyName string, expected ...string) {
	t.Helper()
	schema := object(t, schemas, schemaName)
	properties := object(t, schema, "properties")
	property := object(t, properties, propertyName)
	raw, ok := property["enum"].([]any)
	if !ok {
		t.Fatalf("schema %s property %s enum=%T", schemaName, propertyName, property["enum"])
	}
	actual := make(map[string]bool, len(raw))
	for _, value := range raw {
		actual[value.(string)] = true
	}
	for _, value := range expected {
		if !actual[value] {
			t.Errorf("schema %s property %s missing enum value %s", schemaName, propertyName, value)
		}
	}
}
