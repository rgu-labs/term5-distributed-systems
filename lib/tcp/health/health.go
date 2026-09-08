package health

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"
)

var (
	ErrClosed      = errors.New("server is closed")
	ErrNoListener  = errors.New("listener is not set")
	ErrUnreachable = errors.New("server is unreachable")
)

type CheckFunc func(ctx context.Context) error

type Health struct {
	mu       sync.RWMutex
	checks   map[string]CheckFunc
	timeout  time.Duration
	listener net.Listener
	closed   bool
}

func New(timeout time.Duration) *Health {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Health{
		checks:  make(map[string]CheckFunc),
		timeout: timeout,
	}
}

func (h *Health) Check(name string, fn CheckFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = fn
}

func (h *Health) SetListener(lis net.Listener) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.listener = lis
}

func (h *Health) SetClosed(closed bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = closed
}

func (h *Health) Liveness() error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed {
		return ErrClosed
	}

	if h.listener == nil {
		return ErrNoListener
	}

	conn, err := net.DialTimeout("tcp", h.listener.Addr().String(), time.Second)
	if err != nil {
		return ErrUnreachable
	}
	_ = conn.Close()

	return nil
}

func (h *Health) Readiness() error {
	h.mu.RLock()
	checks := make(map[string]CheckFunc, len(h.checks))
	for k, v := range h.checks {
		checks[k] = v
	}
	closed := h.closed
	lis := h.listener
	h.mu.RUnlock()

	if closed {
		return ErrClosed
	}

	if lis == nil {
		return ErrNoListener
	}

	conn, err := net.DialTimeout("tcp", lis.Addr().String(), time.Second)
	if err != nil {
		return ErrUnreachable
	}
	_ = conn.Close()

	if len(checks) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	for name, check := range checks {
		if err := check(ctx); err != nil {
			slog.Warn("readiness check failed", "check", name, "error", err)
			return err
		}
	}

	return nil
}

func (h *Health) RunChecks(ctx context.Context) map[string]error {
	h.mu.RLock()
	checks := make(map[string]CheckFunc, len(h.checks))
	for k, v := range h.checks {
		checks[k] = v
	}
	h.mu.RUnlock()

	results := make(map[string]error, len(checks))
	for name, check := range checks {
		results[name] = check(ctx)
	}
	return results
}
