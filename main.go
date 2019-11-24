package main

import (
	"os"
	"strings"

	"github.com/AlbinoDrought/creamy-slanger/websockets"
	"github.com/spf13/viper"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

type SlangerOptions struct {
	AppManager     websockets.AppManager
	ChannelManager websockets.ChannelManager
	EventManager   websockets.EventManager
	Handler        websockets.Handler
}

type Options struct {
	WebsocketHost string
	WebsocketPort string
	Debug         bool
	RedisOptions  *redis.Options
}

var (
	slangerOptions SlangerOptions
	options        Options
	daddy          *redis.Client
)

func init() {
	viper.SetDefault("app.id", "6969")
	viper.SetDefault("app.key", "foo")
	viper.SetDefault("app.secret", "bar")
	viper.SetDefault("app.capacity.enabled", false)
	viper.SetDefault("app.capacity.max", 0)
	viper.SetDefault("app.clientmessages.enabled", false)
	viper.SetDefault("app.activitytimeout", 120)
	viper.SetDefault("websocket.host", "0.0.0.0")
	viper.SetDefault("websocket.port", "8080")
	viper.SetDefault("debug", false)
	viper.SetDefault("redis.address", "0.0.0.0:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.database", 0)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("CREAMY_SLANGER")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
}

func main() {
	err := viper.ReadInConfig()
	if err == nil {
		log.Info("[server] loaded config from file")
	} else {
		log.Debug("[server] did not load config from file")
	}

	options = Options{
		WebsocketHost: viper.GetString("websocket.host"),
		WebsocketPort: viper.GetString("websocket.port"),
		Debug:         viper.GetBool("debug"),
		RedisOptions: &redis.Options{
			Addr:     viper.GetString("redis.address"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.database"),
		},
	}

	daddy = redis.NewClient(options.RedisOptions)
	slangerOptions = SlangerOptions{
		AppManager: websockets.NewArrayAppManager([]websockets.App{
			&websockets.StaticApp{
				AppID:                    viper.GetString("app.id"),
				AppKey:                   viper.GetString("app.key"),
				AppSecret:                viper.GetString("app.secret"),
				AppCapacityEnabled:       viper.GetBool("app.capacity.enabled"),
				AppCapacity:              viper.GetInt("app.capacity.max"),
				AppClientMessagesEnabled: viper.GetBool("app.clientmessages.enabled"),
				AppActivityTimeout:       viper.GetInt("app.activitytimeout"),
			},
		}),
		EventManager: websockets.NewRedisEventManager(daddy),
	}
	slangerOptions.ChannelManager = websockets.NewArrayChannelManager(slangerOptions.EventManager)
	slangerOptions.Handler = websockets.NewHandler(slangerOptions.AppManager, slangerOptions.ChannelManager)

	log.SetOutput(os.Stdout)
	if options.Debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

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
