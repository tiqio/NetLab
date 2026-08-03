package compliance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func CaptureCandidate(version, candidateID, binaryPath, contractsDirectory string, builtAt time.Time) (domain.ReleaseIdentity, error) {
	contractDigest, err := DigestTree(contractsDirectory)
	if err != nil {
		return domain.ReleaseIdentity{}, err
	}
	identity := domain.ReleaseIdentity{Version: version, CandidateID: candidateID, ContractDigest: contractDigest, BuiltAt: &builtAt}
	if binaryPath != "" {
		identity.BinaryDigest, err = DigestFile(binaryPath)
		if err != nil {
			return domain.ReleaseIdentity{}, err
		}
	}
	return identity, identity.Validate()
}

func DigestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

func DigestTree(root string) (string, error) {
	paths := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		relative, _ := filepath.Rel(root, path)
		_, _ = io.WriteString(hash, filepath.ToSlash(relative)+"\n")
		body, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		_, _ = hash.Write(body)
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no contract files under %s", strings.TrimSpace(root))
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
