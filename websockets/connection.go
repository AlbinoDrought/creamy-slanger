package websockets

type MessagePayload map[string]interface{}

type ClientMessagePayload MessagePayload

func (cmp ClientMessagePayload) valueOrEmpty(key string) string {
	value, ok := cmp[key]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

func (cmp ClientMessagePayload) Event() string {
	return cmp.valueOrEmpty("event")
}

func (cmp ClientMessagePayload) Channel() string {
	return cmp.valueOrEmpty("channel")
}

func (cmp ClientMessagePayload) ChannelData() string {
	return cmp.valueOrEmpty("channel_data")
}

func (cmp ClientMessagePayload) Auth() string {
	return cmp.valueOrEmpty("auth")
}

func (cmp ClientMessagePayload) MessagePayload() MessagePayload {
	return MessagePayload(cmp)
}

type App interface {
	ID() string
	Secret() string
	CapacityEnabled() bool
	Capacity() int
	ClientMessagesEnabled() bool
}

type Connection interface {
	App() App
	SetApp(app App)

	SocketID() string
	SetSocketID(socketID string)

	Send(message MessagePayload)
}
