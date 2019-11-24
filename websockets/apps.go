package websockets

type App interface {
	ID() string
	Key() string
	Secret() string
	CapacityEnabled() bool
	Capacity() int
	ClientMessagesEnabled() bool
}

type StaticApp struct {
	AppID                    string
	AppKey                   string
	AppSecret                string
	AppCapacityEnabled       bool
	AppCapacity              int
	AppClientMessagesEnabled bool
}

func (a *StaticApp) ID() string {
	return a.AppID
}

func (a *StaticApp) Key() string {
	return a.AppKey
}

func (a *StaticApp) Secret() string {
	return a.AppSecret
}

func (a *StaticApp) CapacityEnabled() bool {
	return a.AppCapacityEnabled
}

func (a *StaticApp) Capacity() int {
	return a.AppCapacity
}

func (a *StaticApp) ClientMessagesEnabled() bool {
	return a.AppClientMessagesEnabled
}
