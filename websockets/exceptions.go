package websockets

// A WebsocketException is an Error that can (and should) be sent over WS to clients
type WebsocketException interface {
	error
	GetPayload() MessagePayload
}

type websocketException struct {
	message string
	code    int
}

func (ex *websocketException) Error() string {
	return ex.message
}

func (ex *websocketException) GetPayload() MessagePayload {
	return map[string]interface{}{
		"event": "pusher:error",
		"data": map[string]interface{}{
			"message": ex.message,
			"code":    ex.code,
		},
	}
}

func connectionsOverCapacityException() WebsocketException {
	return &websocketException{
		message: "Over capacity", // (sic)
		code:    4100,
	}
}

func invalidConnectionException() WebsocketException {
	return &websocketException{
		message: "Invalid Connection",
		code:    4009,
	}
}

func invalidSignatureException() WebsocketException {
	return &websocketException{
		message: "Invalid Signature",
		code:    4009,
	}
}

func unknownAppKeyException(appKey string) WebsocketException {
	return &websocketException{
		message: "Could not find app key `" + appKey + "`.",
		code:    4001,
	}
}
