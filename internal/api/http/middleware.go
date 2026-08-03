package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/netlab/netlab/internal/app/command"
	"github.com/netlab/netlab/internal/domain"
)

func RequestLimits(maxBodyBytes int64) gin.HandlerFunc {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 4 << 20
	}
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBodyBytes {
			writeProblem(c, http.StatusRequestEntityTooLarge, domain.Problem{Code: "resource_exhausted", Message: "request body exceeds configured limit", Retryable: false})
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	}
}

func RevisionFromRequest(c *gin.Context) (domain.Revision, bool) {
	revision, err := ParseRevision(c.GetHeader("If-Match"))
	if err != nil || revision < 1 {
		writeProblem(c, http.StatusPreconditionRequired, domain.Problem{Code: "precondition_required", Message: "valid If-Match revision required"})
		return 0, false
	}
	return revision, true
}

func WriteIdempotencyError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, command.ErrIdempotencyConflict):
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "idempotency_conflict", Message: err.Error(), Phase: "idempotency", Cleanup: "no duplicate mutation executed", OperatorHint: "reuse the key only with the original payload or choose a new key"})
		return true
	case errors.Is(err, command.ErrIdempotencyPending):
		c.Header("Retry-After", "1")
		writeProblem(c, http.StatusConflict, domain.Problem{Code: "idempotency_pending", Message: err.Error(), Retryable: true, Phase: "idempotency", Cleanup: "original mutation remains in progress", OperatorHint: "retry the same key after the indicated delay", RetryAfterSeconds: 1})
		return true
	default:
		return false
	}
}

func SetRetryHeaders(c *gin.Context, problem domain.Problem) {
	if problem.Retryable {
		retryAfter := problem.RetryAfterSeconds
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Header("Retry-After", strconv.Itoa(retryAfter))
	}
	if strings.Contains(problem.Code, "resource_exhausted") {
		c.Header("X-RateLimit-Remaining", strconv.Itoa(0))
	}
}
