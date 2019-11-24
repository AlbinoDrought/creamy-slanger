package websockets

type AppManager interface {
	FindByID(id string) App
	FindByKey(key string) App
}
