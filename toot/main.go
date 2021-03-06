package main

import (
	"fmt"

	"github.com/pusher/pusher-http-go"
)

func main() {
	client := pusher.Client{
		AppId:   "6969",
		Host:    "localhost:8080",
		Key:     "somekey",
		Secret:  "somesecret",
		Cluster: "rms",
		Secure:  false,
	}

	data := map[string]string{"message": "hello world"}
	_, err := client.Trigger("my-channel", "my-event", data)
	fmt.Printf("%+v", err)
}
