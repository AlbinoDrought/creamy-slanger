# Creamy Slanger

I wanted to play with golang and thought remaking [Slanger](https://github.com/stevegraham/slanger) would be a fun start.

This is not currently production ready, but I'm working on it :tm:

## Building

```sh
go get
go build
```

## Running

`creamy-slanger` does the following by default:

- listen for both websocket _and_ API calls on `0.0.0.0:8080`
- use the redis server at `0.0.0.0:6379`, no password, database 0
- set app key as `foo`
- hide debug output

Options can be changed with ENV vars or a config file.

### Configuring with ENV

Common:

```sh
CREAMY_SLANGER_APP_KEY=foo \
CREAMY_SLANGER_REDIS_ADDRESS=0.0.0.0:6379 \
./creamy-slanger
```

Exhaustive:

```sh
CREAMY_SLANGER_DEBUG=true \
CREAMY_SLANGER_APP_KEY=foo \
CREAMY_SLANGER_WEBSOCKET_HOST=0.0.0.0 \
CREAMY_SLANGER_WEBSOCKET_PORT=8080 \
CREAMY_SLANGER_WEBSOCKET_TIMEOUT=120 \
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
    "key": "foo"
  },
  "websocket": {
    "host": "0.0.0.0",
    "port": "8080",
    "timeout": 120
  },
  "redis": {
    "address": "0.0.0.0:6379",
    "password": "",
    "database": 0
  }
}
```
