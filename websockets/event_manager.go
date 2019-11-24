package websockets

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis"
)

// A PubEvent is a message that has been published over pubsub
type PubEvent struct {
	Payload MessagePayload
	Except  string
}

// A Subscription receives messages from a channel
type Subscription interface {
	Close() error
	Channel() <-chan PubEvent
}

// EventManager manages cross-instance event pubsub
type EventManager interface {
	Publish(appID, channel string, payload PubEvent) error
	Subscribe(appID, channel string) Subscription
}

type redisEventManager struct {
	client *redis.Client
}

func (eventManager *redisEventManager) channelName(appID, channel string) string {
	return appID + "\x00" + channel
}

func (eventManager *redisEventManager) Publish(appID, channel string, payload PubEvent) error {
	serialized, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return eventManager.client.Publish(eventManager.channelName(appID, channel), serialized).Err()
}

func (eventManager *redisEventManager) Subscribe(appID, channel string) Subscription {
	subscription := eventManager.client.Subscribe(eventManager.channelName(appID, channel))

	ctx, cancel := context.WithCancel(context.Background())
	messageChannel := make(chan PubEvent)
	go func() {
		redisChannel := subscription.Channel()
		defer close(messageChannel)

		for {
			select {
			case <-ctx.Done():
				return
			case message := <-redisChannel:
				pubEvent := PubEvent{}
				if err := json.Unmarshal([]byte(message.Payload), &pubEvent); err != nil {
					// todo: log
					continue
				}

				messageChannel <- pubEvent
			}
		}
	}()

	return &redisSubscription{
		cancel:   cancel,
		channel:  messageChannel,
		redisSub: subscription,
	}
}

type redisSubscription struct {
	redisSub *redis.PubSub
	cancel   context.CancelFunc
	channel  <-chan PubEvent
}

func (subscription *redisSubscription) Close() error {
	subscription.cancel()
	return subscription.redisSub.Close()
}

func (subscription *redisSubscription) Channel() <-chan PubEvent {
	return subscription.channel
}

// NewRedisEventManager returns a new EventManager that uses redis pubsub
func NewRedisEventManager(client *redis.Client) EventManager {
	return &redisEventManager{
		client,
	}
}
