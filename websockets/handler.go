package websockets

import (
	"fmt"
	"math/rand"
)

type Handler interface {
	OnOpen(con Connection) error
	OnMessage(con Connection, message ClientMessagePayload)
	OnClose(con Connection)
	OnError(con Connection, err error)
}

type websocketHandler struct {
	appManager     AppManager
	channelManager ChannelManager
}

func (h *websocketHandler) verifyAppKey(con Connection) error {
	// todo: implement
	// get AppKey from query
	appKey := "pancakes"

	app := h.appManager.FindByKey(appKey)
	if app == nil {
		return unknownAppKeyException(appKey)
	}

	con.SetApp(app)

	return nil
}

func (h *websocketHandler) limitConcurrentConnections(con Connection) error {
	if !con.App().CapacityEnabled() {
		return nil
	}

	max := con.App().Capacity()
	current := h.channelManager.GetConnectionCount(con.App().ID())

	if current >= max {
		return connectionsOverCapacityException()
	}

	return nil
}

func (h *websocketHandler) generateSocketID(con Connection) {
	// todo: seed rand somewhere?
	socketID := fmt.Sprintf("%d.%d", rand.Intn(1000000000), rand.Intn(1000000000))
	con.SetSocketID(socketID)
}

func (h *websocketHandler) establishConnection(con Connection) {
	con.Send(map[string]interface{}{
		"event": "pusher:connection_established",
		"data": map[string]interface{}{
			"socket_id":        con.SocketID(),
			"activity_timeout": 30,
		},
	})

	// todo: implement

	// track connection
}

func (h *websocketHandler) OnOpen(con Connection) error {
	if err := h.verifyAppKey(con); err != nil {
		return err
	}

	if err := h.limitConcurrentConnections(con); err != nil {
		return err
	}

	h.generateSocketID(con)
	h.establishConnection(con)

	return nil
}

func (h *websocketHandler) OnMessage(con Connection, payload ClientMessagePayload) error {
	message := CreateForMessage(con, payload, h.channelManager)

	if err := message.Respond(); err != nil {
		return err
	}

	// todo: implement
	// track websocket message

	return nil
}

func (h *websocketHandler) OnClose(con Connection) {
	h.channelManager.RemoveFromAllChannels(con)

	// todo: implement
	// track disconnect
}

func (h *websocketHandler) OnError(con Connection, err error) {
	if websocketException, ok := err.(WebsocketException); ok {
		con.Send(websocketException.GetPayload())
	}
}
