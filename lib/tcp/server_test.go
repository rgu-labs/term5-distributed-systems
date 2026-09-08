package tcp

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewDefaultOptions(t *testing.T) {
	_, h := New(Config{Addr: "127.0.0.1:0"}, nil)

	if h == nil {
		t.Fatal("health is nil")
	}
}

func TestNewCustomAddr(t *testing.T) {
	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())

	addr := s.Addr()
	if addr == nil {
		t.Fatal("Addr() is nil")
	}

	conn, err := net.DialTimeout("tcp", addr.String(), time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	conn.Close()
}

func TestAddrReturnsListenerAddr(t *testing.T) {
	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())
	defer s.listener.Close()

	addr := s.Addr()
	if addr == nil {
		t.Fatal("Addr() is nil")
	}
}

func TestStartAcceptsConnections(t *testing.T) {
	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())
	defer s.Shutdown(context.Background())

	var mu sync.Mutex
	var conns int

	s.Start(HandlerFunc(func(ctx context.Context, conn net.Conn) {
		mu.Lock()
		conns++
		mu.Unlock()
		conn.Close()
	}))

	for i := 0; i < 3; i++ {
		conn, err := net.DialTimeout("tcp", s.Addr().String(), time.Second)
		if err != nil {
			t.Fatalf("Dial %d: %v", i, err)
		}
		conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if conns != 3 {
		t.Errorf("conns = %d, want 3", conns)
	}
}

func TestHandlerFuncImplementsHandler(t *testing.T) {
	called := false
	var Handler Handler = HandlerFunc(func(ctx context.Context, conn net.Conn) {
		called = true
		conn.Close()
	})

	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())
	defer s.Shutdown(context.Background())

	s.Start(Handler)

	conn, err := net.DialTimeout("tcp", s.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	conn.Close()

	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Error("HandlerFunc was not called")
	}
}

func TestShutdownWaitsForConnections(t *testing.T) {
	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())

	connDone := make(chan struct{})

	s.Start(HandlerFunc(func(ctx context.Context, conn net.Conn) {
		defer close(connDone)
		buf := make([]byte, 4)
		io.ReadFull(conn, buf)
		conn.Write(buf)
	}))

	conn, err := net.DialTimeout("tcp", s.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte("ping"))
	conn.Read(make([]byte, 4))

	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- s.Shutdown(context.Background())
	}()

	select {
	case <-connDone:
	case <-time.After(time.Second):
		t.Fatal("connection handler did not finish")
	}

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("Shutdown() = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Shutdown() did not complete")
	}
}

func TestShutdownTimeout(t *testing.T) {
	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())

	s.Start(HandlerFunc(func(ctx context.Context, conn net.Conn) {
		time.Sleep(5 * time.Second)
		conn.Close()
	}))

	conn, err := net.DialTimeout("tcp", s.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = s.Shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Shutdown() = %v, want DeadlineExceeded", err)
	}
}

func TestShutdownNoConnections(t *testing.T) {
	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())
	s.Start(HandlerFunc(func(ctx context.Context, conn net.Conn) {}))

	err := s.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Shutdown() = %v, want nil", err)
	}
}

func TestMultipleConnections(t *testing.T) {
	s, _ := New(Config{Addr: "127.0.0.1:0"}, None())
	defer s.Shutdown(context.Background())

	var mu sync.Mutex
	var count int

	s.Start(HandlerFunc(func(ctx context.Context, conn net.Conn) {
		mu.Lock()
		count++
		mu.Unlock()
		conn.Close()
	}))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", s.Addr().String(), time.Second)
			if err != nil {
				t.Errorf("Dial: %v", err)
				return
			}
			conn.Close()
		}()
	}
	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 10 {
		t.Errorf("count = %d, want 10", count)
	}
}

func TestOptionsDefault(t *testing.T) {
	o := Default()
	if !o.health || !o.logger {
		t.Error("Default() should enable health and logger")
	}
}

func TestOptionsNone(t *testing.T) {
	o := None()
	if o.health || o.logger {
		t.Error("None() should disable all options")
	}
}

func TestOptionsWithoutHealth(t *testing.T) {
	base := Default()
	updated := base.WithoutHealth()

	if updated.health {
		t.Error("WithoutHealth() did not disable health")
	}
	if !base.health {
		t.Error("original options were mutated")
	}
}

func TestOptionsWithoutLogger(t *testing.T) {
	base := Default()
	updated := base.WithoutLogger()

	if updated.logger {
		t.Error("WithoutLogger() did not disable logger")
	}
	if !base.logger {
		t.Error("original options were mutated")
	}
}
