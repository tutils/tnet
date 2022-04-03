package tcp

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// connext keys
var (
	ClientContextKey = &contextKey{"tcp-client"}
)

type cliConn struct {
	conn
	client *Client
}

func (c *cliConn) onSetStateHook(state ConnState) {
	s := c.client
	switch state {
	case StateNew:
		s.trackConn(c, true)
	case StateClosed:
		s.trackConn(c, false)
	}
}

type onceCancelDialer struct {
	net.Dialer
	cancel context.CancelFunc
	once   sync.Once
}

func (oc *onceCancelDialer) Cancel() {
	oc.once.Do(oc.cancel)
}

// Client over tcp
type Client struct {
	opts ClientOptions

	mu         sync.Mutex
	dialers    map[*onceCancelDialer]struct{}
	activeConn map[*cliConn]struct{}

	inShutdown atomicBool // true when when server is in shutdown
	onShutdown []func()
	doneChan   chan struct{}
}

const debugClientConnections = false

func (cli *Client) newConn(rwc net.Conn) *cliConn {
	c := &cliConn{
		client: cli,
	}
	c.rwc = rwc
	if debugClientConnections {
		c.rwc = newLoggingConn("client", c.rwc)
	}
	c.onSetState = c.onSetStateHook
	return c
}

// ErrClientClosed means client has been closed
var ErrClientClosed = errors.New("tnet/tcp: Client closed")

// ErrConnectionRefused means connection refused
var ErrConnectionRefused = errors.New("tnet/tcp: Connection refused")

// DialAndServe starts client
func (cli *Client) DialAndServe(ctx context.Context) error {
	addr := cli.opts.addr
	if addr == "" {
		panic("empty address")
	}

	ctx = context.WithValue(ctx, ClientContextKey, cli)

	var dialer net.Dialer
	ctx, cancel := context.WithCancel(ctx)
	d := &onceCancelDialer{Dialer: dialer, cancel: cancel}
	defer d.Cancel()

	if !cli.trackDialer(d, true) {
		return ErrClientClosed
	}
	defer cli.trackDialer(d, false)

	rw, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		select {
		case <-cli.getDoneChan():
			return ErrClientClosed
		default:
			return ErrConnectionRefused
		}
	}

	if period := cli.opts.keepAlivePeriod; period > 0 {
		tcpConn := rw.(*net.TCPConn)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(period)
		if count := cli.opts.keepAliveCount; count > 0 {
			SetKeepAliveCount(tcpConn, count)
		}
	}

	connCtx := ctx
	if cc := cli.opts.connContext; cc != nil {
		connCtx = cc(connCtx, rw)
		if connCtx == nil {
			panic("ConnContext returned nil")
		}
	}
	c := cli.newConn(rw)
	c.setState(StateNew)
	c.serve(connCtx, cli.opts.handler, cli.opts.errorLogFunc)
	return nil
}

func (cli *Client) shuttingDown() bool {
	return cli.inShutdown.isSet()
}

// RegisterOnShutdown registers OnShutdown funcs
func (cli *Client) RegisterOnShutdown(f func()) {
	cli.mu.Lock()
	cli.onShutdown = append(cli.onShutdown, f)
	cli.mu.Unlock()
}

// Close close all connections of client
func (cli *Client) Close() error {
	cli.inShutdown.setTrue()
	cli.mu.Lock()
	defer cli.mu.Unlock()
	cli.closeDoneChanLocked()
	for c := range cli.activeConn {
		c.rwc.Close()
		delete(cli.activeConn, c)
	}
	return nil
}

// Shutdown shutdowns client graceful
func (cli *Client) Shutdown(ctx context.Context) error {
	cli.inShutdown.setTrue()

	cli.mu.Lock()
	cli.closeDoneChanLocked()
	for _, f := range cli.onShutdown {
		go f()
	}
	cli.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if cli.closeIdleConns() && cli.numDialers() == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (cli *Client) closeDoneChanLocked() {
	ch := cli.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by cli.mu.
		close(ch)
	}
}

func (cli *Client) getDoneChan() <-chan struct{} {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	return cli.getDoneChanLocked()
}

func (cli *Client) getDoneChanLocked() chan struct{} {
	if cli.doneChan == nil {
		cli.doneChan = make(chan struct{})
	}
	return cli.doneChan
}

func (cli *Client) closeIdleConns() bool {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	quiescent := true
	for c := range cli.activeConn {
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
		delete(cli.activeConn, c)
	}
	return quiescent
}

func (cli *Client) numDialers() int {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	return len(cli.dialers)
}

func (cli *Client) trackDialer(d *onceCancelDialer, add bool) bool {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	if cli.dialers == nil {
		cli.dialers = make(map[*onceCancelDialer]struct{})
	}
	if add {
		if cli.shuttingDown() {
			return false
		}
		cli.dialers[d] = struct{}{}
	} else {
		delete(cli.dialers, d)
	}
	return true
}

func (cli *Client) trackConn(c *cliConn, add bool) {
	cli.mu.Lock()
	defer cli.mu.Unlock()
	if cli.activeConn == nil {
		cli.activeConn = make(map[*cliConn]struct{})
	}
	if add {
		cli.activeConn[c] = struct{}{}
	} else {
		delete(cli.activeConn, c)
	}
}

// NewClient create a new tcp client
func NewClient(opts ...ClientOption) *Client {
	opt := newClientOptions(opts...)

	return &Client{
		opts: *opt,
	}
}
