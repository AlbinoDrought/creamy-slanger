package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

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
			log.Println(err)
		}
		return
	}

	client := NewClient()

	go writer(ws, client)
	reader(ws, client)
}

func reader(ws *websocket.Conn, client *Client) {
	defer client.Close()
	defer ws.Close()
	ws.SetReadLimit(512)
	for {
		messageType, rawMessage, err := ws.ReadMessage()
		if err != nil {
			log.Printf("bailing on client because read error: %+v", err)
			break
		}
		if messageType == websocket.TextMessage {
			message := &Message{}
			err := json.Unmarshal(rawMessage, message)

			if err != nil {
				log.Printf("bailing on client because unmarshal error: %+v", err)
				break
			}

			client.OnMessageFromClient(message)
		}
	}
}

func writer(ws *websocket.Conn, client *Client) {
	defer client.Close()
	defer ws.Close()
	for message := range client.Messages() {
		log.Printf("forwarding message to client: %v", message)
		err := ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("client error: %+v", err)
			break
		}
	}
}

// {"name":"my-event","channels":["my-channel"],"data":"{\"message\":\"hello world\"}"}

type IncomingEvent struct {
	Name     string
	Channels []string
	Data     interface{}
}

func createEvent(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	body, _ := ioutil.ReadAll(r.Body)
	log.Printf("wew %+v: %+v", r.URL, string(body))
	defer r.Body.Close()

	event := IncomingEvent{}
	json.Unmarshal(body, &event)
	log.Printf("aeiou %+v", event)

	for _, channelName := range event.Channels {
		eventPayload, _ := json.Marshal(map[string]interface{}{
			"event":   event.Name,
			"channel": channelName,
			"data":    event.Data,
		})
		daddy.Publish(channelName, eventPayload)
	}
}

func bootServer() {
	router := httprouter.New()
	router.GET("/app/:appid", serveWs)
	router.POST("/apps/:appid/events", createEvent)

	log.Fatal(http.ListenAndServe(":8080", router))
}
