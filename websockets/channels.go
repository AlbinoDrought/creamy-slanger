package websockets

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/AlbinoDrought/creamy-slanger/websockets/support"
	log "github.com/sirupsen/logrus"
)

// A Channel could be any type of event bus
type Channel interface {
	Open()
	Close()

	HasConnections() bool
	HasLocalConnections() bool
	GetSubscribedConnections() []Connection
	GetSubscriptionCount() int64
	Subscribe(con Connection, payload ClientMessagePayload) error
	Unsubscribe(con Connection)
	Broadcast(message MessagePayload)
	BroadcastToOthers(con Connection, message MessagePayload)
	BroadcastToEveryoneExcept(message MessagePayload, socketID string)
	ToArray() map[string]interface{}
}

// A publicChannel is a public and unsecured event bus
type publicChannel struct {
	lock sync.RWMutex

	cancel context.CancelFunc

	appID string
	name  string

	eventManager EventManager

	localConnections map[string]Connection
}

func (c *publicChannel) Open() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	go func() {
		subscription := c.eventManager.Subscribe(c.appID, c.name)
		defer subscription.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case pubEvent := <-subscription.Channel():
				c.lock.RLock()
				for socketID, connection := range c.localConnections {
					if socketID == pubEvent.Except {
						continue
					}

					connection.Send(pubEvent.Payload)
				}
				c.lock.RUnlock()
			}
		}
	}()
}

func (c *publicChannel) Close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	for socketID := range c.localConnections {
		c.eventManager.RemoveTrackedChannelSubscriber(c.appID, c.name, socketID)
	}
}

// HasConnections returns true if at least one client is subscribed to this channel anywhere
func (c *publicChannel) HasConnections() bool {
	return c.GetSubscriptionCount() > 0
}

// HasLocalConnections returns true if at least one client is subscribed to this channel locally
func (c *publicChannel) HasLocalConnections() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.localConnections) > 0
}

// GetSubscribedConnections returned all instances of subscribed connections
func (c *publicChannel) GetSubscribedConnections() []Connection {
	// todo: implement
	return []Connection{}
}

// GetSubscriptionCount returns the count of connections subscribed to this channel
func (c *publicChannel) GetSubscriptionCount() int64 {
	count, err := c.eventManager.GetTrackedChannelSubscriberCount(c.appID, c.name)
	if err != nil {
		// todo: log or handle
	}
	return count
}

func (c *publicChannel) verifySignature(con Connection, payload ClientMessagePayload) error {
	signature := con.SocketID() + ":" + c.name

	channelData := payload.ChannelData()
	if channelData != "" {
		signature += ":" + channelData
	}

	auth := payload.Auth()
	suppliedMAC := support.StrAfter(auth, ":")

	mac := hmac.New(sha256.New, []byte(con.App().Secret()))
	mac.Write([]byte(signature))
	computedMAC := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(suppliedMAC), []byte(computedMAC)) {
		return invalidSignatureException()
	}

	return nil
}

func (c *publicChannel) saveConnection(con Connection) {
	c.lock.Lock()
	hadConnectionsPreviously := c.HasConnections()
	c.localConnections[con.SocketID()] = con
	c.eventManager.AddTrackedChannelSubscriber(c.appID, c.name, con.SocketID())
	c.lock.Unlock()

	// todo: implement
	if !hadConnectionsPreviously {
		// track occupied
	}

	// track subscribed
}

// Subscribe a connection to this channel
func (c *publicChannel) Subscribe(con Connection, payload ClientMessagePayload) error {
	c.saveConnection(con)
	con.Send(map[string]interface{}{
		"event":   "pusher_internal:subscription_succeeded",
		"channel": c.name,
	})

	log.WithFields(log.Fields{
		"appID":       c.appID,
		"channel":     c.name,
		"subscribers": c.GetSubscriptionCount(),
	}).Debug("local channel sub")

	return nil
}

// Unsubscribe a connection from this channel
func (c *publicChannel) Unsubscribe(con Connection) {
	c.lock.Lock()
	delete(c.localConnections, con.SocketID())
	c.eventManager.RemoveTrackedChannelSubscriber(c.appID, c.name, con.SocketID())
	c.lock.Unlock()

	log.WithFields(log.Fields{
		"appID":       c.appID,
		"channel":     c.name,
		"subscribers": c.GetSubscriptionCount(),
	}).Debug("local channel unsub")

	// todo: implement
	if !c.HasConnections() {
		// track vacated
	}
}

// Broadcast a message to all connections on this channel
func (c *publicChannel) Broadcast(message MessagePayload) {
	c.eventManager.Publish(
		c.appID,
		c.name,
		PubEvent{
			Payload: message,
		},
	)
}

// BroadcastToOthers sends a message to all connections on this channel except
// for the specified connection instance.
func (c *publicChannel) BroadcastToOthers(con Connection, message MessagePayload) {
	c.BroadcastToEveryoneExcept(message, con.SocketID())
}

// BroadcastToEveryoneExcept sends a message to all connections on this channel
// except for the specified socket ID.
func (c *publicChannel) BroadcastToEveryoneExcept(message MessagePayload, socketID string) {
	c.eventManager.Publish(
		c.appID,
		c.name,
		PubEvent{
			Payload: message,
			Except:  socketID,
		},
	)
}

// ToArray transmogrifies this channel to a serializable array
func (c *publicChannel) ToArray() map[string]interface{} {
	return map[string]interface{}{
		"occupied":           c.HasConnections(),
		"subscription_count": c.GetSubscriptionCount(),
	}
}

func newPublicChannel(appID, name string, eventManager EventManager) *publicChannel {
	return &publicChannel{
		appID:        appID,
		name:         name,
		eventManager: eventManager,

		lock:             sync.RWMutex{},
		localConnections: map[string]Connection{},
	}
}

// NewPublicChannel returns a new public and unsecured event bus
func NewPublicChannel(appID, name string, eventManager EventManager) Channel {
	return newPublicChannel(appID, name, eventManager)
}

// A presenceChannel is a private and secured event bus that keeps track of
// users connected to it.
type presenceChannel struct {
	*publicChannel

	localUserData map[string]string
}

func (c *presenceChannel) getChannelDataAsString() string {
	users, err := c.eventManager.GetTrackedChannelUsers(c.appID, c.name)
	if err != nil {
		// todo: log or handle
		return ""
	}

	userHash := make(map[interface{}]interface{}, len(users))
	userIDs := make([]interface{}, len(users))

	for i, userData := range users {
		decoded := map[string]interface{}{}
		err := json.Unmarshal([]byte(userData), &decoded)
		if err != nil {
			// todo: log or handle
			continue
		}

		userID, hasID := decoded["user_id"]
		userInfo, hasUserInfo := decoded["user_info"]

		if !hasID || !hasUserInfo {
			// todo: log or handle
			continue
		}

		userHash[userID] = userInfo
		userIDs[i] = userID
	}

	channelData := map[string]interface{}{
		"presence": map[string]interface{}{
			"ids":   userIDs,
			"hash":  userHash,
			"count": c.getUserCount(),
		},
	}
	channelDataAsBytes, err := json.Marshal(channelData)
	if err != nil {
		// todo: log or handle
		return ""
	}
	return string(channelDataAsBytes)
}

func (c *presenceChannel) getUserCount() int64 {
	count, err := c.eventManager.GetTrackedChannelUserCount(c.appID, c.name)
	if err != nil {
		// todo: log or handle
	}
	return count
}

func (c *presenceChannel) Close() {
	c.publicChannel.Close()

	c.lock.Lock()
	defer c.lock.Unlock()
	for socketID := range c.localUserData {
		c.eventManager.RemoveTrackedChannelUser(c.appID, c.name, socketID)
	}
}

// Subscribe a connection to this channel
func (c *presenceChannel) Subscribe(con Connection, payload ClientMessagePayload) error {
	if err := c.verifySignature(con, payload); err != nil {
		return err
	}

	c.saveConnection(con)

	userData := payload.ChannelData()
	c.lock.Lock()
	c.localUserData[con.SocketID()] = userData
	c.eventManager.AddTrackedChannelUser(c.appID, c.name, con.SocketID(), userData)
	c.lock.Unlock()

	channelData := c.getChannelDataAsString()

	con.Send(map[string]interface{}{
		"event":   "pusher_internal:subscription_succeeded",
		"channel": c.name,
		"data":    channelData,
	})

	c.BroadcastToOthers(con, map[string]interface{}{
		"event":   "pusher_internal:member_added",
		"channel": c.name,
		"data":    userData,
	})

	return nil
}

// Unsubscribe a connection from this channel
func (c *presenceChannel) Unsubscribe(con Connection) {
	c.publicChannel.Unsubscribe(con)

	c.lock.RLock()
	userData, ok := c.localUserData[con.SocketID()]
	c.lock.RUnlock()

	if !ok {
		// already unsubbed
		return
	}

	c.BroadcastToOthers(con, map[string]interface{}{
		"event":   "pusher_internal:member_removed",
		"channel": c.name,
		"data":    userData, // stray: uses entire userData instead of just user_id
	})

	c.lock.Lock()
	delete(c.localUserData, con.SocketID())
	c.eventManager.RemoveTrackedChannelUser(c.appID, c.name, con.SocketID())
	c.lock.Unlock()
}

// ToArray transmogrifies this channel to a serializable array
func (c *presenceChannel) ToArray() map[string]interface{} {
	array := c.publicChannel.ToArray()
	array["user_count"] = c.getUserCount()
	return array
}

// NewPresenceChannel returns a private and secured event bus that keeps track of
// users connected to it.
func NewPresenceChannel(appID, name string, eventManager EventManager) Channel {
	return &presenceChannel{
		publicChannel: newPublicChannel(appID, name, eventManager),
	}
}

// A privateChannel is a private and secured event bus
type privateChannel struct {
	*publicChannel
}

// Subscribe a connection to this channel
func (c *privateChannel) Subscribe(con Connection, payload ClientMessagePayload) error {
	if err := c.verifySignature(con, payload); err != nil {
		return err
	}

	return c.publicChannel.Subscribe(con, payload)
}

// NewPrivateChannel returns a private and secured event bus
func NewPrivateChannel(appID, name string, eventManager EventManager) Channel {
	return &privateChannel{
		publicChannel: newPublicChannel(appID, name, eventManager),
	}
}
