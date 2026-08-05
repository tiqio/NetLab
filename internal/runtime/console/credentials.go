package console

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/netlab/netlab/internal/domain"
	qemuRuntime "github.com/netlab/netlab/internal/runtime/qemu"
)

type SeedCredentialSource interface {
	Credentials(context.Context, string) (qemuRuntime.BootstrapCredentials, error)
}

type CredentialStore struct {
	root  string
	seeds SeedCredentialSource
}

func NewCredentialStore(root string, seeds SeedCredentialSource) *CredentialStore {
	return &CredentialStore{root: filepath.Join(root, "console-credentials", "images"), seeds: seeds}
}

func (s *CredentialStore) PutImage(imageID domain.ID, credentials qemuRuntime.BootstrapCredentials) error {
	if imageID == "" || strings.TrimSpace(credentials.Username) == "" {
		return fmt.Errorf("image ID and console username are required")
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	body, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(s.root, ".credentials-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err = temporary.Chmod(0o600); err == nil {
		_, err = temporary.Write(body)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporaryPath, s.imagePath(imageID))
}

func (s *CredentialStore) CredentialsForNode(ctx context.Context, node domain.Node) (qemuRuntime.BootstrapCredentials, error) {
	if seedPath, _ := node.Config["seed_iso"].(string); seedPath != "" && s.seeds != nil {
		if credentials, err := s.seeds.Credentials(ctx, seedPath); err == nil {
			return credentials, nil
		}
	}
	imageID, _ := node.Config["image_version_id"].(string)
	if imageID == "" {
		return qemuRuntime.BootstrapCredentials{}, fmt.Errorf("node has no managed console credentials")
	}
	body, err := os.ReadFile(s.imagePath(domain.ID(imageID)))
	if err != nil {
		return qemuRuntime.BootstrapCredentials{}, fmt.Errorf("image console credentials are unavailable")
	}
	var credentials qemuRuntime.BootstrapCredentials
	if err = json.Unmarshal(body, &credentials); err != nil || strings.TrimSpace(credentials.Username) == "" {
		return qemuRuntime.BootstrapCredentials{}, fmt.Errorf("image console credentials are invalid")
	}
	return credentials, nil
}

func (s *CredentialStore) imagePath(imageID domain.ID) string {
	return filepath.Join(s.root, string(imageID)+".json")
}
