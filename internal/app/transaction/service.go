package transaction

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/netlab/netlab/internal/domain"
)

type Writer interface {
	Write(context.Context, func(*sql.Tx) error) error
}

type Service struct{ writer Writer }

func New(writer Writer) *Service { return &Service{writer: writer} }

func (s *Service) Write(ctx context.Context, fn func(*sql.Tx) error) error {
	return s.writer.Write(ctx, fn)
}

func RequireRevision(actual, expected domain.Revision) error {
	if actual != expected {
		return domain.Problem{Code: "revision_conflict", Message: fmt.Sprintf("expected revision %d, got %d", expected, actual), Retryable: true}
	}
	return nil
}
