package main

import (
	"sync"

	"github.com/AlbinoDrought/creamy-slanger/websockets"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type websocketConnection struct {
	messageLock sync.Mutex

	ws *websocket.Conn

	app      websockets.App
	appKey   string
	socketID string
}

func (con *websocketConnection) App() websockets.App {
	return con.app
}

func (con *websocketConnection) SetApp(app websockets.App) {
	con.app = app
}

func (con *websocketConnection) AppKey() string {
	return con.appKey
}

func (con *websocketConnection) SocketID() string {
	return con.socketID
}

func (con *websocketConnection) SetSocketID(socketID string) {
	con.socketID = socketID
}

func (con *websocketConnection) Send(message websockets.MessagePayload) {
	con.messageLock.Lock()
	defer con.messageLock.Unlock()

	log.WithFields(log.Fields{
		"client": con.SocketID(),
		"msg":    message,
	}).Debug("outbound")
	con.ws.WriteJSON(message)
}

func newWebsocketConnection(ws *websocket.Conn, appKey string) websockets.Connection {
	return &websocketConnection{
		ws:     ws,
		appKey: appKey,
	}
}
