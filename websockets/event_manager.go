package websockets

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
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

	AddTrackedChannelSubscriber(appID, channel, subscriberID string) error
	RemoveTrackedChannelSubscriber(appID, channel, subscriberID string) error
	GetTrackedChannelSubscriberCount(appID, channel string) (int64, error)

	AddTrackedChannelUser(appID, channel, subscriberID, userData string) error
	RemoveTrackedChannelUser(appID, channel, subscriberID string) error
	GetTrackedChannelUsers(appID, channel string) ([]string, error)
	GetTrackedChannelUserCount(appID, channel string) (int64, error)
}

type redisEventManager struct {
	client *redis.Client
}

func (eventManager *redisEventManager) pubsubKey(appID, channel string) string {
	return appID + "_" + channel
}

func (eventManager *redisEventManager) Publish(appID, channel string, payload PubEvent) error {
	serialized, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return eventManager.client.Publish(eventManager.pubsubKey(appID, channel), serialized).Err()
}

func (eventManager *redisEventManager) Subscribe(appID, channel string) Subscription {
	subscription := eventManager.client.Subscribe(eventManager.pubsubKey(appID, channel))

	ctx, cancel := context.WithCancel(context.Background())
	messageChannel := make(chan PubEvent)
	go func() {
		logger := log.WithFields(log.Fields{
			"appID":   appID,
			"channel": channel,
		})

		redisChannel := subscription.Channel()
		defer close(messageChannel)

		for {
			select {
			case <-ctx.Done():
				return
			case message := <-redisChannel:
				if message == nil {
					logger.Debug("nil message received")
					continue
				}

				pubEvent := PubEvent{}
				if err := json.Unmarshal([]byte(message.Payload), &pubEvent); err != nil {
					logger.WithField("error", err).Debug("redis message unmarshal failed")
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

func (eventManager *redisEventManager) trackedChannelSubscriberKey(appID, channel string) string {
	return "subs:" + eventManager.pubsubKey(appID, channel)
}

func (eventManager *redisEventManager) AddTrackedChannelSubscriber(appID, channel, subscriberID string) error {
	return eventManager.client.SAdd(eventManager.trackedChannelSubscriberKey(appID, channel), subscriberID).Err()
}

func (eventManager *redisEventManager) RemoveTrackedChannelSubscriber(appID, channel, subscriberID string) error {
	return eventManager.client.SRem(eventManager.trackedChannelSubscriberKey(appID, channel), subscriberID).Err()
}

func (eventManager *redisEventManager) GetTrackedChannelSubscriberCount(appID, channel string) (int64, error) {
	return eventManager.client.SCard(eventManager.trackedChannelSubscriberKey(appID, channel)).Result()
}

func (eventManager *redisEventManager) trackedChannelUserKey(appID, channel string) string {
	return "subs:" + eventManager.pubsubKey(appID, channel)
}

func (eventManager *redisEventManager) AddTrackedChannelUser(appID, channel, subscriberID, userData string) error {
	return eventManager.client.HSet(eventManager.trackedChannelUserKey(appID, channel), subscriberID, userData).Err()
}

func (eventManager *redisEventManager) RemoveTrackedChannelUser(appID, channel, subscriberID string) error {
	return eventManager.client.HDel(eventManager.trackedChannelUserKey(appID, channel), subscriberID).Err()
}

func (eventManager *redisEventManager) GetTrackedChannelUsers(appID, channel string) ([]string, error) {
	return eventManager.client.HVals(eventManager.trackedChannelUserKey(appID, channel)).Result()
}

func (eventManager *redisEventManager) GetTrackedChannelUserCount(appID, channel string) (int64, error) {
	return eventManager.client.HLen(eventManager.trackedChannelUserKey(appID, channel)).Result()
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
