package websockets

import "sync"

// An AppManager fetches apps
type AppManager interface {
	// FindByID finds and returns an app by ID if it exists, or nil
	FindByID(id string) App

	// FindByKey finds and returns an app by key if it exists, or nil
	FindByKey(key string) App
}

type arrayAppManager struct {
	rwLock sync.RWMutex
	idMap  map[string]App
	keyMap map[string]App
}

func (appManager *arrayAppManager) FindByID(id string) App {
	appManager.rwLock.RLock()
	defer appManager.rwLock.RUnlock()

	if app, ok := appManager.idMap[id]; ok {
		return app
	}

	return nil
}

func (appManager *arrayAppManager) FindByKey(ley string) App {
	appManager.rwLock.RLock()
	defer appManager.rwLock.RUnlock()

	if app, ok := appManager.keyMap[ley]; ok {
		return app
	}

	return nil
}

// NewArrayAppManager returns a new static AppManager
func NewArrayAppManager(apps []App) AppManager {
	manager := &arrayAppManager{
		rwLock: sync.RWMutex{},
		idMap:  map[string]App{},
		keyMap: map[string]App{},
	}

	for _, app := range apps {
		manager.idMap[app.ID()] = app
		manager.keyMap[app.Key()] = app
	}

	return manager
}
