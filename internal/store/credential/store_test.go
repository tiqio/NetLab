package credential

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/netlab/netlab/internal/domain"
)

func TestStoreEncryptsAndScopesNodeCredentials(t *testing.T) {
	root := t.TempDir()
	keyPath := filepath.Join(root, "master.key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(root, "credentials.db"), keyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	password := []byte("FortiGate-Secret-123")
	metadata, err := store.Put(context.Background(), "lab-a", "node-a", domain.CredentialKindConsoleAdmin, domain.CredentialSlotActive, "admin", password)
	if err != nil || !metadata.Configured || metadata.State != "pending_verification" {
		t.Fatalf("metadata=%+v err=%v", metadata, err)
	}
	secret, err := store.Get(context.Background(), "lab-a", "node-a", domain.CredentialKindConsoleAdmin, domain.CredentialSlotActive)
	if err != nil || secret.Username != "admin" || string(secret.Password) != string(password) {
		t.Fatalf("secret mismatch: username=%q err=%v", secret.Username, err)
	}
	secret.Clear()
	if _, err = store.Get(context.Background(), "lab-a", "node-b", domain.CredentialKindConsoleAdmin, domain.CredentialSlotActive); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-node read err=%v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "credentials.db"))
	if err != nil {
		t.Fatal(err)
	}
	if stringContains(body, []byte("FortiGate-Secret-123")) || stringContains(body, []byte("admin")) {
		t.Fatal("credential database contains plaintext material")
	}
}

func TestOpenRequiresMasterKey(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "credentials.db"), "")
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("err=%v", err)
	}
}

func TestPromoteStagedReencryptsForActiveSlot(t *testing.T) {
	root := t.TempDir()
	keyPath := filepath.Join(root, "master.key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901"))), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(filepath.Join(root, "credentials.db"), keyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if _, err = store.Put(ctx, "lab-a", "node-a", domain.CredentialKindConsoleAdmin, domain.CredentialSlotActive, "admin", []byte("old-secret")); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Put(ctx, "lab-a", "node-a", domain.CredentialKindConsoleAdmin, domain.CredentialSlotStaged, "admin", []byte("new-secret")); err != nil {
		t.Fatal(err)
	}
	if err = store.PromoteStaged(ctx, "lab-a", "node-a", domain.CredentialKindConsoleAdmin); err != nil {
		t.Fatal(err)
	}
	secret, err := store.Get(ctx, "lab-a", "node-a", domain.CredentialKindConsoleAdmin, domain.CredentialSlotActive)
	if err != nil {
		t.Fatal(err)
	}
	defer secret.Clear()
	if string(secret.Password) != "new-secret" {
		t.Fatalf("password=%q", secret.Password)
	}
	if _, err = store.Get(ctx, "lab-a", "node-a", domain.CredentialKindConsoleAdmin, domain.CredentialSlotStaged); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("staged credential still exists: %v", err)
	}
}

func stringContains(body, needle []byte) bool {
	if len(needle) == 0 || len(body) < len(needle) {
		return false
	}
	for index := 0; index <= len(body)-len(needle); index++ {
		match := true
		for offset := range needle {
			if body[index+offset] != needle[offset] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
