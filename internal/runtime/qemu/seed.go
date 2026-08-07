package qemu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/netlab/netlab/internal/domain"
)

type SeedSpec struct {
	UserData      string
	MetaData      string
	NetworkConfig string
	VendorData    string
}

type SeedManager struct {
	Root    string
	Xorriso string
}

type BootstrapCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type PreparedSeedUpdate struct {
	temporaryPath string
	targetPath    string
	directory     string
}

func (u *PreparedSeedUpdate) Commit() error {
	if err := os.Rename(u.temporaryPath, u.targetPath); err != nil {
		return err
	}
	return os.RemoveAll(u.directory)
}

func (u *PreparedSeedUpdate) Cleanup() { _ = os.RemoveAll(u.directory) }

func NewSeedManager(root string) (*SeedManager, error) {
	path, err := exec.LookPath("xorriso")
	if err != nil {
		return nil, err
	}
	return &SeedManager{Root: filepath.Join(root, "bootstrap"), Xorriso: path}, nil
}

func (m *SeedManager) Build(ctx context.Context, laboratoryID, nodeID domain.ID, spec SeedSpec) (string, error) {
	if spec.UserData == "" {
		return "", fmt.Errorf("cloud-init user-data is required")
	}
	directory := filepath.Join(m.Root, string(laboratoryID), string(nodeID))
	return m.buildInDirectory(ctx, directory, nodeID, spec)
}

func (m *SeedManager) buildInDirectory(ctx context.Context, directory string, nodeID domain.ID, spec SeedSpec) (string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	files := map[string]string{"user-data": spec.UserData, "meta-data": spec.MetaData}
	if files["meta-data"] == "" {
		files["meta-data"] = "instance-id: " + string(nodeID) + "\nlocal-hostname: netlab-" + string(nodeID) + "\n"
	}
	if spec.NetworkConfig != "" {
		files["network-config"] = spec.NetworkConfig
	}
	if spec.VendorData != "" {
		files["vendor-data"] = spec.VendorData
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0o600); err != nil {
			cleanup()
			return "", err
		}
	}
	isoPath := filepath.Join(directory, "seed.iso")
	args := []string{"-as", "mkisofs", "-volid", "cidata", "-joliet", "-rock", "-output", isoPath, filepath.Join(directory, "user-data"), filepath.Join(directory, "meta-data")}
	for _, name := range []string{"network-config", "vendor-data"} {
		if _, ok := files[name]; ok {
			args = append(args, filepath.Join(directory, name))
		}
	}
	if output, err := exec.CommandContext(ctx, m.Xorriso, args...).CombinedOutput(); err != nil {
		cleanup()
		return "", fmt.Errorf("create cloud-init seed: %s: %w", output, err)
	}
	for name := range files {
		_ = os.Remove(filepath.Join(directory, name))
	}
	if err := os.Chmod(isoPath, 0o600); err != nil {
		cleanup()
		return "", err
	}
	return isoPath, nil
}

func (m *SeedManager) Delete(laboratoryID, nodeID domain.ID) error {
	return os.RemoveAll(filepath.Join(m.Root, string(laboratoryID), string(nodeID)))
}

func (m *SeedManager) Credentials(ctx context.Context, seedPath string) (BootstrapCredentials, error) {
	seedPath, err := m.managedSeedPath(seedPath)
	if err != nil {
		return BootstrapCredentials{}, fmt.Errorf("seed path is outside the managed bootstrap directory")
	}
	directory, err := os.MkdirTemp("", "netlab-seed-read-")
	if err != nil {
		return BootstrapCredentials{}, err
	}
	defer os.RemoveAll(directory)
	userDataPath := filepath.Join(directory, "user-data")
	if output, extractErr := exec.CommandContext(ctx, m.Xorriso, "-osirrox", "on", "-indev", seedPath, "-extract", "/user-data", userDataPath).CombinedOutput(); extractErr != nil {
		return BootstrapCredentials{}, fmt.Errorf("extract cloud-init user-data: %s: %w", output, extractErr)
	}
	userData, err := os.ReadFile(userDataPath)
	if err != nil {
		return BootstrapCredentials{}, err
	}
	payload := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(userData)), "#cloud-config"))
	var cloudConfig struct {
		Chpasswd struct {
			Users []struct {
				Name     string `json:"name"`
				Password string `json:"password"`
				Type     string `json:"type"`
			} `json:"users"`
		} `json:"chpasswd"`
	}
	if err = json.Unmarshal([]byte(payload), &cloudConfig); err != nil {
		return BootstrapCredentials{}, fmt.Errorf("cloud-init credentials are not in the supported format: %w", err)
	}
	for _, user := range cloudConfig.Chpasswd.Users {
		if strings.TrimSpace(user.Name) != "" && user.Password != "" && (user.Type == "" || user.Type == "text") {
			return BootstrapCredentials{user.Name, user.Password}, nil
		}
	}
	return BootstrapCredentials{}, fmt.Errorf("cloud-init seed does not contain recoverable plaintext credentials")
}

func (m *SeedManager) PrepareNetworkConfig(ctx context.Context, seedPath, networkConfig string) (*PreparedSeedUpdate, error) {
	seedPath, err := m.managedSeedPath(seedPath)
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(filepath.Dir(seedPath), ".seed-update-")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	for _, name := range []string{"user-data", "meta-data"} {
		output, extractErr := exec.CommandContext(ctx, m.Xorriso, "-osirrox", "on", "-indev", seedPath, "-extract", "/"+name, filepath.Join(directory, name)).CombinedOutput()
		if extractErr != nil {
			cleanup()
			return nil, fmt.Errorf("extract cloud-init %s: %s: %w", name, output, extractErr)
		}
	}
	vendorDataPath := filepath.Join(directory, "vendor-data")
	if _, extractErr := exec.CommandContext(ctx, m.Xorriso, "-osirrox", "on", "-indev", seedPath, "-extract", "/vendor-data", vendorDataPath).CombinedOutput(); extractErr != nil {
		_ = os.Remove(vendorDataPath)
	}
	metaDataPath := filepath.Join(directory, "meta-data")
	metaData, err := os.ReadFile(metaDataPath)
	if err != nil {
		cleanup()
		return nil, err
	}
	lines := strings.Split(string(metaData), "\n")
	replacedInstanceID := false
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "instance-id:") {
			lines[index] = "instance-id: netlab-settings-" + string(domain.NewID())
			replacedInstanceID = true
			break
		}
	}
	if !replacedInstanceID {
		lines = append([]string{"instance-id: netlab-settings-" + string(domain.NewID())}, lines...)
	}
	if err = os.WriteFile(metaDataPath, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		cleanup()
		return nil, err
	}
	if err = os.WriteFile(filepath.Join(directory, "network-config"), []byte(networkConfig), 0o600); err != nil {
		cleanup()
		return nil, err
	}
	temporaryPath := filepath.Join(directory, "seed.iso")
	args := []string{"-as", "mkisofs", "-volid", "cidata", "-joliet", "-rock", "-output", temporaryPath, filepath.Join(directory, "user-data"), filepath.Join(directory, "meta-data"), filepath.Join(directory, "network-config")}
	if _, statErr := os.Stat(vendorDataPath); statErr == nil {
		args = append(args, vendorDataPath)
	}
	if output, buildErr := exec.CommandContext(ctx, m.Xorriso, args...).CombinedOutput(); buildErr != nil {
		cleanup()
		return nil, fmt.Errorf("rebuild cloud-init seed: %s: %w", output, buildErr)
	}
	if err = os.Chmod(temporaryPath, 0o600); err != nil {
		cleanup()
		return nil, err
	}
	return &PreparedSeedUpdate{temporaryPath: temporaryPath, targetPath: seedPath, directory: directory}, nil
}

func (m *SeedManager) managedSeedPath(seedPath string) (string, error) {
	seedPath = filepath.Clean(seedPath)
	relative, err := filepath.Rel(m.Root, seedPath)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.Base(seedPath) != "seed.iso" {
		return "", fmt.Errorf("seed path is outside the managed bootstrap directory")
	}
	return seedPath, nil
}
