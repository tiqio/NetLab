package console

import (
	"errors"
	"io"
	"sync"
	"time"
)

const (
	defaultDetachGrace = 2 * time.Minute
	defaultHistorySize = 256 << 10
	clientWriteTimeout = 5 * time.Second
)

type PersistentSession struct {
	backend        io.ReadWriteCloser
	detachGrace    time.Duration
	historyLimit   int
	backendWriteMu sync.Mutex
	mu             sync.Mutex
	history        []byte
	client         *sessionClient
	subscribers    map[uint64]chan []byte
	nextSubscriber uint64
	detachTimer    *time.Timer
	maximumTimer   *time.Timer
	closed         chan struct{}
	closeOnce      sync.Once
	onClose        func()
}

type sessionClient struct {
	connection io.ReadWriteCloser
	writeMu    sync.Mutex
}

func NewPersistentSession(backend io.ReadWriteCloser, limits Limits, detachGrace time.Duration, historyLimit int, onClose func()) *PersistentSession {
	if detachGrace <= 0 {
		detachGrace = defaultDetachGrace
	}
	if historyLimit <= 0 {
		historyLimit = defaultHistorySize
	}
	if limits.MaximumSession <= 0 {
		limits.MaximumSession = 8 * time.Hour
	}
	session := &PersistentSession{
		backend:      backend,
		detachGrace:  detachGrace,
		historyLimit: historyLimit,
		subscribers:  map[uint64]chan []byte{},
		closed:       make(chan struct{}),
		onClose:      onClose,
	}
	session.mu.Lock()
	session.detachTimer = time.AfterFunc(detachGrace, session.Close)
	session.maximumTimer = time.AfterFunc(limits.MaximumSession, session.Close)
	session.mu.Unlock()
	go session.readBackend()
	return session
}

func (s *PersistentSession) Attach(connection io.ReadWriteCloser) error {
	client := &sessionClient{connection: connection}
	client.writeMu.Lock()

	s.mu.Lock()
	select {
	case <-s.closed:
		s.mu.Unlock()
		client.writeMu.Unlock()
		_ = connection.Close()
		return io.ErrClosedPipe
	default:
	}
	if s.detachTimer != nil {
		s.detachTimer.Stop()
	}
	previous := s.client
	s.client = client
	history := append([]byte(nil), s.history...)
	s.mu.Unlock()

	if previous != nil {
		_ = previous.connection.Close()
	}
	if len(history) > 0 {
		if err := writeSessionClient(client.connection, history); err != nil {
			client.writeMu.Unlock()
			s.detach(client)
			return err
		}
	}
	client.writeMu.Unlock()

	buffer := make([]byte, 32<<10)
	for {
		count, err := connection.Read(buffer)
		if count > 0 {
			s.backendWriteMu.Lock()
			_, writeErr := s.backend.Write(buffer[:count])
			s.backendWriteMu.Unlock()
			if writeErr != nil {
				s.Close()
				return writeErr
			}
		}
		if err != nil {
			s.detach(client)
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
				return nil
			}
			return err
		}
	}
}

func (s *PersistentSession) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		client := s.client
		s.client = nil
		if s.detachTimer != nil {
			s.detachTimer.Stop()
		}
		if s.maximumTimer != nil {
			s.maximumTimer.Stop()
		}
		close(s.closed)
		s.mu.Unlock()
		if client != nil {
			_ = client.connection.Close()
		}
		_ = s.backend.Close()
		if s.onClose != nil {
			s.onClose()
		}
	})
}

func (s *PersistentSession) Done() <-chan struct{} {
	return s.closed
}

func (s *PersistentSession) Write(value []byte) error {
	select {
	case <-s.closed:
		return io.ErrClosedPipe
	default:
	}
	s.backendWriteMu.Lock()
	defer s.backendWriteMu.Unlock()
	_, err := s.backend.Write(value)
	return err
}

func (s *PersistentSession) Subscribe(buffer int) (<-chan []byte, func()) {
	if buffer < 1 {
		buffer = 1
	}
	updates := make(chan []byte, buffer)
	s.mu.Lock()
	s.nextSubscriber++
	id := s.nextSubscriber
	s.subscribers[id] = updates
	s.mu.Unlock()
	var once sync.Once
	return updates, func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subscribers, id)
			s.mu.Unlock()
		})
	}
}

func (s *PersistentSession) detach(client *sessionClient) {
	s.mu.Lock()
	if s.client == client {
		s.client = nil
		if s.detachTimer != nil {
			s.detachTimer.Stop()
		}
		s.detachTimer = time.AfterFunc(s.detachGrace, s.Close)
	}
	s.mu.Unlock()
	_ = client.connection.Close()
}

func (s *PersistentSession) readBackend() {
	buffer := make([]byte, 32<<10)
	for {
		count, err := s.backend.Read(buffer)
		if count > 0 {
			s.publish(buffer[:count])
		}
		if err != nil {
			s.Close()
			return
		}
	}
}

func (s *PersistentSession) publish(value []byte) {
	chunk := append([]byte(nil), value...)
	s.mu.Lock()
	s.history = append(s.history, chunk...)
	if overflow := len(s.history) - s.historyLimit; overflow > 0 {
		copy(s.history, s.history[overflow:])
		s.history = s.history[:s.historyLimit]
	}
	client := s.client
	for _, subscriber := range s.subscribers {
		select {
		case subscriber <- chunk:
		default:
		}
	}
	s.mu.Unlock()
	if client == nil {
		return
	}
	client.writeMu.Lock()
	err := writeSessionClient(client.connection, chunk)
	client.writeMu.Unlock()
	if err != nil {
		s.detach(client)
	}
}

func writeSessionClient(destination io.Writer, value []byte) error {
	if deadline, ok := destination.(interface{ SetWriteDeadline(time.Time) error }); ok {
		_ = deadline.SetWriteDeadline(time.Now().Add(clientWriteTimeout))
		defer deadline.SetWriteDeadline(time.Time{})
	}
	_, err := destination.Write(value)
	return err
}
