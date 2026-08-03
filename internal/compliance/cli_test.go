package compliance

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteReportProducesSingleConclusion(t *testing.T) {
	action := "fix"
	ledger := Ledger{Findings: []Finding{{Status: "open", NextAction: &action}}}
	var output bytes.Buffer
	if err := WriteReport(&output, ledger, false); err != nil {
		t.Fatal(err)
	}
	if strings.Count(output.String(), "Conclusion:") != 1 || !strings.Contains(output.String(), "not_ready") {
		t.Fatalf("unexpected report: %s", output.String())
	}
}

func TestExitCodesRemainStable(t *testing.T) {
	tests := []struct {
		message string
		code    int
	}{
		{"malformed document", ExitMalformed},
		{"finding references missing evidence", ExitInvalidReference},
		{"verified finding uses non-accepted evidence", ExitStaleContradiction},
		{"expected exactly one externally reachable authoritative instance", ExitDeploymentAuthority},
		{"prohibited artifact type", ExitProhibitedContent},
	}
	for _, test := range tests {
		if actual := ExitCode(errors.New(test.message)); actual != test.code {
			t.Fatalf("%q: got %d want %d", test.message, actual, test.code)
		}
	}
}
