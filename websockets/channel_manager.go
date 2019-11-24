package websockets

// A ChannelManager handles storage and retrieval of channels.
type ChannelManager interface {
	FindOrCreate(appID, channelName string) Channel
	Find(appID, channelName string) Channel
	GetChannels(appID string) []Channel
	GetConnectionCount(appID string) int
	RemoveFromAllChannels(con Connection)
}
