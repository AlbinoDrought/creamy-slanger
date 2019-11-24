package websockets

import (
	"crypto/hmac"
	"crypto/sha256"

	"github.com/AlbinoDrought/creamy-slanger/websockets/support"
)

// A Channel could be any type of event bus
type Channel interface {
	HasConnections() bool
	GetSubscribedConnections() []Connection
	GetSubscriptionCount() int
	Subscribe(con Connection, payload ClientMessagePayload) error
	Unsubscribe(con Connection)
	Broadcast(message MessagePayload)
	BroadcastToOthers(con Connection, message MessagePayload)
	BroadcastToEveryoneExcept(message MessagePayload, socketID string)
	ToArray() map[string]interface{}
}

// A PublicChannel is a public and unsecured event bus
type PublicChannel struct {
	name string
}

// HasConnections returns true if at least one client is subscribed to this channel
func (c *PublicChannel) HasConnections() bool {
	return c.GetSubscriptionCount() > 0
}

// GetSubscribedConnections returned all instances of subscribed connections
func (c *PublicChannel) GetSubscribedConnections() []Connection {
	// todo: implement
	return []Connection{}
}

// GetSubscriptionCount returns the count of connections subscribed to this channel
func (c *PublicChannel) GetSubscriptionCount() int {
	// todo: implement
	return 0
}

func (c *PublicChannel) verifySignature(con Connection, payload ClientMessagePayload) error {
	signature := con.SocketID() + ":" + c.name

	channelData := payload.ChannelData()
	if channelData != "" {
		signature += ":" + channelData
	}

	auth := payload.Auth()
	actualMAC := []byte(support.StrAfter(auth, ":"))

	mac := hmac.New(sha256.New, []byte(con.App().Secret()))
	mac.Write([]byte(signature))
	expectedMAC := mac.Sum(nil)

	if !hmac.Equal(actualMAC, expectedMAC) {
		return invalidSignatureException()
	}

	return nil
}

func (c *PublicChannel) saveConnection(con Connection) {
	// todo: implement
	hadConnectionsPreviously := c.HasConnections()

	// add connection

	if !hadConnectionsPreviously {
		// track occupied
	}

	// track subscribed
}

// Subscribe a connection to this channel
func (c *PublicChannel) Subscribe(con Connection, payload ClientMessagePayload) error {
	c.saveConnection(con)
	con.Send(map[string]interface{}{
		"event":   "pusher_internal:subscription_succeeded",
		"channel": c.name,
	})

	return nil
}

// Unsubscribe a connection from this channel
func (c *PublicChannel) Unsubscribe(con Connection) {
	// todo: implement

	// remove connection

	if !c.HasConnections() {
		// track vacated
	}
}

// Broadcast a message to all connections on this channel
func (c *PublicChannel) Broadcast(message MessagePayload) {
	// todo: implement
}

// BroadcastToOthers sends a message to all connections on this channel except
// for the specified connection instance.
func (c *PublicChannel) BroadcastToOthers(con Connection, message MessagePayload) {
	c.BroadcastToEveryoneExcept(message, con.SocketID())
}

// BroadcastToEveryoneExcept sends a message to all connections on this channel
// except for the specified socket ID.
func (c *PublicChannel) BroadcastToEveryoneExcept(message MessagePayload, socketID string) {
	if socketID == "" {
		c.Broadcast(message)
		return
	}

	// todo: implement
}

// ToArray transmogrifies this channel to a serializable array
func (c *PublicChannel) ToArray() map[string]interface{} {
	return map[string]interface{}{
		"occupied":           c.HasConnections(),
		"subscription_count": c.GetSubscriptionCount(),
	}
}

// A PresenceChannel is a private and secured event bus that keeps track of
// users connected to it.
type PresenceChannel struct {
	*PublicChannel
}

func (c *PresenceChannel) getChannelDataAsString() string {
	// todo: implement
	return ""
}

func (c *PresenceChannel) getUserCount() int {
	// todo: implement
	return 0
}

// Subscribe a connection to this channel
func (c *PresenceChannel) Subscribe(con Connection, payload ClientMessagePayload) error {
	if err := c.verifySignature(con, payload); err != nil {
		return err
	}

	c.saveConnection(con)

	// todo: implement

	// save payload->channel_data as user data

	channelData := c.getChannelDataAsString()

	con.Send(map[string]interface{}{
		"event":   "pusher_internal:subscription_succeeded",
		"channel": c.name,
		"data":    channelData,
	})

	c.BroadcastToOthers(con, map[string]interface{}{
		"event":   "pusher_internal:member_added",
		"channel": c.name,
		"data":    channelData,
	})

	return nil
}

// Unsubscribe a connection from this channel
func (c *PresenceChannel) Unsubscribe(con Connection) {
	c.PublicChannel.Unsubscribe(con)

	// todo: implement

	// jump out if user already unsubbed

	c.BroadcastToOthers(con, map[string]interface{}{
		"event":   "pusher_internal:member_removed",
		"channel": c.name,
		"data":    "", // todo: implement
	})

	// remove user from list of users
}

// ToArray transmogrifies this channel to a serializable array
func (c *PresenceChannel) ToArray() map[string]interface{} {
	array := c.PublicChannel.ToArray()
	array["user_count"] = c.getUserCount()
	return array
}

// A PrivateChannel is a private and secured event bus
type PrivateChannel struct {
	*PublicChannel
}

// Subscribe a connection to this channel
func (c *PrivateChannel) Subscribe(con Connection, payload ClientMessagePayload) error {
	if err := c.verifySignature(con, payload); err != nil {
		return err
	}

	return c.PublicChannel.Subscribe(con, payload)
}
