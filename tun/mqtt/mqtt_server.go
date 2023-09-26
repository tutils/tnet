package mqtt

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tutils/tnet/tun"
)

// tcp://localhost:port/topic/test
type addr struct {
	url *url.URL
}

func (a *addr) String() string {
	return a.url.String()
}

func (a *addr) scheme() string {
	return a.url.Scheme
}

func (a *addr) host() string {
	return a.url.Hostname()
}

func (a *addr) port() string {
	return a.url.Port()
}

func (a *addr) user() string {
	return a.url.User.Username()
}

func (a *addr) pass() string {
	p, _ := a.url.User.Password()
	return p
}

func (a *addr) path() string {
	return a.url.Path
}

func newAddr(rawURL string) *addr {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	return &addr{
		url: u,
	}
}

type reader struct {
	pr *os.File
	pw *os.File

	cli   mqtt.Client
	topic string
}

func (r *reader) Read(p []byte) (n int, err error) {
	return r.pr.Read(p)
}

func (r *reader) RunPipe() error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	r.pr = pr
	r.pw = pw

	token := r.cli.Subscribe(r.topic, 0, func(c mqtt.Client, m mqtt.Message) {
		// fmt.Println("@@Read: len", len(m.Payload()), m.Payload())
		pw.Write(m.Payload())
	})
	token.Wait()
	return token.Error()
}

func (r *reader) Close() error {
	token := r.cli.Unsubscribe(r.topic)
	token.Wait()

	err := r.pr.Close()
	if err != nil {
		return r.pw.Close()
	}
	return r.pw.Close()
}

func newReader(cli mqtt.Client, topic string) *reader {
	return &reader{
		cli:   cli,
		topic: topic,
	}
}

type writer struct {
	cli   mqtt.Client
	topic string
}

func (w *writer) Write(p []byte) (n int, err error) {
	// fmt.Println("@@Write: len", len(p), p)
	t := w.cli.Publish(w.topic, 0, false, p)
	t.Wait()
	return len(p), nil
}

func newWriter(cli mqtt.Client, topic string) *writer {
	return &writer{
		cli:   cli,
		topic: topic,
	}
}

var _ tun.Server = &server{}

type server struct {
	opts    tun.ServerOptions
	cli     mqtt.Client
	session sync.Map

	quit chan struct{}
}

func (s *server) Handler() tun.Handler {
	return s.opts.Handler
}

type conn struct {
	srv  *server
	tunr *reader
	tunw *writer
}

func (c *conn) serve(ctx context.Context) {
	defer c.tunr.Close()
	if h := c.srv.Handler(); h != nil {
		h.ServeTun(ctx, c.tunr, c.tunw)
	}
}

func (s *server) newMQTTTunConn(tunr *reader, tunw *writer) *conn {
	conn := &conn{
		srv:  s,
		tunr: tunr,
		tunw: tunw,
	}
	return conn
}

func (s *server) ListenAndServe() error {
	token := s.cli.Connect()
	token.Wait()
	err := token.Error()
	if err != nil {
		// TODO: log here
		fmt.Println("!!", err)
		return err
	}

	var tunID int64

	addr := newAddr(s.opts.Address)
	listenerTopic := fmt.Sprintf("%s/listener", addr.path())

	token = s.cli.Subscribe(listenerTopic, 0, func(mqttCli mqtt.Client, m mqtt.Message) {
		tunID++

		srvTopic := fmt.Sprintf("%s/srv/%d", addr.path(), tunID)
		cliTopic := fmt.Sprintf("%s/cli/%d", addr.path(), tunID)

		tunr := newReader(mqttCli, srvTopic)
		tunw := newWriter(mqttCli, cliTopic)
		ctx := context.WithValue(context.Background(), tun.TunIDContextKey{}, tunID)

		tunr.RunPipe()
		conn := s.newMQTTTunConn(tunr, tunw)
		go conn.serve(ctx)

		uniq := string(m.Payload())
		connectedTopic := fmt.Sprintf("%s/uniq/%s", addr.path(), uniq)
		s.cli.Publish(connectedTopic, 0, false, strconv.FormatInt(tunID, 10))
		fmt.Println("new conn: uniq", uniq, "tunID", tunID)
		s.session.Store(tunID, struct{}{})
	})
	token.Wait()
	err = token.Error()
	if err != nil {
		fmt.Println("!!", err)
		return err
	}
	// fmt.Println("sub", listenerTopic)

	// TODO: 替换成服务器循环向各个链接发送响应ping/pong
	<-s.quit
	return nil
}

func NewServer(opts ...tun.ServerOption) tun.Server {
	opt := tun.NewServerOptions(opts...)

	s := &server{
		opts: *opt,

		quit: make(chan struct{}, 1),
	}

	addr := newAddr(opt.Address)
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(fmt.Sprintf("%s://%s:%s", addr.scheme(), addr.host(), addr.port()))
	mqttOpts.SetClientID(genClientID("tnet_tun_srv"))
	if addr.user() != "" {
		mqttOpts.SetUsername(addr.user())
		mqttOpts.SetPassword(addr.pass())
	}

	mqttOpts.SetOnConnectHandler(func(c mqtt.Client) {
	}).SetConnectionLostHandler(func(c mqtt.Client, err error) {
	})

	cli := mqtt.NewClient(mqttOpts)
	s.cli = cli

	// r := newMQTTReader()

	// mux := http.NewServeMux()
	// mux.HandleFunc(addr.uri(), s.serveHTTP)
	// var tunID int64
	// srv := &http.Server{
	// 	Addr:    addr.host(),
	// 	Handler: mux,
	// 	ConnContext: func(ctx context.Context, c net.Conn) context.Context {
	// 		tunID++ // FIXME: not safe
	// 		return context.WithValue(ctx, TunIDContextKey{}, connID)
	// 	},
	// }
	// s.srv = srv

	return s
}

// const readTimeout = time.Second * 15
// const pingPeriod = time.Second * 10
// const writeTimeout = time.Second

// func startPing(conn *websocket.Conn, done chan struct{}) {
// 	conn.SetReadDeadline(time.Now().Add(readTimeout))
// 	conn.SetPongHandler(func(string) error {
// 		conn.SetReadDeadline(time.Now().Add(readTimeout))
// 		return nil
// 	})
// 	ticker := time.NewTicker(pingPeriod)
// 	defer ticker.Stop()
// 	for {
// 		select {
// 		case <-ticker.C:
// 			conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeTimeout))
// 		case <-done:
// 			return
// 		}
// 	}
// }
