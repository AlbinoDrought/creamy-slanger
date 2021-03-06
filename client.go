package main

import (
	"crypto/rand"
	"encoding/json"
	"math"
	"math/big"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Client struct {
	toClientChannel chan []byte

	SocketID  string
	Handshake string
	// a client can subscribe to channels
	Subscriptions map[string]*Subscription
}

func NewClient() *Client {
	// TODO: generate socket id
	client := &Client{
		toClientChannel: make(chan []byte),
		Subscriptions:   make(map[string]*Subscription),
	}

	client.generateSocketID()

	go func() {
		msg, _ := json.Marshal(map[string]interface{}{
			"event": "pusher:connection_established",
			"data": map[string]interface{}{
				"socket_id":        client.SocketID,
				"activity_timeout": options.ActivityTimeout,
			},
		})
		client.toClientChannel <- msg
	}()

	log.Debugf("[client %v] connected", client.SocketID)

	return client
}

func (c *Client) generateSocketID() {
	pid := strconv.Itoa(os.Getpid())
	r, _ := rand.Int(rand.Reader, big.NewInt(int64(math.Pow(10, 6))))

	c.SocketID = pid + "." + r.String()
}

func (c *Client) Close() {
	for channel := range c.Subscriptions {
		c.Unsubscribe(channel)
	}
}

// Messages channel
func (c *Client) Messages() chan []byte {
	return c.toClientChannel
}

// Subscribe to a channel
func (c *Client) Subscribe(channel string) error {
	if _, ok := c.Subscriptions[channel]; ok {
		// already subscribed
		log.Debugf("[client %v] attempted to re-subscribe: %v", c.SocketID, channel)
		return nil
	}

	subscription, err := NewSubscription(channel)
	if err != nil {
		return err
	}

	c.Subscriptions[channel] = subscription

	// poll messages
	go func() {
		for message := range subscription.Messages() {
			c.toClientChannel <- []byte(message)
		}
	}()

	// {"event":"pusher_internal:subscription_succeeded","data":"{}","channel":"my-channel"}
	msg, _ := json.Marshal(map[string]interface{}{
		"event":   "pusher_internal:subscription_succeeded",
		"data":    map[string]string{},
		"channel": channel,
	})
	subscription.Receive(msg)
	log.Debugf("[client %v] subscribed to %v", c.SocketID, channel)

	return nil
}

func (c *Client) Unsubscribe(channel string) {
	subscription, ok := c.Subscriptions[channel]
	if !ok {
		// not subscribed
		log.Debugf("[client %v] attempted to unsubscribe but not subscribed: %v", c.SocketID, channel)
		return
	}
	delete(c.Subscriptions, channel)
	subscription.Unsubscribe()
	log.Debugf("[client %v] unsubscribed from %v", c.SocketID, channel)
}

func (c Client) OnMessageFromClient(message *Message) {
	if message.Event == SubscribeEvent {
		subscribeMessage := SubscribeMessage{
			Channel: message.Data["channel"],
		}

		err := c.Subscribe(subscribeMessage.Channel)
		if err != nil {
			log.Warnf("[client %v] error subscribing to %v: %+v", c.SocketID, subscribeMessage.Channel, err)
		}
	} else if message.Event == UnsubscribeEvent {
		unsubscribeMessage := UnsubscribeMessage{
			Channel: message.Data["channel"],
		}

		c.Unsubscribe(unsubscribeMessage.Channel)
	} else if message.Event == PingEvent {
		c.Pong()
	} else {
		log.Debugf("[client %v] told us %v %+v", c.SocketID, message.Event, message.Data)
	}
}

func (c Client) Pong() {
	msg, _ := json.Marshal(map[string]interface{}{
		"event": "pusher:pong",
		"data":  map[string]string{},
	})

	c.toClientChannel <- msg
	log.Debugf("[client %v] ping -> pong", c.SocketID)
}

func (c Client) AppKey() string {
	// TODO: implement
	return "foo"
}

func (c Client) ValidAppKey() bool {
	return c.AppKey() == options.AppKey
}

func (c Client) ProtocolVersion() int {
	// TODO: implement
	return 5
}

func (c Client) ValidProtocolVersion() bool {
	protocolVersion := c.ProtocolVersion()
	return protocolVersion > 3 && protocolVersion < 7
}
