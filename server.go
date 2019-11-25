package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

var (
	websocketWaitGroup sync.WaitGroup
	websocketContext   context.Context
)

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

	websocketWaitGroup.Add(1)
	defer websocketWaitGroup.Done()

	requestContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-websocketContext.Done():
			ws.Close()
			return
		case <-requestContext.Done():
		}
	}()

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
	Data     string
}

func createEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	body, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	appID := ps.ByName("appid")
	app := slangerOptions.AppManager.FindByID(appID)

	if app == nil {
		w.WriteHeader(404)
		w.Write([]byte("App not found"))
		return
	}

	if !requestVerified(app.Secret(), r.Method, r.URL.Path, r.URL.Query(), body) {
		w.WriteHeader(401)
		w.Write([]byte("Invalid auth signature provided."))
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

	for _, channelName := range event.Channels {
		log.WithFields(log.Fields{
			"appID":   appID,
			"channel": channelName,
			"event":   event.Name,
			"message": messagePayload,
		}).Debugf("publishing")

		slangerOptions.EventManager.Publish(
			appID,
			channelName,
			websockets.PubEvent{
				Payload: map[string]interface{}{
					"channel": channelName,
					"event":   event.Name,
					"data":    event.Data,
				},
			},
		)
	}
}

func bootServer() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	wsContext, wsCancel := context.WithCancel(context.Background())
	websocketContext = wsContext

	sweeper := sync.WaitGroup{}

	go func() {
		sweeper.Add(1)
		sweepEvery(wsContext, time.Minute*10)
		sweeper.Done()
	}()

	router := httprouter.New()
	router.GET("/app/:appkey", serveWs)
	router.POST("/apps/:appid/events", createEvent)

	server := &http.Server{Addr: options.WebsocketHost + ":" + options.WebsocketPort, Handler: router}
	stopped := false

	go func() {
		log.WithField("address", server.Addr).Info("listening")
		if err := server.ListenAndServe(); err != nil {
			if stopped {
				log.Warn(err)
			} else {
				log.Fatal(err)
			}
		}
	}()

	<-stop
	shutdownContext, _ := context.WithTimeout(context.Background(), 10*time.Second)

	stopped = true
	server.Shutdown(shutdownContext)
	wsCancel()

	wsDrain := make(chan struct{}, 1)
	go func() {
		websocketWaitGroup.Wait()
		log.Debug("websockets drained")
		sweeper.Wait()
		log.Debug("sweeper finished")
		wsDrain <- struct{}{}
	}()

	log.Info("draining server")
	select {
	case <-shutdownContext.Done():
		log.Fatal("drain timeout")
	case <-stop:
		log.Fatal("drain aborted")
	case <-wsDrain:
		log.Info("drain complete")
	}
}
