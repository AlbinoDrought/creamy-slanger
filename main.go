package main

import (
	"github.com/go-redis/redis"
)

type Options struct {
	AppKey          string
	APIHost         string
	APIPort         string
	WebsocketHost   string
	WebsocketPort   string
	Debug           bool
	RedisOptions    *redis.Options
	ActivityTimeout int
}

var (
	options Options
	daddy   *redis.Client
)

func main() {
	options = Options{
		AppKey:        "foo",
		APIHost:       "0.0.0.0",
		APIPort:       "4567",
		WebsocketHost: "0.0.0.0",
		WebsocketPort: "8080",
		Debug:         true,
		// RedisAddress:    "redis://0.0.0.0:6379",
		RedisOptions: &redis.Options{
			Addr:     "0.0.0.0:6379",
			Password: "",
			DB:       0,
		},
		ActivityTimeout: 30,
	}

	daddy = redis.NewClient(options.RedisOptions)

	bootServer()
}
