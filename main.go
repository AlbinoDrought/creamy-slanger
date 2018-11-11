package main

import (
	"github.com/go-redis/redis"
)

type Options struct {
	AppKey          string
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
		WebsocketHost: "0.0.0.0",
		WebsocketPort: "8080",
		Debug:         true,
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
