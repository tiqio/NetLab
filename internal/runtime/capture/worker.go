package capture

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

var interfaceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,64}$`)

type WorkerConfig struct {
	OwnershipID domain.ID
	Interface   string
	Filter      string
	Format      string
	MaxBytes    int64
	Duration    time.Duration
	RetainPath  string
	Direction   string
}

type Worker struct {
	config      WorkerConfig
	cancel      context.CancelFunc
	done        chan struct{}
	mu          sync.Mutex
	subscribers map[int]chan []byte
	nextID      int
	bytes       atomic.Int64
	packets     atomic.Int64
	truncated   atomic.Bool
	stopping    atomic.Bool
	err         error
	replay      []byte
	counter     *packetCounter
}

func NewWorker(config WorkerConfig) (*Worker, error) {
	if !interfaceNamePattern.MatchString(config.Interface) {
		return nil, fmt.Errorf("invalid capture interface")
	}
	if config.MaxBytes <= 0 {
		config.MaxBytes = 256 << 20
	}
	if config.Duration <= 0 {
		config.Duration = 15 * time.Minute
	}
	if config.Format == "" {
		config.Format = "pcap"
	}
	return &Worker{config: config, done: make(chan struct{}), subscribers: map[int]chan []byte{}, counter: newPacketCounter(config.Format)}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, w.config.Duration)
	w.cancel = cancel
	command, err := captureCommand(ctx, w.config)
	if err != nil {
		cancel()
		return err
	}
	if w.config.OwnershipID != "" {
		command.Env = append(os.Environ(), "NETLAB_OWNERSHIP=capture:"+string(w.config.OwnershipID))
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		cancel()
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		cancel()
		return err
	}
	if err = command.Start(); err != nil {
		cancel()
		return err
	}
	go io.Copy(io.Discard, io.LimitReader(stderr, 64<<10))
	go func() {
		w.consume(ctx, stdout)
		waitErr := command.Wait()
		if shouldRecordWaitError(waitErr, w.stopping.Load(), w.truncated.Load(), ctx.Err()) {
			w.err = waitErr
		}
		w.finish()
	}()
	return nil
}

func shouldRecordWaitError(waitErr error, stopping, truncated bool, contextErr error) bool {
	return waitErr != nil && !stopping && !truncated && !errors.Is(contextErr, context.Canceled) && !errors.Is(contextErr, context.DeadlineExceeded)
}

func (w *Worker) StartReader(ctx context.Context, reader io.Reader) {
	ctx, cancel := context.WithTimeout(ctx, w.config.Duration)
	w.cancel = cancel
	go func() {
		w.consume(ctx, reader)
		w.finish()
	}()
}

func captureCommand(ctx context.Context, config WorkerConfig) (*exec.Cmd, error) {
	if config.Direction != "" && config.Direction != "ingress" && config.Direction != "egress" {
		return nil, fmt.Errorf("invalid capture direction")
	}
	if path, err := exec.LookPath("dumpcap"); err == nil && config.Direction == "" {
		args := []string{"-i", config.Interface, "-P", "-w", "-"}
		if config.Format == "pcapng" {
			args = []string{"-i", config.Interface, "-w", "-"}
		}
		if config.Filter != "" {
			args = append(args, "-f", config.Filter)
		}
		return exec.CommandContext(ctx, path, args...), nil
	}
	path, err := exec.LookPath("tcpdump")
	if err != nil {
		return nil, fmt.Errorf("neither dumpcap nor tcpdump is installed")
	}
	args := []string{"--immediate-mode", "-U", "-n", "-i", config.Interface, "-w", "-"}
	if config.Direction == "ingress" {
		args = append(args, "-Q", "in")
	}
	if config.Direction == "egress" {
		args = append(args, "-Q", "out")
	}
	if config.Filter != "" {
		args = append(args, config.Filter)
	}
	return exec.CommandContext(ctx, path, args...), nil
}

func (w *Worker) consume(ctx context.Context, reader io.Reader) {
	var retained *os.File
	if w.config.RetainPath != "" {
		retained, w.err = os.OpenFile(w.config.RetainPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if w.err != nil {
			return
		}
		defer retained.Close()
	}
	buffer := make([]byte, 64<<10)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			remaining := w.config.MaxBytes - w.bytes.Load()
			if remaining <= 0 {
				w.truncated.Store(true)
				return
			}
			if int64(count) > remaining {
				count = int(remaining)
				w.truncated.Store(true)
			}
			chunk := append([]byte(nil), buffer[:count]...)
			w.bytes.Add(int64(count))
			w.packets.Add(w.counter.Add(chunk))
			if retained != nil {
				_, _ = retained.Write(chunk)
			}
			w.publish(chunk)
			if w.truncated.Load() {
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				w.err = err
			}
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (w *Worker) publish(chunk []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.replay) < 1<<20 {
		remaining := (1 << 20) - len(w.replay)
		if len(chunk) > remaining {
			w.replay = append(w.replay, chunk[:remaining]...)
		} else {
			w.replay = append(w.replay, chunk...)
		}
	}
	for _, subscriber := range w.subscribers {
		select {
		case subscriber <- chunk:
		default:
		}
	}
}

func (w *Worker) Subscribe() (<-chan []byte, func()) {
	w.mu.Lock()
	id := w.nextID
	w.nextID++
	channel := make(chan []byte, 16)
	w.subscribers[id] = channel
	if len(w.replay) > 0 {
		channel <- append([]byte(nil), w.replay...)
	}
	w.mu.Unlock()
	return channel, func() {
		w.mu.Lock()
		if current := w.subscribers[id]; current != nil {
			delete(w.subscribers, id)
			close(current)
		}
		w.mu.Unlock()
	}
}

func (w *Worker) Stop() {
	w.stopping.Store(true)
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *Worker) Done() <-chan struct{} { return w.done }
func (w *Worker) Bytes() int64          { return w.bytes.Load() }
func (w *Worker) Packets() int64        { return w.packets.Load() }
func (w *Worker) Truncated() bool       { return w.truncated.Load() }
func (w *Worker) Stopping() bool        { return w.stopping.Load() }
func (w *Worker) Error() error          { return w.err }

func (w *Worker) finish() {
	w.mu.Lock()
	for id, subscriber := range w.subscribers {
		delete(w.subscribers, id)
		close(subscriber)
	}
	w.mu.Unlock()
	select {
	case <-w.done:
	default:
		close(w.done)
	}
}
