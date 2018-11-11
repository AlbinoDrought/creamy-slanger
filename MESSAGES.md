# Messages

## Greeting

To client:
`{"event":"pusher:connection_established","data":"{\"socket_id\":\"2282.7910578\",\"activity_timeout\":120}"}`

## Ping

To server:
`{"event":"pusher:ping","data":{}}`

To client:
`{"event":"pusher:pong","data":"{}"}`

## Subscribe to channel

To server:
`{"event":"pusher:subscribe","data":{"channel":"my-channel"}}`

To client:
`{"event":"pusher_internal:subscription_succeeded","data":"{}","channel":"my-channel"}`

## Inbound data

To client:
`{"event":"my-event","data":"{\"message\":\"hello world\"}","channel":"my-channel"}`