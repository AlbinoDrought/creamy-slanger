package main

import (
	"context"
	"log"
	"sync"

	"github.com/go-redis/redis"
)

var (
	channels = make(map[string]*Channel)
)

// A ChannelSubscriber receives messages from a Channel
type ChannelSubscriber interface {
	Receive(message []byte)
}

// A Channel handles Redis connections and the passing of messages
// to ChannelSubscribers
type Channel struct {
	subscribed *sync.Mutex
	cancel     context.CancelFunc

	Name        string
	Subscribers map[*Subscription]bool
}

// GetChannel gets or creates a channel in the global space
func GetChannel(name string) (*Channel, error) {
	// if channel already exists, return it
	if channel, ok := channels[name]; ok {
		return channel, nil
	}

	// otherwise, make one
	channel := &Channel{
		Name:        name,
		Subscribers: make(map[*Subscription]bool),
		subscribed:  &sync.Mutex{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	channel.cancel = cancel
	err := channel.register(ctx, daddy)

	if err != nil {
		return nil, err
	}

	channels[name] = channel

	return channel, nil
}

func (c *Channel) register(ctx context.Context, client *redis.Client) error {
	c.subscribed.Lock()
	pubsub := client.Subscribe(c.Name)

	// test receive
	_, err := pubsub.Receive()
	if err != nil {
		c.subscribed.Unlock()
		pubsub.Close()
		return err
	}

	go func() {
		defer c.subscribed.Unlock()
		defer pubsub.Close()

		for {
			select {
			case <-ctx.Done():
				// stop
				return
			case msg := <-pubsub.Channel():
				c.Dispatch(msg.Payload)
			}
		}

	}()

	return nil
}

// Close the channel and release redis resources
func (c Channel) Close() {
	c.cancel()
	// wait for subscriber to die
	c.subscribed.Lock()
	c.subscribed.Unlock()
}

// Subscribe to this channel
func (c *Channel) Subscribe(subscriber *Subscription) {
	if c.Subscribers[subscriber] {
		// already subscribed
		return
	}

	c.Subscribers[subscriber] = true
}

// Unsubscribe from this channel
func (c *Channel) Unsubscribe(subscriber *Subscription) {
	log.Printf("subscribers: %+v", c.Subscribers)
	if !c.Subscribers[subscriber] {
		// not subscribed
		log.Printf("attempted to unsubscribe but not subscribed: %p", subscriber)
		return
	}

	delete(c.Subscribers, subscriber)
}

// Dispatch a message to all registered subscribers
func (c Channel) Dispatch(message string) {
	byteMessage := []byte(message)
	for subscription := range c.Subscribers {
		subscription.Receive(byteMessage)
	}
}
