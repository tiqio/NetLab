package contract_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/netlab/netlab/internal/compliance"
)

func TestFortiGateRequiresReviewedOperatorMediaOrException(t *testing.T) {
	body, err := os.ReadFile("../../compliance/template-readiness.json")
	if err != nil {
		t.Fatal(err)
	}
	var matrix compliance.TemplateReadinessMatrix
	if err := json.Unmarshal(body, &matrix); err != nil {
		t.Fatal(err)
	}
	for _, item := range matrix.Templates {
		if item["template_key"] != "fortigate" {
			continue
		}
		status, _ := item["status"].(string)
		if status != "blocked" && status != "accepted_exception" && status != "genuine_validated" {
			t.Fatalf("invalid FortiGate readiness %s", status)
		}
		if status == "mechanics_validated" {
			t.Fatal("FortiGate must not be represented by a substitute mechanics workload")
		}
		return
	}
	t.Fatal("FortiGate readiness record missing")
}
