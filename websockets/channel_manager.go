package websockets

import (
	"errors"
	"strings"
	"sync"
)

// A ChannelManager handles storage and retrieval of channels.
type ChannelManager interface {
	FindOrCreate(appID, channelName string) Channel
	Find(appID, channelName string) Channel
	GetChannels(appID string) []Channel
	GetConnectionCount(appID string) int
	RemoveFromAllChannels(con Connection)
}

type channelMap struct {
	lock     sync.RWMutex
	channels map[string]Channel
}

func (cm *channelMap) Clean() bool {
	canBeCollected := false
	emptyChannels := []string{}

	cm.lock.RLock()
	for name, channel := range cm.channels {
		if !channel.HasConnections() {
			emptyChannels = append(emptyChannels, name)
		}
	}
	cm.lock.RUnlock()

	if len(emptyChannels) > 0 {
		cm.lock.Lock()
		for _, emptyChannelName := range emptyChannels {
			delete(cm.channels, emptyChannelName)
		}
		canBeCollected = len(cm.channels) == 0
		cm.lock.Unlock()
	}

	return canBeCollected
}

type arrayChannelManager struct {
	lock            sync.RWMutex
	channelsByAppID map[string]channelMap
}

func (channelManager *arrayChannelManager) FindOrCreate(appID, channelName string) Channel {
	channelManager.lock.RLock()
	channels, ok := channelManager.channelsByAppID[appID]
	channelManager.lock.RUnlock()

	if !ok {
		channelManager.lock.Lock()

		// re-check in attempt to avoid race cond
		channels, ok = channelManager.channelsByAppID[appID]
		if !ok {
			channels = channelMap{
				lock:     sync.RWMutex{},
				channels: map[string]Channel{},
			}

			channelManager.channelsByAppID[appID] = channels
		}

		channelManager.lock.Unlock()
	}

	channels.lock.RLock()
	channel, ok := channels.channels[channelName]
	channels.lock.RUnlock()

	if !ok {
		channels.lock.Lock()

		// re-check in attempt to avoid race cond
		channel, ok = channels.channels[channelName]
		if !ok {
			// actually create channel
			if strings.HasPrefix(channelName, "private-") {
				channel = &PrivateChannel{
					&PublicChannel{name: channelName},
				}
			} else if strings.HasPrefix(channelName, "presence-") {
				channel = &PresenceChannel{
					&PublicChannel{name: channelName},
				}
			} else {
				channel = &PublicChannel{name: channelName}
			}

			channels.channels[channelName] = channel
		}

		channels.lock.Unlock()
	}

	return channel
}

func (channelManager *arrayChannelManager) Find(appID, channelName string) Channel {
	channelManager.lock.RLock()
	channels, ok := channelManager.channelsByAppID[appID]
	channelManager.lock.RUnlock()

	if !ok {
		return nil
	}

	channels.lock.RLock()
	channel, ok := channels.channels[channelName]
	channels.lock.RUnlock()

	if !ok {
		return nil
	}

	return channel
}

func (channelManager *arrayChannelManager) GetChannels(appID string) []Channel {
	panic(errors.New("not implemented"))
}

func (channelManager *arrayChannelManager) GetConnectionCount(appID string) int {
	panic(errors.New("not implemented"))
}

func (channelManager *arrayChannelManager) RemoveFromAllChannels(con Connection) {
	if con.App() == nil {
		return
	}

	id := con.App().ID()

	channelManager.lock.RLock()
	channels, ok := channelManager.channelsByAppID[id]
	channelManager.lock.RUnlock()

	if !ok {
		return
	}

	// todo: optimize
	channels.lock.RLock()
	for _, channel := range channels.channels {
		channel.Unsubscribe(con)
	}
	channels.lock.RUnlock()

	if channels.Clean() {
		channelManager.lock.Lock()
		if len(channels.channels) == 0 {
			delete(channelManager.channelsByAppID, id)
		}
		channelManager.lock.Unlock()
	}
}

// NewArrayChannelManager returns a new in-memory channel manager
func NewArrayChannelManager() ChannelManager {
	return &arrayChannelManager{
		lock:            sync.RWMutex{},
		channelsByAppID: map[string]channelMap{},
	}
}
