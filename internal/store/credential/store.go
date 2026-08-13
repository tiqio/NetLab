package credential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/netlab/netlab/internal/domain"
	_ "modernc.org/sqlite"
)

var ErrLocked = errors.New("credential store is locked")

type Store struct {
	db   *sql.DB
	aead cipher.AEAD
}

func Open(path, keyPath string) (*Store, error) {
	key, err := readKey(keyPath)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	clearBytes(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		"PRAGMA busy_timeout=5000",
		"PRAGMA journal_mode=WAL",
		`CREATE TABLE IF NOT EXISTS node_credentials (
id TEXT PRIMARY KEY,
laboratory_id TEXT NOT NULL,
node_id TEXT NOT NULL,
kind TEXT NOT NULL,
slot TEXT NOT NULL,
username_cipher BLOB NOT NULL,
password_cipher BLOB NOT NULL,
username_nonce BLOB NOT NULL,
password_nonce BLOB NOT NULL,
revision INTEGER NOT NULL,
state TEXT NOT NULL,
created_at TEXT NOT NULL,
rotated_at TEXT NOT NULL,
last_verified_at TEXT,
last_error_code TEXT NOT NULL DEFAULT '',
UNIQUE(node_id, kind, slot)
)`,
		"CREATE INDEX IF NOT EXISTS idx_node_credentials_owner ON node_credentials(laboratory_id,node_id,kind)",
	} {
		if _, err = db.Exec(statement); err != nil {
			db.Close()
			return nil, err
		}
	}
	if err = os.Chmod(path, 0o600); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, aead: aead}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Put(ctx context.Context, laboratoryID, nodeID domain.ID, kind, slot, username string, password []byte) (domain.NodeCredentialMetadata, error) {
	if laboratoryID == "" || nodeID == "" || strings.TrimSpace(kind) == "" || (slot != domain.CredentialSlotActive && slot != domain.CredentialSlotStaged) {
		return domain.NodeCredentialMetadata{}, fmt.Errorf("credential owner, kind, and slot are required")
	}
	if strings.TrimSpace(username) == "" {
		return domain.NodeCredentialMetadata{}, fmt.Errorf("credential username is required")
	}
	now := time.Now().UTC()
	var revision int64 = 1
	_ = s.db.QueryRowContext(ctx, `SELECT revision+1 FROM node_credentials WHERE node_id=? AND kind=? AND slot=?`, nodeID, kind, slot).Scan(&revision)
	usernameCipher, usernameNonce, err := s.seal([]byte(username), aad(laboratoryID, nodeID, kind, slot, revision, "username"))
	if err != nil {
		return domain.NodeCredentialMetadata{}, err
	}
	passwordCipher, passwordNonce, err := s.seal(password, aad(laboratoryID, nodeID, kind, slot, revision, "password"))
	if err != nil {
		return domain.NodeCredentialMetadata{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO node_credentials(id,laboratory_id,node_id,kind,slot,username_cipher,password_cipher,username_nonce,password_nonce,revision,state,created_at,rotated_at,last_verified_at,last_error_code)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_id,kind,slot) DO UPDATE SET laboratory_id=excluded.laboratory_id,username_cipher=excluded.username_cipher,password_cipher=excluded.password_cipher,username_nonce=excluded.username_nonce,password_nonce=excluded.password_nonce,revision=excluded.revision,state=excluded.state,rotated_at=excluded.rotated_at,last_verified_at=NULL,last_error_code=''`,
		domain.NewID(), laboratoryID, nodeID, kind, slot, usernameCipher, passwordCipher, usernameNonce, passwordNonce, revision, "pending_verification", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), nil, "")
	if err != nil {
		return domain.NodeCredentialMetadata{}, err
	}
	return s.Metadata(ctx, laboratoryID, nodeID, kind)
}

func (s *Store) Get(ctx context.Context, laboratoryID, nodeID domain.ID, kind, slot string) (domain.NodeCredentialSecret, error) {
	var usernameCipher, passwordCipher, usernameNonce, passwordNonce []byte
	var revision int64
	err := s.db.QueryRowContext(ctx, `SELECT username_cipher,password_cipher,username_nonce,password_nonce,revision FROM node_credentials WHERE laboratory_id=? AND node_id=? AND kind=? AND slot=?`, laboratoryID, nodeID, kind, slot).Scan(&usernameCipher, &passwordCipher, &usernameNonce, &passwordNonce, &revision)
	if err == sql.ErrNoRows {
		return domain.NodeCredentialSecret{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.NodeCredentialSecret{}, err
	}
	username, err := s.open(usernameCipher, usernameNonce, aad(laboratoryID, nodeID, kind, slot, revision, "username"))
	if err != nil {
		return domain.NodeCredentialSecret{}, fmt.Errorf("decrypt credential username: %w", err)
	}
	password, err := s.open(passwordCipher, passwordNonce, aad(laboratoryID, nodeID, kind, slot, revision, "password"))
	if err != nil {
		clearBytes(username)
		return domain.NodeCredentialSecret{}, fmt.Errorf("decrypt credential password: %w", err)
	}
	secret := domain.NodeCredentialSecret{Username: string(username), Password: password}
	clearBytes(username)
	return secret, nil
}

func (s *Store) Metadata(ctx context.Context, laboratoryID, nodeID domain.ID, kind string) (domain.NodeCredentialMetadata, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT slot,revision,state,created_at,rotated_at,last_verified_at,last_error_code FROM node_credentials WHERE laboratory_id=? AND node_id=? AND kind=? ORDER BY slot`, laboratoryID, nodeID, kind)
	if err != nil {
		return domain.NodeCredentialMetadata{}, err
	}
	defer rows.Close()
	metadata := domain.NodeCredentialMetadata{NodeID: nodeID, LaboratoryID: laboratoryID, Kind: kind, State: "credential_missing"}
	for rows.Next() {
		var slot, state, created, rotated, lastError string
		var verified sql.NullString
		var revision domain.Revision
		if err = rows.Scan(&slot, &revision, &state, &created, &rotated, &verified, &lastError); err != nil {
			return metadata, err
		}
		if slot == domain.CredentialSlotStaged {
			metadata.Staged = true
			continue
		}
		metadata.Configured = true
		metadata.Revision = revision
		metadata.State = state
		metadata.LastErrorCode = lastError
		metadata.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		metadata.RotatedAt, _ = time.Parse(time.RFC3339Nano, rotated)
		if verified.Valid {
			metadata.LastVerifiedAt, _ = time.Parse(time.RFC3339Nano, verified.String)
		}
	}
	return metadata, rows.Err()
}

func (s *Store) Mark(ctx context.Context, laboratoryID, nodeID domain.ID, kind, state, errorCode string) error {
	verified := any(nil)
	if state == "authenticated" {
		verified = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE node_credentials SET state=?,last_verified_at=COALESCE(?,last_verified_at),last_error_code=? WHERE laboratory_id=? AND node_id=? AND kind=? AND slot=?`, state, verified, errorCode, laboratoryID, nodeID, kind, domain.CredentialSlotActive)
	return err
}

func (s *Store) PromoteStaged(ctx context.Context, laboratoryID, nodeID domain.ID, kind string) error {
	staged, err := s.Get(ctx, laboratoryID, nodeID, kind, domain.CredentialSlotStaged)
	if err != nil {
		return err
	}
	defer staged.Clear()
	var activeRevision int64
	_ = s.db.QueryRowContext(ctx, `SELECT revision FROM node_credentials WHERE laboratory_id=? AND node_id=? AND kind=? AND slot=?`, laboratoryID, nodeID, kind, domain.CredentialSlotActive).Scan(&activeRevision)
	revision := activeRevision + 1
	if revision < 1 {
		revision = 1
	}
	usernameCipher, usernameNonce, err := s.seal([]byte(staged.Username), aad(laboratoryID, nodeID, kind, domain.CredentialSlotActive, revision, "username"))
	if err != nil {
		return err
	}
	passwordCipher, passwordNonce, err := s.seal(staged.Password, aad(laboratoryID, nodeID, kind, domain.CredentialSlotActive, revision, "password"))
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM node_credentials WHERE laboratory_id=? AND node_id=? AND kind=? AND slot=?`, laboratoryID, nodeID, kind, domain.CredentialSlotActive); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE node_credentials SET slot=?,username_cipher=?,password_cipher=?,username_nonce=?,password_nonce=?,revision=?,state='authenticated',last_verified_at=?,last_error_code='' WHERE laboratory_id=? AND node_id=? AND kind=? AND slot=?`, domain.CredentialSlotActive, usernameCipher, passwordCipher, usernameNonce, passwordNonce, revision, time.Now().UTC().Format(time.RFC3339Nano), laboratoryID, nodeID, kind, domain.CredentialSlotStaged)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

func (s *Store) Delete(ctx context.Context, laboratoryID, nodeID domain.ID, kind string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM node_credentials WHERE laboratory_id=? AND node_id=? AND kind=?`, laboratoryID, nodeID, kind)
	return err
}

func (s *Store) seal(plaintext, additionalData []byte) ([]byte, []byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	return s.aead.Seal(nil, nonce, plaintext, additionalData), nonce, nil
}

func (s *Store) open(ciphertext, nonce, additionalData []byte) ([]byte, error) {
	return s.aead.Open(nil, nonce, ciphertext, additionalData)
}

func aad(laboratoryID, nodeID domain.ID, kind, slot string, revision int64, field string) []byte {
	return []byte(fmt.Sprintf("netlab/v1/%s/%s/%s/%s/%d/%s", laboratoryID, nodeID, kind, slot, revision, field))
}

func readKey(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, ErrLocked
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLocked, err)
	}
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) == 32 {
		return body, nil
	}
	decoded, decodeErr := base64.StdEncoding.DecodeString(string(body))
	clearBytes(body)
	if decodeErr != nil || len(decoded) != 32 {
		clearBytes(decoded)
		return nil, fmt.Errorf("%w: master key must contain 32 raw bytes or base64-encoded 32 bytes", ErrLocked)
	}
	return decoded, nil
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
