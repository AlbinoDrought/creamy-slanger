package websockets

func valueOrEmpty(values map[string]interface{}, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

type MessagePayload map[string]interface{}

type ClientMessagePayload MessagePayload

func (cmp ClientMessagePayload) valueOrEmpty(key string) string {
	return valueOrEmpty(cmp, key)
}

func (cmp ClientMessagePayload) Data() map[string]interface{} {
	data, ok := cmp["data"]
	if !ok {
		return map[string]interface{}{}
	}

	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	return dataMap
}

func (cmp ClientMessagePayload) Event() string {
	return cmp.valueOrEmpty("event")
}

func (cmp ClientMessagePayload) Channel() string {
	return valueOrEmpty(cmp.Data(), "channel")
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

type Connection interface {
	App() App
	SetApp(app App)

	AppKey() string

	SocketID() string
	SetSocketID(socketID string)

	Send(message MessagePayload)
}
