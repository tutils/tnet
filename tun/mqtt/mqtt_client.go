package mqtt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tutils/tnet/tun"
)

var _ tun.Client = &client{}

type client struct {
	opts tun.ClientOptions
	cli  mqtt.Client
}

func (c *client) Handler() tun.Handler {
	return c.opts.Handler
}

func genUniq() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func genClientID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, genUniq())
}

func (c *client) DialAndServe() error {
	token := c.cli.Connect()
	token.Wait()
	err := token.Error()
	if err != nil {
		return err
	}

	tunIDChan := make(chan int64)

	uniq := genUniq()
	addr := newAddr(c.opts.Address)
	connectedTopic := fmt.Sprintf("%s/uniq/%s", addr.path(), uniq)
	token = c.cli.Subscribe(connectedTopic, 0, func(mqttCli mqtt.Client, m mqtt.Message) {
		t := mqttCli.Unsubscribe(connectedTopic)
		t.Wait()

		tunID, err := strconv.ParseInt(string(m.Payload()), 10, 64)
		if err != nil {
			// TODO: log here
			fmt.Println("!!", err)
			return
		}

		tunIDChan <- tunID
		close(tunIDChan)
	})
	token.Wait()
	err = token.Error()
	if err != nil {
		fmt.Println("!!", err)
		return err
	}
	// fmt.Println("sub", connectedTopic)

	listenerTopic := fmt.Sprintf("%s/listener", addr.path())
	token = c.cli.Publish(listenerTopic, 0, false, uniq)
	token.Wait()
	err = token.Error()
	if err != nil {
		fmt.Println("!!", err)
		return err
	}
	// fmt.Println("pub", listenerTopic, uniq)

	tunID := <-tunIDChan
	if h := c.Handler(); h != nil {
		srvTopic := fmt.Sprintf("%s/srv/%d", addr.path(), tunID)
		cliTopic := fmt.Sprintf("%s/cli/%d", addr.path(), tunID)

		tunr := newReader(c.cli, cliTopic)
		tunw := newWriter(c.cli, srvTopic)
		// ctx := context.WithValue(context.Background(), tun.TunIDContextKey{}, tunID)
		ctx := context.Background()

		tunr.RunPipe()
		h.ServeTun(ctx, tunr, tunw)
		tunr.Close()
	}

	return nil
}

func NewClient(opts ...tun.ClientOption) tun.Client {
	opt := tun.NewClientOptions(opts...)

	c := &client{
		opts: *opt,
	}
	addr := newAddr(opt.Address)
	mqttOpts := mqtt.NewClientOptions()
	mqttOpts.AddBroker(fmt.Sprintf("%s://%s:%s", addr.scheme(), addr.host(), addr.port()))
	mqttOpts.SetClientID(genClientID("tnet_tun_cli"))
	if addr.user() != "" {
		mqttOpts.SetUsername(addr.user())
		mqttOpts.SetPassword(addr.pass())
	}

	mqttOpts.SetOnConnectHandler(func(c mqtt.Client) {
	}).SetConnectionLostHandler(func(c mqtt.Client, err error) {
	})

	cli := mqtt.NewClient(mqttOpts)
	c.cli = cli

	return c
}
