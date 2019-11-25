package websockets

import (
	"strings"

	"github.com/AlbinoDrought/creamy-slanger/websockets/support"
	log "github.com/sirupsen/logrus"
)

// A Message is anything that can be sent by a client connection
type Message interface {
	Respond() error
}

// A ClientMessage is something a client wants to say to other clients
type ClientMessage struct {
	payload        ClientMessagePayload
	con            Connection
	channelManager ChannelManager
}

// Respond to a message spoken by a client connection
func (m *ClientMessage) Respond() error {
	if !strings.HasPrefix(m.payload.Event(), "client-") {
		return nil
	}

	if !m.con.App().ClientMessagesEnabled() {
		return nil
	}

	// todo: implement

	// track clientMessage

	channel := m.channelManager.Find(m.con.App().ID(), m.payload.Channel())
	if channel != nil {
		channel.BroadcastToOthers(m.con, m.payload.MessagePayload())
	}

	return nil
}

// A ChannelProtocolMessage is something sent by the client that will act on their connection
type ChannelProtocolMessage struct {
	payload        ClientMessagePayload
	con            Connection
	channelManager ChannelManager
	eventManager   EventManager
}

func (m *ChannelProtocolMessage) ping() error {
	m.eventManager.KeepSubscriber(m.con.App().ID(), m.con.SocketID())
	m.con.Send(map[string]interface{}{
		"event": "pusher:pong",
	})
	return nil
}

func (m *ChannelProtocolMessage) subscribe() error {
	channel := m.channelManager.FindOrCreate(m.con.App().ID(), m.payload.Channel())
	return channel.Subscribe(m.con, m.payload)
}

func (m *ChannelProtocolMessage) unsubscribe() error {
	// todo: maybe just "find"?
	channel := m.channelManager.FindOrCreate(m.con.App().ID(), m.payload.Channel())
	channel.Unsubscribe(m.con)
	return nil
}

// Respond to a channel protocol message and perform the desired action
func (m *ChannelProtocolMessage) Respond() error {
	eventName := support.StrAfter(m.payload.Event(), ":")

	switch eventName {
	case "ping":
		return m.ping()
	case "subscribe":
		return m.subscribe()
	case "unsubscribe":
		return m.unsubscribe()
	default:
		log.WithFields(log.Fields{
			"client": m.con.SocketID(),
			"event":  eventName,
		}).Debug("unhandled protocol message")
	}

	return nil
}

// CreateForMessage converts any arbitrary ClientMessagePayload into an actionable Message
func CreateForMessage(con Connection, payload ClientMessagePayload, channelManager ChannelManager, eventManager EventManager) Message {
	// stray: assumes message already decoded

	if strings.HasPrefix(payload.Event(), "pusher:") {
		return &ChannelProtocolMessage{
			payload,
			con,
			channelManager,
			eventManager,
		}
	}

	return &ClientMessage{
		payload,
		con,
		channelManager,
	}
}
