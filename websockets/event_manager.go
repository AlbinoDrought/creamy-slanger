package websockets

import (
	"context"
	"encoding/json"
	"strings"
	"time"

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

	KeepSubscriber(appID, subscriberID string) error
	SweepSubscriber(subscriberID string) error
	SweepExpired() (uint64, error)
}

type redisEventManager struct {
	client *redis.Client
}

func (eventManager *redisEventManager) pubsubKey(appID, channel string) string {
	return appID + "_ " + channel
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
	eventManager.client.SAdd("sub:"+subscriberID+"_subs", appID+":"+channel)

	return eventManager.client.SAdd(eventManager.trackedChannelSubscriberKey(appID, channel), subscriberID).Err()
}

func (eventManager *redisEventManager) RemoveTrackedChannelSubscriber(appID, channel, subscriberID string) error {
	eventManager.client.SRem("sub:"+subscriberID+"_subs", appID+":"+channel)

	return eventManager.client.SRem(eventManager.trackedChannelSubscriberKey(appID, channel), subscriberID).Err()
}

func (eventManager *redisEventManager) GetTrackedChannelSubscriberCount(appID, channel string) (int64, error) {
	return eventManager.client.SCard(eventManager.trackedChannelSubscriberKey(appID, channel)).Result()
}

func (eventManager *redisEventManager) trackedChannelUserKey(appID, channel string) string {
	return "users:" + eventManager.pubsubKey(appID, channel)
}

func (eventManager *redisEventManager) AddTrackedChannelUser(appID, channel, subscriberID, userData string) error {
	eventManager.client.SAdd("sub:"+subscriberID+"_users", appID+":"+channel)

	return eventManager.client.HSet(eventManager.trackedChannelUserKey(appID, channel), subscriberID, userData).Err()
}

func (eventManager *redisEventManager) RemoveTrackedChannelUser(appID, channel, subscriberID string) error {
	eventManager.client.SRem("sub:"+subscriberID+"_users", appID+":"+channel)

	return eventManager.client.HDel(eventManager.trackedChannelUserKey(appID, channel), subscriberID).Err()
}

func (eventManager *redisEventManager) GetTrackedChannelUsers(appID, channel string) ([]string, error) {
	return eventManager.client.HVals(eventManager.trackedChannelUserKey(appID, channel)).Result()
}

func (eventManager *redisEventManager) GetTrackedChannelUserCount(appID, channel string) (int64, error) {
	return eventManager.client.HLen(eventManager.trackedChannelUserKey(appID, channel)).Result()
}

func (eventManager *redisEventManager) KeepSubscriber(appID, subscriberID string) error {
	eventManager.client.SAdd("subs", subscriberID)

	return eventManager.client.Set("sub:"+subscriberID+"_alive", true, time.Hour).Err()
}

func (eventManager *redisEventManager) SweepSubscriber(subscriberID string) error {
	subscriptions := eventManager.client.SMembers("sub:" + subscriberID + "_subs").Val()
	for _, subscription := range subscriptions {
		index := strings.Index(subscription, ":")
		if index != -1 {
			appID := subscription[:index]
			channel := subscription[index+1:]

			eventManager.RemoveTrackedChannelSubscriber(appID, channel, subscriberID)
		}
		eventManager.client.SRem("sub:"+subscriberID+"_subs", subscription)
	}

	users := eventManager.client.SMembers("sub:" + subscriberID + "_users").Val()
	for _, user := range users {
		index := strings.Index(user, ":")
		if index != -1 {
			appID := user[:index]
			channel := user[index+1:]

			eventManager.RemoveTrackedChannelUser(appID, channel, subscriberID)
		}
		eventManager.client.SRem("sub:"+subscriberID+"_users", user)
	}

	eventManager.client.Del("sub:" + subscriberID + "_alive")
	return eventManager.client.SRem("subs", subscriberID).Err()
}

func (eventManager *redisEventManager) SweepExpired() (uint64, error) {
	var (
		subscribers []string
		cursor      uint64
		err         error
		sweeped     uint64
	)

	for {
		subscribers, cursor, err = eventManager.client.SScan("subs", cursor, "", 0).Result()
		if err != nil {
			return sweeped, err
		}

		for _, subscriber := range subscribers {
			if eventManager.client.Exists("sub:"+subscriber+"_alive").Val() <= 0 {
				eventManager.SweepSubscriber(subscriber)
				sweeped++
			}
		}

		if cursor <= 0 {
			break
		}
	}

	return sweeped, nil
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
