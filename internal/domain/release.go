package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type ReleaseIdentity struct {
	Version        string     `json:"version"`
	CandidateID    string     `json:"candidate_id"`
	BinaryDigest   string     `json:"binary_digest,omitempty"`
	ContractDigest string     `json:"contract_digest"`
	BuiltAt        *time.Time `json:"built_at,omitempty"`
}

func (r ReleaseIdentity) Validate() error {
	if strings.TrimSpace(r.Version) == "" || strings.TrimSpace(r.CandidateID) == "" {
		return errors.New("release version and candidate id required")
	}
	if r.BinaryDigest != "" && !ValidSHA256Digest(r.BinaryDigest) {
		return errors.New("invalid binary digest")
	}
	if !ValidSHA256Digest(r.ContractDigest) {
		return errors.New("invalid contract digest")
	}
	return nil
}

func DigestBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func ValidSHA256Digest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
