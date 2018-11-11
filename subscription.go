package main

// A Subscription abstracts the connection to a channel
type Subscription struct {
	//	pubsub *redis.PubSub
	messages chan []byte
	channel  *Channel
}

// Messages returns a channel of raw (string) messages
func (s Subscription) Messages() chan []byte {
	return s.messages
}

// Unsubscribe from the channel
func (s *Subscription) Unsubscribe() {
	s.channel.Unsubscribe(s)
}

// Close this subscription
func (s Subscription) Close() {
	s.Unsubscribe()
	close(s.messages)
}

// Receive a message
func (s Subscription) Receive(message []byte) {
	s.messages <- message
}

// NewSubscription creates a subscription for the channel
func NewSubscription(channelName string) (*Subscription, error) {
	subscription := &Subscription{
		messages: make(chan []byte),
	}

	channel, err := GetChannel(channelName)
	if err != nil {
		return nil, err
	}

	subscription.channel = channel
	channel.Subscribe(subscription)

	return subscription, nil
}
