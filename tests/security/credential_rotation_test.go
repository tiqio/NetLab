package security_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRepositoryContainsNoAssignedCredentialValues(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.ts") || strings.HasSuffix(path, ".spec.ts") || strings.Contains(path, string(filepath.Separator)+"tests"+string(filepath.Separator)) || strings.Contains(path, "node_modules") || strings.Contains(path, "webdist") || strings.Contains(path, string(filepath.Separator)+"bin"+string(filepath.Separator)) || strings.Contains(path, "test-results") {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".go" && extension != ".ts" && extension != ".vue" && extension != ".json" && extension != ".yaml" && extension != ".yml" && extension != ".md" && extension != ".sh" && extension != ".service" && extension != ".nft" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if containsAssignedCredential(extension, body) {
			t.Fatalf("credential-like assignment in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAssignedCredentialDetectionDistinguishesValuesFromSchemasAndVariables(t *testing.T) {
	for _, test := range []struct {
		name      string
		extension string
		body      string
		expected  bool
	}{
		{name: "quoted source literal", extension: ".go", body: `password := "hardcoded"`, expected: true},
		{name: "bare yaml literal", extension: ".yaml", body: "password: hardcoded\n", expected: true},
		{name: "environment reference", extension: ".yaml", body: "password: ${NETLAB_PASSWORD}\n"},
		{name: "OpenAPI schema", extension: ".yaml", body: "password: {type: string}\n"},
		{name: "dynamic Go value", extension: ".go", body: `Password: user.Password`},
		{name: "TypeScript type", extension: ".ts", body: `password: string`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if actual := containsAssignedCredential(test.extension, []byte(test.body)); actual != test.expected {
				t.Fatalf("containsAssignedCredential()=%v, expected %v", actual, test.expected)
			}
		})
	}
}

var credentialLiteralPattern = regexp.MustCompile(`(?i)\b(password|passwd|secret|token)\b\s*(?::=|:|=)\s*["'][^"'\r\n]+["']`)
var bareCredentialConfigPattern = regexp.MustCompile(`(?im)^\s*(password|passwd|secret|token)\s*[:=]\s*([^\s#]+)`)

func containsAssignedCredential(extension string, body []byte) bool {
	if credentialLiteralPattern.Find(body) != nil {
		return true
	}
	if extension != ".yaml" && extension != ".yml" && extension != ".sh" && extension != ".service" {
		return false
	}
	for _, match := range bareCredentialConfigPattern.FindAllSubmatch(body, -1) {
		value := strings.TrimSpace(string(match[2]))
		if !strings.HasPrefix(value, "$") && !strings.HasPrefix(value, "{") && !strings.HasPrefix(value, "[") {
			return true
		}
	}
	return false
}
