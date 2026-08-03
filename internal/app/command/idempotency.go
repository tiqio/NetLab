package command

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

var (
	ErrIdempotencyConflict = errors.New("idempotency key was already used with a different request")
	ErrIdempotencyPending  = errors.New("idempotent request is still in progress")
)

type IdempotencyRepository interface {
	GetIdempotency(context.Context, string, string) (domain.IdempotencyRecord, error)
	CreateIdempotency(context.Context, domain.IdempotencyRecord) error
	CompleteIdempotency(context.Context, domain.IdempotencyRecord) error
	DeleteIdempotency(context.Context, string, string) error
}

type IdempotencyResult struct {
	StatusCode int
	Body       []byte
	Replay     bool
}

type IdempotencyService struct {
	repository IdempotencyRepository
	ttl        time.Duration
	now        func() time.Time
	mu         sync.Mutex
}

func NewIdempotencyService(repository IdempotencyRepository, ttl time.Duration) *IdempotencyService {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &IdempotencyService{repository: repository, ttl: ttl, now: func() time.Time { return time.Now().UTC() }}
}

func RequestFingerprint(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *IdempotencyService) Execute(ctx context.Context, scope, key string, request []byte, operation func(context.Context) (int, []byte, error)) (IdempotencyResult, error) {
	if key == "" {
		status, body, err := operation(ctx)
		return IdempotencyResult{StatusCode: status, Body: body}, err
	}
	if len(key) > 256 {
		return IdempotencyResult{}, fmt.Errorf("idempotency key exceeds 256 bytes")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	fingerprint := RequestFingerprint(request)
	record, err := s.repository.GetIdempotency(ctx, scope, key)
	if err == nil {
		if !record.ExpiresAt.After(now) {
			if deleteErr := s.repository.DeleteIdempotency(ctx, scope, key); deleteErr != nil {
				return IdempotencyResult{}, deleteErr
			}
		} else {
			if record.RequestFingerprint != fingerprint {
				return IdempotencyResult{}, ErrIdempotencyConflict
			}
			if record.State != "completed" {
				return IdempotencyResult{}, ErrIdempotencyPending
			}
			return IdempotencyResult{StatusCode: record.StatusCode, Body: append([]byte(nil), record.Response...), Replay: true}, nil
		}
	} else if !isNotFound(err) {
		return IdempotencyResult{}, err
	}

	record = domain.IdempotencyRecord{Scope: scope, Key: key, RequestFingerprint: fingerprint, State: "pending", CreatedAt: now, ExpiresAt: now.Add(s.ttl)}
	if err = s.repository.CreateIdempotency(ctx, record); err != nil {
		return IdempotencyResult{}, err
	}
	status, body, operationErr := operation(ctx)
	record.State = "completed"
	record.StatusCode = status
	if operationErr == nil {
		record.Response = append([]byte(nil), body...)
	} else {
		record.Error = []byte(operationErr.Error())
	}
	if err = s.repository.CompleteIdempotency(ctx, record); err != nil {
		return IdempotencyResult{}, err
	}
	return IdempotencyResult{StatusCode: status, Body: body}, operationErr
}

func isNotFound(err error) bool {
	return err != nil && err.Error() == "not found"
}
