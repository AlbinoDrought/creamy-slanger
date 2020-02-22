# Creamy Slanger

<a href="https://hub.docker.com/r/albinodrought/creamy-slanger">
  <img alt="albinodrought/creamy-slanger Docker Pulls" src="https://img.shields.io/docker/pulls/albinodrought/creamy-slanger">
</a>
<a href="https://github.com/AlbinoDrought/creamy-slanger/blob/master/LICENSE">
  <img alt="AGPL-3.0 License" src="https://img.shields.io/github/license/AlbinoDrought/creamy-slanger">
</a>

I wanted to play with golang and thought remaking [Slanger](https://github.com/stevegraham/slanger) would be a fun start.

After that I discovered [laravel-websockets](https://github.com/beyondcode/laravel-websockets) which had support for more of the API.

This is my adaptation of the above two projects. It is not currently production ready, but I'm working on it :tm:

## Running

`creamy-slanger` does the following by default:

- listen for both websocket _and_ API calls on `0.0.0.0:8080`
- use the redis server at `0.0.0.0:6379`, no password, database 0
- use the app `{ "id": "42", "key": "foo", "secret": "bar" }`
- hide debug output

Options can be changed with ENV vars or a config file.

### Configuring with ENV

Common:

```sh
CREAMY_SLANGER_APP_ID=42 \
CREAMY_SLANGER_APP_KEY=foo \
CREAMY_SLANGER_APP_SECRET=bar \
CREAMY_SLANGER_REDIS_ADDRESS=0.0.0.0:6379 \
./creamy-slanger
```

Exhaustive:

```sh
CREAMY_SLANGER_DEBUG=true \
CREAMY_SLANGER_APP_ID=42 \
CREAMY_SLANGER_APP_KEY=foo \
CREAMY_SLANGER_APP_SECRET=bar \
CREAMY_SLANGER_APP_CAPACITY_ENABLED=false \
CREAMY_SLANGER_APP_CAPACITY_MAX=0 \
CREAMY_SLANGER_APP_CLIENTMESSAGES_ENABLED=false \
CREAMY_SLANGER_APP_ACTIVITYTIMEOUT=120 \
CREAMY_SLANGER_WEBSOCKET_HOST=0.0.0.0 \
CREAMY_SLANGER_WEBSOCKET_PORT=8080 \
CREAMY_SLANGER_WEBSOCKET_READ_BUFFER_SIZE=1024 \
CREAMY_SLANGER_WEBSOCKET_WRITE_BUFFER_SIZE=1024 \
CREAMY_SLANGER_REDIS_ADDRESS=0.0.0.0:6379 \
CREAMY_SLANGER_REDIS_PASSWORD=foo \
CREAMY_SLANGER_REDIS_DATABASE=0 \
./creamy-slanger
```

### Configuring with file

See `config.sample.json`:

```json
{
  "debug": true,
  "app": {
    "id": "42",
    "key": "foo",
    "secret": "bar",
    "capacity": {
      "enabled": false,
      "max": 0
    },
    "clientmessages": {
      "enabled": false
    },
    "timeout": 120
  },
  "websocket": {
    "host": "0.0.0.0",
    "port": "8080",
    "read_buffer_size": 1024,
    "write_buffer_size": 1024
  },
  "redis": {
    "address": "0.0.0.0:6379",
    "password": "",
    "database": 0
  }
}
```

## Building

### Without Docker

```sh
go get
go build
```

### With Docker

```sh
docker build -t albinodrought/creamy-slanger .
```