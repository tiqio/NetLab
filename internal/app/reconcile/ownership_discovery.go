package reconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
	"github.com/netlab/netlab/internal/runtime/ownership"
)

type DiscoveredOwnership struct {
	ResourceType   string
	ResourceID     domain.ID
	ObjectKind     string
	ObjectName     string
	Metadata       map[string]string
	QuarantinePath string
	QuarantineRoot string
	CleanupSafe    bool
	Cleanup        func(context.Context) error
}

type OwnershipScanner interface {
	Name() string
	Discover(context.Context) ([]DiscoveredOwnership, error)
}

type OwnershipDiscoveryStore interface {
	ListRuntimeOwnership(context.Context) ([]ownership.Record, error)
	UpsertRuntimeOwnership(context.Context, string, domain.ID, string, string, map[string]string, string) error
	DeleteRuntimeOwnership(context.Context, string, domain.ID, string, string) error
	RuntimeOwnerExists(context.Context, string, domain.ID) (bool, error)
}

type OwnershipAuditRecorder interface {
	Record(context.Context, string, string, string, domain.ID, domain.ID, string, string, map[string]any) (domain.AuditEvent, error)
}

type OwnershipDiscoveryReconciler struct {
	store          OwnershipDiscoveryStore
	audit          OwnershipAuditRecorder
	scanners       []OwnershipScanner
	scannerTimeout time.Duration
}

func NewOwnershipDiscoveryReconciler(store OwnershipDiscoveryStore, audit OwnershipAuditRecorder, scanners ...OwnershipScanner) *OwnershipDiscoveryReconciler {
	return &OwnershipDiscoveryReconciler{store: store, audit: audit, scanners: scanners, scannerTimeout: 5 * time.Second}
}

func (r *OwnershipDiscoveryReconciler) Name() string { return "ownership-discovery" }

func (r *OwnershipDiscoveryReconciler) ReconcileWithCheckpoints(ctx context.Context, checkpoint func(RecoveryResourceOutcome) error) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("reconciler", domain.ID(r.Name()), "ownership_discovery"))
	if err := r.Reconcile(ctx); err != nil {
		return err
	}
	records, err := r.store.ListRuntimeOwnership(ctx)
	if err != nil {
		return err
	}
	for _, record := range records {
		state := "recovered"
		if record.CleanupState != "active" {
			state = "validation_required"
		}
		if record.CleanupState == "missing_validation_required" && (record.ObjectKind == "console_proxy" || record.ObjectKind == "capture_worker_process" || record.ObjectKind == "helper_process") {
			state = "reconnect_required"
		}
		details := cloneMetadata(record.Metadata)
		details["object_kind"] = record.ObjectKind
		details["cleanup_state"] = record.CleanupState
		if err = checkpoint(RecoveryResourceOutcome{ResourceType: record.ResourceType, ResourceID: record.ResourceID, RuntimeID: record.ObjectName, State: state, Details: details}); err != nil {
			return err
		}
	}
	return nil
}

func (r *OwnershipDiscoveryReconciler) Reconcile(ctx context.Context) (err error) {
	defer normalizeTerminalError(&err, terminalProblem("reconciler", domain.ID(r.Name()), "ownership_reconcile"))
	known, err := r.store.ListRuntimeOwnership(ctx)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	knownByObject := map[string]ownership.Record{}
	for _, value := range known {
		knownByObject[ownershipKey(value.ObjectKind, value.ObjectName)] = value
	}
	for _, scanner := range r.scanners {
		scanCtx, cancel := context.WithTimeout(ctx, r.scannerTimeout)
		values, scanErr := scanner.Discover(scanCtx)
		cancel()
		if scanErr != nil {
			r.record(ctx, "ownership.discovery.failed", "host", domain.ID(scanner.Name()), "failed", map[string]any{"scanner": scanner.Name(), "error": scanErr.Error()})
			continue
		}
		for _, value := range values {
			key := ownershipKey(value.ObjectKind, value.ObjectName)
			seen[key] = true
			record, registered := knownByObject[key]
			registeredOwned := registered && record.ResourceType != "unknown"
			ownerExists := false
			if value.ResourceType != "" && value.ResourceID != "" {
				ownerExists, err = r.store.RuntimeOwnerExists(ctx, value.ResourceType, value.ResourceID)
				if err != nil {
					return err
				}
			}
			if registeredOwned || ownerExists {
				if registeredOwned {
					value.ResourceType, value.ResourceID = record.ResourceType, record.ResourceID
				}
				if err = r.store.UpsertRuntimeOwnership(ctx, value.ResourceType, value.ResourceID, value.ObjectKind, value.ObjectName, withDiscoveryMetadata(value.Metadata, scanner.Name()), "active"); err != nil {
					return err
				}
				continue
			}
			if registered && record.ResourceType == "unknown" && record.CleanupState == "unknown_observed" && value.CleanupSafe && value.Cleanup != nil {
				if cleanupErr := value.Cleanup(ctx); cleanupErr == nil {
					if err = r.store.DeleteRuntimeOwnership(ctx, record.ResourceType, record.ResourceID, record.ObjectKind, record.ObjectName); err != nil {
						return err
					}
					r.record(ctx, "ownership.resource.cleaned", record.ResourceType, record.ResourceID, "cleaned", map[string]any{"scanner": scanner.Name(), "object_kind": value.ObjectKind, "object_name": value.ObjectName})
					continue
				} else {
					metadata := withDiscoveryMetadata(value.Metadata, scanner.Name())
					metadata["cleanup_error"] = cleanupErr.Error()
					if err = r.store.UpsertRuntimeOwnership(ctx, record.ResourceType, record.ResourceID, value.ObjectKind, value.ObjectName, metadata, "cleanup_failed"); err != nil {
						return err
					}
					continue
				}
			}
			if registered && record.ResourceType == "unknown" {
				if err = r.store.UpsertRuntimeOwnership(ctx, record.ResourceType, record.ResourceID, value.ObjectKind, value.ObjectName, withDiscoveryMetadata(value.Metadata, scanner.Name()), record.CleanupState); err != nil {
					return err
				}
				continue
			}
			quarantinedPath, quarantineErr := quarantineDiscovered(value)
			state := "unknown_observed"
			action := "ownership.resource.discovered"
			if value.QuarantinePath != "" {
				state = "quarantined"
				action = "ownership.resource.quarantined"
				if quarantineErr != nil {
					state = "quarantine_failed"
				}
			}
			unknownID := value.ResourceID
			if unknownID == "" {
				unknownID = domain.ID(value.ObjectName)
			}
			metadata := withDiscoveryMetadata(value.Metadata, scanner.Name())
			metadata["ownership_class"] = ownership.ClassForeignObserved
			if quarantinedPath != "" {
				metadata["quarantine_path"] = quarantinedPath
			}
			if quarantineErr != nil {
				metadata["quarantine_error"] = quarantineErr.Error()
			}
			if err = r.store.UpsertRuntimeOwnership(ctx, "unknown", unknownID, value.ObjectKind, value.ObjectName, metadata, state); err != nil {
				return err
			}
			r.record(ctx, action, "unknown", unknownID, state, map[string]any{"scanner": scanner.Name(), "object_kind": value.ObjectKind, "object_name": value.ObjectName, "metadata": metadata})
		}
	}
	for _, value := range known {
		if value.ResourceType == "unknown" && !seen[ownershipKey(value.ObjectKind, value.ObjectName)] {
			if err = r.store.DeleteRuntimeOwnership(ctx, value.ResourceType, value.ResourceID, value.ObjectKind, value.ObjectName); err != nil {
				return err
			}
			r.record(ctx, "ownership.resource.resolved", value.ResourceType, value.ResourceID, "resolved", map[string]any{"object_kind": value.ObjectKind, "object_name": value.ObjectName})
			continue
		}
		if value.CleanupState == "active" && !seen[ownershipKey(value.ObjectKind, value.ObjectName)] {
			metadata := cloneMetadata(value.Metadata)
			metadata["missing_since"] = time.Now().UTC().Format(time.RFC3339Nano)
			if err = r.store.UpsertRuntimeOwnership(ctx, value.ResourceType, value.ResourceID, value.ObjectKind, value.ObjectName, metadata, "missing_validation_required"); err != nil {
				return err
			}
			r.record(ctx, "ownership.resource.missing", value.ResourceType, value.ResourceID, "observed", map[string]any{"object_kind": value.ObjectKind, "object_name": value.ObjectName, "cleanup": "not removed; abandonment validation required"})
		}
	}
	return nil
}

func cloneMetadata(input map[string]string) map[string]string {
	result := make(map[string]string, len(input)+1)
	for key, value := range input {
		result[key] = value
	}
	return result
}

func ownershipKey(kind, name string) string { return kind + "\x00" + name }

func withDiscoveryMetadata(input map[string]string, scanner string) map[string]string {
	result := map[string]string{"scanner": scanner, "last_seen": time.Now().UTC().Format(time.RFC3339Nano)}
	for key, value := range input {
		result[key] = value
	}
	return result
}

func (r *OwnershipDiscoveryReconciler) record(ctx context.Context, action, resourceType string, resourceID domain.ID, outcome string, details map[string]any) {
	if r.audit != nil {
		_, _ = r.audit.Record(ctx, "reconciler", action, resourceType, resourceID, "", outcome, string(resourceID), details)
	}
}

func quarantineDiscovered(value DiscoveredOwnership) (string, error) {
	if value.QuarantinePath == "" {
		return "", nil
	}
	root, err := filepath.Abs(value.QuarantineRoot)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(value.QuarantinePath)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || relative == ".." {
		return "", fmt.Errorf("refusing to quarantine path outside owned root")
	}
	directory := filepath.Join(root, "quarantine")
	if err = os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	destination := filepath.Join(directory, filepath.Base(path)+"-"+strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
	if err = os.Rename(path, destination); err != nil {
		return "", err
	}
	return destination, nil
}

type QEMUOwnershipScanner struct{ root string }

func NewQEMUOwnershipScanner(stateDir string) *QEMUOwnershipScanner {
	return &QEMUOwnershipScanner{root: filepath.Join(stateDir, "runtime", "qemu")}
}

func (s *QEMUOwnershipScanner) Name() string { return "qemu-runtime" }

func (s *QEMUOwnershipScanner) Discover(_ context.Context) ([]DiscoveredOwnership, error) {
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
		directory := filepath.Join(s.root, entry.Name())
		value := DiscoveredOwnership{ObjectKind: "qemu_runtime_dir", ObjectName: entry.Name(), Metadata: map[string]string{"path": directory}, QuarantinePath: directory, QuarantineRoot: s.root}
		body, readErr := os.ReadFile(filepath.Join(directory, "launch.json"))
		if os.IsNotExist(readErr) {
			body, readErr = os.ReadFile(filepath.Join(directory, "manifest.json"))
		}
		if readErr == nil {
			var manifest struct {
				NodeID domain.ID `json:"node_id"`
				PID    int       `json:"pid"`
				QMP    string    `json:"qmp"`
				QGA    string    `json:"qga"`
				Serial string    `json:"serial"`
			}
			if json.Unmarshal(body, &manifest) == nil && manifest.NodeID != "" {
				value.ResourceType = "node"
				value.ResourceID = manifest.NodeID
				value.Metadata["pid"] = strconv.Itoa(manifest.PID)
				values = append(values, value)
				if qemuProcessMatches(directory, manifest.PID) {
					values = append(values, DiscoveredOwnership{ResourceType: "node", ResourceID: manifest.NodeID, ObjectKind: "qemu_process", ObjectName: strconv.Itoa(manifest.PID), Metadata: map[string]string{"runtime_dir": directory}})
				}
				for kind, path := range map[string]string{"qmp_socket": manifest.QMP, "qga_socket": manifest.QGA, "serial_socket": manifest.Serial} {
					if ownedSocketExists(directory, path) {
						values = append(values, DiscoveredOwnership{ResourceType: "node", ResourceID: manifest.NodeID, ObjectKind: kind, ObjectName: path, Metadata: map[string]string{"runtime_dir": directory}})
					}
				}
				continue
			}
		}
		values = append(values, value)
	}
	return values, nil
}

func qemuProcessMatches(runtimeDir string, pid int) bool {
	if pid <= 0 {
		return false
	}
	body, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	return err == nil && strings.Contains(strings.ReplaceAll(string(body), "\x00", " "), runtimeDir)
}

func ownedSocketExists(runtimeDir, path string) bool {
	if path == "" {
		return false
	}
	absoluteRoot, err := filepath.Abs(runtimeDir)
	if err != nil {
		return false
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return false
	}
	info, err := os.Stat(absolutePath)
	return err == nil && info.Mode()&os.ModeSocket != 0
}
