package compliance

import "testing"

func TestScanEvidenceRejectsSecretsAndPayloadArtifacts(t *testing.T) {
	secretAssignment := "pass" + "word=" + "example"
	if findings := ScanEvidence("result.json", []byte(secretAssignment)); len(findings) == 0 {
		t.Fatal("expected secret finding")
	}
	if findings := ScanEvidence("capture.pcap", []byte("metadata")); len(findings) == 0 {
		t.Fatal("expected artifact finding")
	}
	if findings := ScanEvidence("result.json", []byte(`{"outcome":"passed"}`)); len(findings) != 0 {
		t.Fatalf("unexpected findings: %v", findings)
	}
}
