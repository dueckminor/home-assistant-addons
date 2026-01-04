package mqtt

import (
	"context"
	"crypto/tls"
	"io"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTClientConfig struct {
	URI      string `yaml:"uri"`
	ClientID string `yaml:"client_id"`
}

type Broker interface {
	Dial(clientId string, statusTopic string) (Conn, error)
}

type Conn interface {
	io.Closer
	Publish(topic string, payload string)
	PublishRetain(topic string, payload string)
	Subscribe(topic string, cb func(topic string, payload string))
	SubscribeCtx(ctx context.Context, topic string, cb func(topic string, payload string))
	Forward(topic string, target Conn)
}

type broker struct {
	tlsConfig *tls.Config
	uri       string
	username  string
	password  string
}

func (b *broker) Dial(clientId string, statusTopic string) (Conn, error) {
	c := &conn{}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.uri)
	opts.Password = b.password
	opts.Username = b.username
	opts.SetClientID(clientId).SetTLSConfig(b.tlsConfig)
	if statusTopic != "" {
		opts.SetWill(statusTopic, "offline", 0, true)
	}
	c.client = mqtt.NewClient(opts)
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	if statusTopic != "" {
		c.PublishRetain(statusTopic, "online")
	}
	return c, nil
}

type conn struct {
	client mqtt.Client
}

func (c *conn) Close() error {
	c.client.Disconnect(500)
	return nil
}

func (c *conn) Publish(topic string, payload string) {
	c.client.Publish(topic, 0, false, payload)
}

func (c *conn) PublishRetain(topic string, payload string) {
	c.client.Publish(topic, 2, true, payload)
}

func (c *conn) Subscribe(topic string, cb func(topic string, payload string)) {
	c.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		cb(msg.Topic(), string(msg.Payload()))
	})
}

func (c *conn) SubscribeCtx(ctx context.Context, topic string, cb func(topic string, payload string)) {
	c.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		cb(msg.Topic(), string(msg.Payload()))
	})
	<-ctx.Done()
	c.client.Unsubscribe(topic)
}

func (c *conn) Forward(topic string, target Conn) {
	targetClient := target.(*conn).client

	c.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		topic := msg.Topic()
		if strings.HasPrefix(topic, "$SYS") {
			return
		}
		targetClient.Publish(msg.Topic(), msg.Qos(), msg.Retained(), msg.Payload())
	})
}

func NewBroker(uri string, user string, password string) Broker {
	b := &broker{}
	b.uri = uri
	b.username = user
	b.password = password
	return b
}
