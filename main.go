package main

import (
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
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
	viper.SetDefault("app.key", "foo")
	viper.SetDefault("websocket.host", "0.0.0.0")
	viper.SetDefault("websocket.port", "8080")
	viper.SetDefault("websocket.timeout", 120)
	viper.SetDefault("debug", false)
	viper.SetDefault("redis.address", "0.0.0.0:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.database", 0)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("CREAMY_SLANGER")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err == nil {
		log.Info("[server] loaded config from file")
	} else {
		log.Debug("[server] did not load config from file")
	}

	options = Options{
		AppKey:        viper.GetString("app.key"),
		WebsocketHost: viper.GetString("websocket.host"),
		WebsocketPort: viper.GetString("websocket.port"),
		Debug:         viper.GetBool("debug"),
		RedisOptions: &redis.Options{
			Addr:     viper.GetString("redis.address"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.database"),
		},
		ActivityTimeout: viper.GetInt("websocket.timeout"),
	}

	log.SetOutput(os.Stdout)
	if options.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	daddy = redis.NewClient(options.RedisOptions)

	// test redis connection
	pubsub := daddy.Subscribe("creamy-slanger")
	_, err = pubsub.Receive()
	if err != nil {
		log.Panicf("error testing redis connection: %+v", err)
	}
	pubsub.Close()
	log.Debug("[server] redis ok")

	bootServer()
}
