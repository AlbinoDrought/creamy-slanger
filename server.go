package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/AlbinoDrought/creamy-slanger/websockets"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func serveWs(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	defer r.Body.Close()
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Warnf("[server] err on ws upgrade: %+v", err)
		}
		return
	}
	ws.SetReadLimit(512)
	defer ws.Close()

	con := newWebsocketConnection(ws, p.ByName("appkey"))
	defer slangerOptions.Handler.OnClose(con)

	err = slangerOptions.Handler.OnOpen(con)
	if err != nil {
		slangerOptions.Handler.OnError(con, err)
		return
	}

	var (
		messageType int
		rawMessage  []byte
		message     websockets.ClientMessagePayload
	)
	for {
		messageType, rawMessage, err = ws.ReadMessage()
		if err != nil {
			log.Debugf("[client %v] error on read: %+v", con.SocketID(), err)
			break
		}
		if messageType == websocket.TextMessage {
			message = websockets.ClientMessagePayload{}
			err = json.Unmarshal(rawMessage, &message)

			if err != nil {
				log.Warnf("[client %v] error on unmarshal: %+v", con.SocketID(), err)
				break
			}

			log.Debugf("[client %v] message received: %+v", con.SocketID(), message)
			if err = slangerOptions.Handler.OnMessage(con, message); err != nil {
				slangerOptions.Handler.OnError(con, err)
			}
		} else {
			log.Warnf("[client %v] unhandled message type: %v, %v", con.SocketID(), messageType, rawMessage)
		}
	}
}

type IncomingEvent struct {
	Name     string
	Channels []string
	Data     websockets.MessagePayload
}

func createEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	appID := ps.ByName("appid")
	app := slangerOptions.AppManager.FindByID(appID)

	if app == nil {
		w.WriteHeader(404)
		return
	}

	event := IncomingEvent{}
	json.Unmarshal(body, &event)

	for _, channelName := range event.Channels {
		log.Debugf("[channel %v] publishing %v %+v", channelName, event.Name, event.Data)
		slangerOptions.EventManager.Publish(
			appID,
			channelName,
			websockets.PubEvent{
				Payload: event.Data,
			},
		)
	}
}

func bootServer() {
	router := httprouter.New()
	router.GET("/app/:appkey", serveWs)
	router.POST("/apps/:appid/events", createEvent)

	addr := options.WebsocketHost + ":" + options.WebsocketPort
	log.Infof("[server] listening on %v", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
