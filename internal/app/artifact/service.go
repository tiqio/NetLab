package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

type Repository interface {
	CreateArtifact(context.Context, domain.Artifact) error
	GetArtifact(context.Context, domain.ID) (domain.Artifact, error)
}

type Service struct {
	repository Repository
	directory  string
}

func NewService(repository Repository, stateDir string) *Service {
	return &Service{repository: repository, directory: filepath.Join(stateDir, "artifacts")}
}

func (s *Service) Create(ctx context.Context, kind, mediaType, ownerType string, ownerID domain.ID, body []byte, ttl time.Duration) (domain.Artifact, error) {
	return s.CreateWithID(ctx, domain.NewID(), kind, mediaType, ownerType, ownerID, body, ttl)
}

func (s *Service) CreateWithID(ctx context.Context, id domain.ID, kind, mediaType, ownerType string, ownerID domain.ID, body []byte, ttl time.Duration) (domain.Artifact, error) {
	if existing, err := s.repository.GetArtifact(ctx, id); err == nil {
		return existing, nil
	}
	if err := os.MkdirAll(s.directory, 0o700); err != nil {
		return domain.Artifact{}, err
	}
	path := filepath.Join(s.directory, string(id))
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, body, 0o600); err != nil {
		return domain.Artifact{}, err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return domain.Artifact{}, err
	}
	sum := sha256.Sum256(body)
	now := time.Now().UTC()
	artifact := domain.Artifact{ID: id, Kind: kind, Path: path, MediaType: mediaType, SizeBytes: int64(len(body)), SHA256: hex.EncodeToString(sum[:]), OwnerType: ownerType, OwnerID: ownerID, CreatedAt: now}
	if ttl > 0 {
		expiresAt := now.Add(ttl)
		artifact.ExpiresAt = &expiresAt
	}
	if err := s.repository.CreateArtifact(ctx, artifact); err != nil {
		_ = os.Remove(path)
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func (s *Service) Open(ctx context.Context, id domain.ID) (domain.Artifact, *os.File, error) {
	artifact, err := s.repository.GetArtifact(ctx, id)
	if err != nil {
		return artifact, nil, err
	}
	if artifact.ExpiresAt != nil && !artifact.ExpiresAt.After(time.Now().UTC()) {
		return artifact, nil, fmt.Errorf("artifact expired")
	}
	file, err := os.Open(artifact.Path)
	return artifact, file, err
}
