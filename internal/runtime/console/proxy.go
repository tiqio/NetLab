package console

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

type Limits struct {
	IdleTimeout    time.Duration
	BytesPerSecond int64
	MaximumSession time.Duration
}

func Bridge(ctx context.Context, client io.ReadWriteCloser, backend io.ReadWriteCloser, limits Limits) error {
	defer client.Close()
	defer backend.Close()
	if limits.MaximumSession <= 0 {
		limits.MaximumSession = 8 * time.Hour
	}
	ctx, cancel := context.WithTimeout(ctx, limits.MaximumSession)
	defer cancel()
	type result struct{ err error }
	results := make(chan result, 2)
	var once sync.Once
	copyDirection := func(destination io.Writer, source io.Reader) {
		buffer := make([]byte, 32<<10)
		for {
			if limits.IdleTimeout > 0 {
				if deadline, ok := source.(interface{ SetReadDeadline(time.Time) error }); ok {
					_ = deadline.SetReadDeadline(time.Now().Add(limits.IdleTimeout))
				}
			}
			count, err := source.Read(buffer)
			if count > 0 {
				if limits.BytesPerSecond > 0 {
					delay := time.Duration(int64(time.Second) * int64(count) / limits.BytesPerSecond)
					if delay > 0 {
						select {
						case <-ctx.Done():
							results <- result{ctx.Err()}
							return
						case <-time.After(delay):
						}
					}
				}
				if _, writeErr := destination.Write(buffer[:count]); writeErr != nil {
					results <- result{writeErr}
					return
				}
			}
			if err != nil {
				results <- result{err}
				return
			}
		}
	}
	go copyDirection(backend, client)
	go copyDirection(client, backend)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case value := <-results:
		once.Do(func() { _ = client.Close(); _ = backend.Close() })
		if errors.Is(value.err, io.EOF) || errors.Is(value.err, io.ErrClosedPipe) {
			return nil
		}
		return value.err
	}
}
