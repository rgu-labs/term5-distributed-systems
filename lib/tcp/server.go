package tcp

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/rgu-labs/term5-distributed-systems/lib/log"
	"github.com/rgu-labs/term5-distributed-systems/lib/tcp/health"
)

type Handler interface {
	HandleConn(ctx context.Context, conn net.Conn)
}

type HandlerFunc func(ctx context.Context, conn net.Conn)

func (f HandlerFunc) HandleConn(ctx context.Context, conn net.Conn) {
	f(ctx, conn)
}

type Server struct {
	listener net.Listener
	health   *health.Health
	opts     *Options
	mu       sync.Mutex
	conns    map[net.Conn]struct{}
	done     chan struct{}
}

func New(cfg Config, opts *Options) (*Server, *health.Health) {
	if opts == nil {
		opts = Default()
	}

	h := health.New(5 * time.Second)

	s := &Server{
		health: h,
		opts:   opts,
		conns:  make(map[net.Conn]struct{}),
		done:   make(chan struct{}),
	}

	addr := cfg.Addr
	if addr == "" {
		addr = ":0"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("failed to listen", "addr", addr, "err", err)
		panic("tcp: " + err.Error())
	}

	s.listener = lis
	h.SetListener(lis)

	return s, h
}

func (s *Server) Start(handler Handler) {
	close(s.done)

	go s.acceptLoop(handler)
}

func (s *Server) acceptLoop(handler Handler) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Error("accept error", "err", err)
			continue
		}

		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		if s.opts.logger {
			log.Info("client connected", "remote", conn.RemoteAddr().String())
		}

		go s.handleConn(handler, conn)
	}
}

func (s *Server) handleConn(handler Handler, conn net.Conn) {
	defer func() {
		_ = conn.Close()
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()

		if s.opts.logger {
			log.Info("client disconnected", "remote", conn.RemoteAddr().String())
		}
	}()

	ctx := context.Background()
	handler.HandleConn(ctx, conn)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.health.SetClosed(true)
	_ = s.listener.Close()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		s.mu.Lock()
		n := len(s.conns)
		s.mu.Unlock()

		if n == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}
