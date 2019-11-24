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
	appKey := p.ByName("appkey")
	logger := log.WithField("appKey", appKey)

	defer r.Body.Close()
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			logger.WithField("error", err).Warn("ws upgrade failed")
		}
		return
	}
	ws.SetReadLimit(512)
	defer ws.Close()

	con := newWebsocketConnection(ws, appKey)
	defer slangerOptions.Handler.OnClose(con)

	err = slangerOptions.Handler.OnOpen(con)
	if err != nil {
		slangerOptions.Handler.OnError(con, err)
		return
	}

	logger = logger.WithField("client", con.SocketID())

	var (
		messageType int
		rawMessage  []byte
		message     websockets.ClientMessagePayload
	)
	for {
		messageType, rawMessage, err = ws.ReadMessage()
		if err != nil {
			logger.WithField("error", err).Debug("error on read")
			break
		}
		if messageType == websocket.TextMessage {
			message = websockets.ClientMessagePayload{}
			err = json.Unmarshal(rawMessage, &message)

			if err != nil {
				logger.WithField("error", err).Debug("error on unmarshal")
				break
			}

			logger.WithField("msg", message).Debug("inbound")
			if err = slangerOptions.Handler.OnMessage(con, message); err != nil {
				logger.WithFields(log.Fields{
					"msg":   message,
					"error": err,
				}).Debug("message error")
				slangerOptions.Handler.OnError(con, err)
			}
		} else {
			logger.WithFields(log.Fields{
				"type": messageType,
				"raw":  rawMessage,
			}).Debug("unhandled message type")
		}
	}
}

type IncomingEvent struct {
	Name     string
	Channels []string
	Data     interface{}
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
	messagePayload := websockets.MessagePayload{}
	if err := json.Unmarshal(body, &event); err != nil {
		log.WithFields(log.Fields{
			"appID": appID,
			"error": err,
		}).Debug("createEvent json unmarshal failure")
	}

	if dataBytes, ok := event.Data.([]byte); ok {
		if err := json.Unmarshal(dataBytes, messagePayload); err != nil {
			log.WithFields(log.Fields{
				"appID": appID,
				"error": err,
			}).Debug("createEvent dataBytes json unmarshal failure")
		}
	}

	for _, channelName := range event.Channels {
		log.WithFields(log.Fields{
			"appID":   appID,
			"channel": channelName,
			"event":   event.Name,
			"message": event.Data,
		}).Debugf("publishing")

		slangerOptions.EventManager.Publish(
			appID,
			channelName,
			websockets.PubEvent{
				Payload: messagePayload,
			},
		)
	}
}

func bootServer() {
	router := httprouter.New()
	router.GET("/app/:appkey", serveWs)
	router.POST("/apps/:appid/events", createEvent)

	addr := options.WebsocketHost + ":" + options.WebsocketPort
	log.WithField("address", addr).Info("listening")
	log.Fatal(http.ListenAndServe(addr, router))
}
