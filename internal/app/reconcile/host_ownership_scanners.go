package reconcile

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type RuntimeOwnershipSource interface {
	DiscoverRuntimeOwnership(context.Context) ([]ownership.Record, error)
}

type RuntimeOwnershipSourceScanner struct {
	name   string
	source RuntimeOwnershipSource
}

func NewRuntimeOwnershipSourceScanner(name string, source RuntimeOwnershipSource) *RuntimeOwnershipSourceScanner {
	return &RuntimeOwnershipSourceScanner{name: name, source: source}
}

func (s *RuntimeOwnershipSourceScanner) Name() string { return s.name }
func (s *RuntimeOwnershipSourceScanner) Discover(ctx context.Context) ([]DiscoveredOwnership, error) {
	if s.source == nil {
		return nil, nil
	}
	records, err := s.source.DiscoverRuntimeOwnership(ctx)
	if err != nil {
		return nil, err
	}
	values := make([]DiscoveredOwnership, 0, len(records))
	for _, record := range records {
		values = append(values, DiscoveredOwnership{ResourceType: record.ResourceType, ResourceID: record.ResourceID, ObjectKind: record.ObjectKind, ObjectName: record.ObjectName, Metadata: record.Metadata})
	}
	return values, nil
}

type commandOutput func(context.Context, string, ...string) ([]byte, error)

func systemCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type LinuxOwnershipScanner struct {
	ip     string
	nft    string
	output commandOutput
}

func NewLinuxOwnershipScanner() *LinuxOwnershipScanner {
	ip, _ := exec.LookPath("ip")
	nft, _ := exec.LookPath("nft")
	return &LinuxOwnershipScanner{ip: ip, nft: nft, output: systemCommandOutput}
}

func (s *LinuxOwnershipScanner) Name() string { return "linux-network" }
func (s *LinuxOwnershipScanner) Discover(ctx context.Context) ([]DiscoveredOwnership, error) {
	var values []DiscoveredOwnership
	var failures []error
	if s.ip != "" {
		links, err := s.discoverLinks(ctx)
		if err != nil {
			failures = append(failures, err)
		} else {
			values = append(values, links...)
		}
		namespaces, err := s.discoverNamespaces(ctx)
		if err != nil {
			failures = append(failures, err)
		} else {
			values = append(values, namespaces...)
		}
	}
	if s.nft != "" {
		rules, err := s.discoverNFTRules(ctx)
		if err != nil {
			failures = append(failures, err)
		} else {
			values = append(values, rules...)
		}
	}
	if len(values) > 0 {
		return values, nil
	}
	return nil, errors.Join(failures...)
}

func (s *LinuxOwnershipScanner) discoverLinks(ctx context.Context) ([]DiscoveredOwnership, error) {
	body, err := s.output(ctx, s.ip, "-j", "-d", "link", "show")
	if err != nil {
		return nil, err
	}
	var links []struct {
		IfName   string `json:"ifname"`
		IfAlias  string `json:"ifalias"`
		LinkInfo struct {
			InfoKind string `json:"info_kind"`
		} `json:"linkinfo"`
	}
	if err = json.Unmarshal(body, &links); err != nil {
		return nil, err
	}
	values := make([]DiscoveredOwnership, 0, len(links))
	for _, link := range links {
		if !strings.HasPrefix(link.IfAlias, "netlab:") {
			continue
		}
		ownerID := domain.ID(strings.TrimPrefix(link.IfAlias, "netlab:"))
		if ownerID == "" {
			continue
		}
		resourceType := resourceTypeForLinuxName(link.IfName)
		values = append(values, DiscoveredOwnership{ResourceType: resourceType, ResourceID: ownerID, ObjectKind: "linux_link", ObjectName: link.IfName, Metadata: map[string]string{"alias": link.IfAlias, "link_kind": link.LinkInfo.InfoKind}})
	}
	return values, nil
}

func (s *LinuxOwnershipScanner) discoverNamespaces(ctx context.Context) ([]DiscoveredOwnership, error) {
	body, err := s.output(ctx, s.ip, "netns", "list")
	if err != nil {
		return nil, err
	}
	var values []DiscoveredOwnership
	for _, line := range strings.Split(string(body), "\n") {
		name := strings.Fields(line)
		if len(name) == 0 || !hasOwnedPrefix(name[0], "nlpc", "nlsw", "nlr", "n2sw", "n2r") {
			continue
		}
		values = append(values, DiscoveredOwnership{ObjectKind: "network_namespace", ObjectName: name[0], Metadata: map[string]string{"ownership_evidence": "reserved_namespace_prefix"}})
	}
	return values, nil
}

func (s *LinuxOwnershipScanner) discoverNFTRules(ctx context.Context) ([]DiscoveredOwnership, error) {
	body, err := s.output(ctx, s.nft, "-a", "list", "table", "inet", "netlab_nat")
	if err != nil {
		if strings.Contains(string(body), "No such file") {
			return nil, nil
		}
		return nil, err
	}
	var values []DiscoveredOwnership
	for _, line := range strings.Split(string(body), "\n") {
		marker := `comment "netlab:`
		start := strings.Index(line, marker)
		if start < 0 {
			continue
		}
		owner := line[start+len(marker):]
		end := strings.Index(owner, `"`)
		if end <= 0 {
			continue
		}
		ownerID := domain.ID(owner[:end])
		handle := ""
		if fields := strings.Fields(line); len(fields) >= 2 {
			for index := range fields {
				if fields[index] == "handle" && index+1 < len(fields) {
					handle = fields[index+1]
				}
			}
		}
		name := string(ownerID)
		if handle != "" {
			name += ":" + handle
		}
		values = append(values, DiscoveredOwnership{ResourceType: "network_object", ResourceID: ownerID, ObjectKind: "nft_rule", ObjectName: name, Metadata: map[string]string{"table": "inet netlab_nat", "handle": handle}})
	}
	return values, nil
}

func resourceTypeForLinuxName(name string) string {
	switch {
	case hasOwnedPrefix(name, "nli", "nlp", "nlt"):
		return "interface"
	case hasOwnedPrefix(name, "nll"):
		return "link"
	case hasOwnedPrefix(name, "nlbr", "nlnat"):
		return "network_object"
	case hasOwnedPrefix(name, "nla", "nah", "nap"):
		return "network_attachment"
	default:
		return "unknown"
	}
}

func hasOwnedPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

type OwnedDirectoryScanner struct {
	name, root, objectKind, resourceType string
}

func NewOwnedDirectoryScanner(name, root, objectKind, resourceType string) *OwnedDirectoryScanner {
	return &OwnedDirectoryScanner{name: name, root: root, objectKind: objectKind, resourceType: resourceType}
}

func (s *OwnedDirectoryScanner) Name() string { return s.name }
func (s *OwnedDirectoryScanner) Discover(context.Context) ([]DiscoveredOwnership, error) {
	entries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var values []DiscoveredOwnership
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "quarantine" {
			continue
		}
		values = append(values, DiscoveredOwnership{ResourceType: s.resourceType, ResourceID: domain.ID(entry.Name()), ObjectKind: s.objectKind, ObjectName: entry.Name(), Metadata: map[string]string{"path": filepath.Join(s.root, entry.Name())}})
	}
	return values, nil
}

type CaptureStateScanner struct{ path string }

func NewCaptureStateScanner(stateDir string) *CaptureStateScanner {
	return &CaptureStateScanner{path: filepath.Join(stateDir, "captures", "index.json")}
}
func (s *CaptureStateScanner) Name() string { return "capture-state" }
func (s *CaptureStateScanner) Discover(context.Context) ([]DiscoveredOwnership, error) {
	body, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var records []struct {
		Metadata struct {
			ID    domain.ID `json:"id"`
			State string    `json:"state"`
		} `json:"metadata"`
		Path string `json:"path"`
	}
	if err = json.Unmarshal(body, &records); err != nil {
		return nil, err
	}
	var values []DiscoveredOwnership
	for _, record := range records {
		if record.Metadata.ID == "" {
			continue
		}
		values = append(values, DiscoveredOwnership{ResourceType: "capture", ResourceID: record.Metadata.ID, ObjectKind: "capture_record", ObjectName: string(record.Metadata.ID), Metadata: map[string]string{"state": record.Metadata.State, "path": record.Path}})
	}
	return values, nil
}

type OwnedProcessScanner struct{ procRoot, stateDir string }

func NewOwnedProcessScanner(stateDir string) *OwnedProcessScanner {
	return &OwnedProcessScanner{procRoot: "/proc", stateDir: stateDir}
}
func (s *OwnedProcessScanner) Name() string { return "owned-processes" }
func (s *OwnedProcessScanner) Discover(context.Context) ([]DiscoveredOwnership, error) {
	entries, err := os.ReadDir(s.procRoot)
	if err != nil {
		return nil, err
	}
	var values []DiscoveredOwnership
	for _, entry := range entries {
		if _, err = strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(s.procRoot, entry.Name(), "cmdline"))
		if readErr != nil {
			continue
		}
		cmdline := strings.ReplaceAll(string(body), "\x00", " ")
		environment, _ := os.ReadFile(filepath.Join(s.procRoot, entry.Name(), "environ"))
		ownerType, ownerID := processOwner(environment)
		kind := ""
		switch {
		case strings.Contains(cmdline, "netlab-console-proxy"):
			kind = "console_proxy_process"
		case strings.Contains(cmdline, "netlab-helper"):
			kind = "helper_process"
		case ownerType == "capture" && (strings.Contains(cmdline, "dumpcap") || strings.Contains(cmdline, "tcpdump")):
			kind = "capture_worker_process"
		case ownerType == "node" && strings.Contains(cmdline, "qemu-system"):
			kind = "qemu_process"
		case ownerType != "" && strings.Contains(cmdline, "dhclient"):
			kind = "helper_process"
		}
		if kind != "" {
			values = append(values, DiscoveredOwnership{ResourceType: ownerType, ResourceID: ownerID, ObjectKind: kind, ObjectName: entry.Name(), Metadata: map[string]string{"cmdline": cmdline}})
		}
	}
	return values, nil
}

func processOwner(environment []byte) (string, domain.ID) {
	for _, value := range strings.Split(string(environment), "\x00") {
		if !strings.HasPrefix(value, "NETLAB_OWNERSHIP=") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(value, "NETLAB_OWNERSHIP="), ":", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], domain.ID(parts[1])
		}
	}
	return "", ""
}
