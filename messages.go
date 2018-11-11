package main

import (
	"encoding/json"
)

type Message struct {
	Data     map[string]string
	Event    string
	Channel  string
	SocketID string
}

func (m Message) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"event":     m.Event,
		"data":      m.Data,
		"channel":   m.Channel,
		"socket_id": m.SocketID,
	})
}

const SubscribeEvent = "pusher:subscribe"

type SubscribeMessage struct {
	Channel string
}

const UnsubscribeEvent = "pusher:unsubscribe"

type UnsubscribeMessage struct {
	Channel string
}

const PingEvent = "pusher:ping"
