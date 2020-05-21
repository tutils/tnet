package tcp

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "tnet/tcp context value " + k.name }

// context keys
var (
	ServerContextKey = &contextKey{"tcp-server"}
)

type srvConn struct {
	conn
	server *Server
}

func (c *srvConn) onSetStateHook(state ConnState) {
	s := c.server
	switch state {
	case StateNew:
		s.trackConn(c, true)
	case StateClosed:
		s.trackConn(c, false)
	}
}

type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() { oc.closeErr = oc.Listener.Close() }

// Server over tcp
type Server struct {
	opts ServerOptions

	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	activeConn map[*srvConn]struct{}

	inShutdown atomicBool // true when when server is in shutdown
	onShutdown []func()
	doneChan   chan struct{}
}

// ErrServerClosed means server has been closed
var ErrServerClosed = errors.New("tnet/tcp: Server closed")

// ListenAndServe starts tcp server
func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}

	l, err := net.Listen("tcp", srv.opts.addr)
	if err != nil {
		return err
	}
	srv.opts.errorLogFunc("tnet/tcp: serve on %s\n", l.Addr().String())

	origListener := l
	l = &onceCloseListener{Listener: l}
	defer l.Close()

	if !srv.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer srv.trackListener(&l, false)

	baseCtx := context.Background()
	if srv.opts.baseContext != nil {
		baseCtx = srv.opts.baseContext(origListener)
		if baseCtx == nil {
			panic("BaseContext returned a nil context")
		}
	}

	var tempDelay time.Duration // how long to sleep on accept failure

	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	for {
		rw, err := l.Accept()
		if err != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.opts.errorLogFunc("tnet/tcp: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}

		connCtx := ctx
		if cc := srv.opts.connContext; cc != nil {
			connCtx = cc(connCtx, rw)
			if connCtx == nil {
				panic("ConnContext returned nil")
			}
		}
		tempDelay = 0

		c := srv.newConn(rw)
		c.setState(StateNew) // before serve can return
		go c.serve(connCtx, srv.opts.handler, srv.opts.errorLogFunc)
	}
}

func (srv *Server) shuttingDown() bool {
	return srv.inShutdown.isSet()
}

// RegisterOnShutdown registers OnShutdown functions
func (srv *Server) RegisterOnShutdown(f func()) {
	srv.mu.Lock()
	srv.onShutdown = append(srv.onShutdown, f)
	srv.mu.Unlock()
}

// Close close all connections of server
func (srv *Server) Close() error {
	srv.inShutdown.setTrue()
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.closeDoneChanLocked()
	err := srv.closeListenersLocked()
	for c := range srv.activeConn {
		c.rwc.Close()
		delete(srv.activeConn, c)
	}
	return err
}

// Shutdown shutdowns client graceful
func (srv *Server) Shutdown(ctx context.Context) error {
	srv.inShutdown.setTrue()

	srv.mu.Lock()
	lnerr := srv.closeListenersLocked()
	srv.closeDoneChanLocked()
	for _, f := range srv.onShutdown {
		go f()
	}
	srv.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if srv.closeIdleConns() && srv.numListeners() == 0 {
			return lnerr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (srv *Server) closeListenersLocked() error {
	var err error
	for ln := range srv.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

func (srv *Server) closeDoneChanLocked() {
	ch := srv.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by srv.mu.
		close(ch)
	}
}

func (srv *Server) getDoneChan() <-chan struct{} {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.getDoneChanLocked()
}

func (srv *Server) getDoneChanLocked() chan struct{} {
	if srv.doneChan == nil {
		srv.doneChan = make(chan struct{})
	}
	return srv.doneChan
}

func (srv *Server) closeIdleConns() bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	quiescent := true
	for c := range srv.activeConn {
		st, unixSec := c.getState()
		// Issue 22682: treat StateNew connections as if
		// they're idle if we haven't read the first request's
		// header in over 5 seconds.
		if st == StateNew && unixSec < time.Now().Unix()-5 {
			st = StateIdle
		}
		if st != StateIdle || unixSec == 0 {
			// Assume unixSec == 0 means it's a very new
			// connection, without state set yet.
			quiescent = false
			continue
		}
		c.rwc.Close()
		delete(srv.activeConn, c)
	}
	return quiescent
}

func (srv *Server) numListeners() int {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return len(srv.listeners)
}

func (srv *Server) trackListener(ln *net.Listener, add bool) bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.listeners == nil {
		srv.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if srv.shuttingDown() {
			return false
		}
		srv.listeners[ln] = struct{}{}
	} else {
		delete(srv.listeners, ln)
	}
	return true
}

func (srv *Server) trackConn(c *srvConn, add bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.activeConn == nil {
		srv.activeConn = make(map[*srvConn]struct{})
	}
	if add {
		srv.activeConn[c] = struct{}{}
	} else {
		delete(srv.activeConn, c)
	}
}

const debugServerConnections = false

func (srv *Server) newConn(rwc net.Conn) *srvConn {
	c := &srvConn{
		server: srv,
	}
	c.rwc = rwc
	if debugServerConnections {
		c.rwc = newLoggingConn("server", c.rwc)
	}
	c.onSetState = c.onSetStateHook
	return c
}

// NewServer create a new tcp server
func NewServer(opts ...ServerOption) *Server {
	opt := newServerOptions(opts...)

	return &Server{
		opts: *opt,
	}
}
