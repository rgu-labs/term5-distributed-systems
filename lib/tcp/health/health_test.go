package health

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestNewDefaultTimeout(t *testing.T) {
	h := New(0)
	if h.timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", h.timeout)
	}
}

func TestNewCustomTimeout(t *testing.T) {
	h := New(10 * time.Second)
	if h.timeout != 10*time.Second {
		t.Errorf("timeout = %v, want 10s", h.timeout)
	}
}

func TestCheckRegisters(t *testing.T) {
	h := New(0)
	h.Check("db", func(ctx context.Context) error { return nil })

	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.checks) != 1 {
		t.Errorf("checks = %d, want 1", len(h.checks))
	}
}

func TestLivenessNoListener(t *testing.T) {
	h := New(0)
	if err := h.Liveness(); !errors.Is(err, ErrNoListener) {
		t.Errorf("Liveness() = %v, want ErrNoListener", err)
	}
}

func TestLivenessClosed(t *testing.T) {
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	defer lis.Close()

	h := New(0)
	h.SetListener(lis)
	h.SetClosed(true)

	if err := h.Liveness(); !errors.Is(err, ErrClosed) {
		t.Errorf("Liveness() = %v, want ErrClosed", err)
	}
}

func TestLivenessOK(t *testing.T) {
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	defer lis.Close()

	h := New(0)
	h.SetListener(lis)

	if err := h.Liveness(); err != nil {
		t.Errorf("Liveness() = %v, want nil", err)
	}
}

func TestReadinessNoListener(t *testing.T) {
	h := New(0)
	if err := h.Readiness(); !errors.Is(err, ErrNoListener) {
		t.Errorf("Readiness() = %v, want ErrNoListener", err)
	}
}

func TestReadinessClosed(t *testing.T) {
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	defer lis.Close()

	h := New(0)
	h.SetListener(lis)
	h.SetClosed(true)

	if err := h.Readiness(); !errors.Is(err, ErrClosed) {
		t.Errorf("Readiness() = %v, want ErrClosed", err)
	}
}

func TestReadinessOKNoChecks(t *testing.T) {
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	defer lis.Close()

	h := New(0)
	h.SetListener(lis)

	if err := h.Readiness(); err != nil {
		t.Errorf("Readiness() = %v, want nil", err)
	}
}

func TestReadinessOKWithChecks(t *testing.T) {
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	defer lis.Close()

	h := New(0)
	h.SetListener(lis)
	h.Check("db", func(ctx context.Context) error { return nil })
	h.Check("cache", func(ctx context.Context) error { return nil })

	if err := h.Readiness(); err != nil {
		t.Errorf("Readiness() = %v, want nil", err)
	}
}

func TestReadinessFailingCheck(t *testing.T) {
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	defer lis.Close()

	h := New(0)
	h.SetListener(lis)
	h.Check("db", func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	if err := h.Readiness(); err == nil {
		t.Error("Readiness() = nil, want error")
	}
}

func TestReadinessCheckTimeout(t *testing.T) {
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	defer lis.Close()

	h := New(10 * time.Millisecond)
	h.SetListener(lis)
	h.Check("slow", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	start := time.Now()
	if err := h.Readiness(); err == nil {
		t.Error("Readiness() = nil, want timeout error")
	}
	if time.Since(start) > time.Second {
		t.Error("Readiness() took too long")
	}
}

func TestRunChecks(t *testing.T) {
	h := New(0)
	h.Check("ok", func(ctx context.Context) error { return nil })
	h.Check("fail", func(ctx context.Context) error {
		return errors.New("down")
	})

	results := h.RunChecks(context.Background())
	if len(results) != 2 {
		t.Errorf("results = %d, want 2", len(results))
	}
	if results["ok"] != nil {
		t.Errorf("ok check = %v, want nil", results["ok"])
	}
	if results["fail"] == nil {
		t.Error("fail check = nil, want error")
	}
}

func TestRunChecksEmpty(t *testing.T) {
	h := New(0)
	results := h.RunChecks(context.Background())
	if len(results) != 0 {
		t.Errorf("results = %d, want 0", len(results))
	}
}
