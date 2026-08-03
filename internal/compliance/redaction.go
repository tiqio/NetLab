package compliance

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var prohibitedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|secret|token|private[_ -]?key)\s*[:=]\s*[^\s,]+`),
	regexp.MustCompile(`(?i)"(password|passwd|secret|token|private[_ -]?key)"\s*:\s*"[^"]+"`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
}

var prohibitedExtensions = map[string]struct{}{".qcow2": {}, ".raw": {}, ".iso": {}, ".pcap": {}, ".pcapng": {}, ".key": {}}

func ScanEvidence(path string, body []byte) []string {
	findings := []string{}
	if _, prohibited := prohibitedExtensions[strings.ToLower(filepath.Ext(path))]; prohibited {
		findings = append(findings, "prohibited artifact type")
	}
	for _, pattern := range prohibitedPatterns {
		if pattern.Match(body) {
			findings = append(findings, fmt.Sprintf("matched %s", pattern.String()))
		}
	}
	if bytes.IndexByte(body, 0) >= 0 {
		findings = append(findings, "binary payload")
	}
	return findings
}
