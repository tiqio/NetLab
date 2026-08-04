package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen                string `yaml:"listen"`
	StateDir              string `yaml:"state_dir"`
	DatabasePath          string `yaml:"database_path"`
	RuntimeDir            string `yaml:"runtime_dir"`
	TemplateDir           string `yaml:"template_dir"`
	TemplateReadinessPath string `yaml:"template_readiness_path"`
	Release               struct {
		Version        string `yaml:"version"`
		CandidateID    string `yaml:"candidate_id"`
		BinaryDigest   string `yaml:"binary_digest"`
		ContractDigest string `yaml:"contract_digest"`
		BuiltAt        string `yaml:"built_at"`
	} `yaml:"release"`
	Deployment struct {
		Role             string   `yaml:"role"`
		ManagementScopes []string `yaml:"management_scopes"`
	} `yaml:"deployment"`
	StartupConcurrency struct {
		QEMU  int `yaml:"qemu"`
		Other int `yaml:"other"`
	} `yaml:"startup_concurrency"`
	Captures struct {
		Concurrent     int           `yaml:"concurrent"`
		Duration       time.Duration `yaml:"-"`
		DurationText   string        `yaml:"duration"`
		MaxBytes       int64         `yaml:"max_bytes"`
		Retention      time.Duration `yaml:"-"`
		RetentionText  string        `yaml:"retention"`
		GlobalMaxBytes int64         `yaml:"global_max_bytes"`
	} `yaml:"captures"`
}

func Defaults() Config {
	var c Config
	c.Listen = "0.0.0.0:8080"
	c.StateDir = "/var/lib/netlab"
	c.DatabasePath = "/var/lib/netlab/netlab.db"
	c.RuntimeDir = "/run/netlab"
	c.TemplateDir = "/usr/local/share/netlab/templates"
	c.TemplateReadinessPath = "/etc/netlab/template-readiness.json"
	c.Release.Version = "dev"
	c.Release.CandidateID = "dev"
	c.Release.ContractDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	c.Deployment.Role = "development"
	c.StartupConcurrency.QEMU = 2
	c.StartupConcurrency.Other = 4
	c.Captures.Concurrent = 16
	c.Captures.Duration = 15 * time.Minute
	c.Captures.MaxBytes = 256 << 20
	c.Captures.Retention = 24 * time.Hour
	c.Captures.GlobalMaxBytes = 10 << 30
	return c
}

func Load(path string) (Config, error) {
	c, err := LoadRaw(path)
	if err != nil {
		return c, err
	}
	return c, c.Validate()
}

func LoadRaw(path string) (Config, error) {
	c := Defaults()
	if path == "" {
		return c, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err = yaml.Unmarshal(body, &c); err != nil {
		return c, err
	}
	if role := strings.TrimSpace(os.Getenv("NETLAB_DEPLOYMENT_ROLE")); role != "" {
		c.Deployment.Role = role
	}
	if c.Captures.DurationText != "" {
		c.Captures.Duration, err = time.ParseDuration(c.Captures.DurationText)
		if err != nil {
			return c, err
		}
	}
	if c.Captures.RetentionText != "" {
		c.Captures.Retention, err = time.ParseDuration(c.Captures.RetentionText)
		if err != nil {
			return c, err
		}
	}
	return c, nil
}

func (c Config) Validate() error {
	host, port, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return err
	}
	if port == "" {
		return errors.New("listen port required")
	}
	_ = host
	if c.StateDir == "" || c.DatabasePath == "" || c.RuntimeDir == "" || c.TemplateDir == "" {
		return errors.New("state, database, runtime, and template paths required")
	}
	if c.Release.Version == "" || c.Release.CandidateID == "" {
		return errors.New("release version and candidate id required")
	}
	switch c.Deployment.Role {
	case "development", "validation", "authoritative":
	default:
		return fmt.Errorf("invalid deployment role %q", c.Deployment.Role)
	}
	if c.Deployment.Role == "authoritative" {
		if isPlaceholderReleaseValue(c.Release.Version) || isPlaceholderReleaseValue(c.Release.CandidateID) || !validNonZeroSHA256(c.Release.BinaryDigest) || !validNonZeroSHA256(c.Release.ContractDigest) || c.Release.BuiltAt == "" {
			return errors.New("authoritative release requires immutable non-placeholder version, candidate, binary digest, contract digest, and build time")
		}
		if _, err = time.Parse(time.RFC3339, c.Release.BuiltAt); err != nil {
			return fmt.Errorf("authoritative release build time: %w", err)
		}
	}
	if c.StartupConcurrency.QEMU < 1 || c.StartupConcurrency.Other < 1 {
		return errors.New("startup concurrency must be positive")
	}
	if c.Captures.Concurrent < 1 || c.Captures.MaxBytes < 1 || c.Captures.GlobalMaxBytes < c.Captures.MaxBytes {
		return errors.New("invalid capture limits")
	}
	return nil
}

func isPlaceholderReleaseValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "" || value == "dev" || value == "development" || value == "unbuilt" || strings.HasPrefix(value, "operator-supplied")
}

func validNonZeroSHA256(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	digits := strings.TrimPrefix(value, "sha256:")
	if strings.Trim(digits, "0") == "" {
		return false
	}
	for _, digit := range digits {
		if !strings.ContainsRune("0123456789abcdef", digit) {
			return false
		}
	}
	return true
}

func (c Config) SecurityWarning() string {
	host, _, _ := net.SplitHostPort(c.Listen)
	if host == "" || host == "0.0.0.0" || host == "::" {
		return fmt.Sprintf("SECURITY WARNING: unauthenticated management API listens on all interfaces at %s; restrict access with a host firewall or trusted network", c.Listen)
	}
	return ""
}
