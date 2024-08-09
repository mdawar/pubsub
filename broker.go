package pubsub

// Message represents a message delivered by the broker to a subscriber.
type Message[T any, P any] struct {
	// Topic is the topic on which the message is published.
	Topic T
	// Payload holds the published value.
	Payload P
}

// Broker represents a message broker.
//
// The Broker is the core component of the pub/sub library.
// It manages the registration of subscribers and handles the publishing
// of messages to specific topics or broadcasting to all subscribers.
//
// The Broker supports concurrent operations.
type Broker[T comparable, P any] struct {
	// topics holds the topics and their subscriptions as a slice.
	topics map[T][]chan Message[T, P]
	// subs holds all of the subscription channels.
	subs []chan Message[T, P]
}

// NewBroker creates a new message broker instance.
func NewBroker[T comparable, P any]() *Broker[T, P] {
	return &Broker[T, P]{
		topics: make(map[T][]chan Message[T, P]),
	}
}

// Topics returns a slice of all the topics registered on the Broker.
//
// A nil slice is returned if there are no topics.
//
// Note: The order of the topics is not guaranteed.
func (b *Broker[T, P]) Topics() []T {
	var topics []T
	// The iteration order over maps is not guaranteed.
	for topic := range b.topics {
		topics = append(topics, topic)
	}

	return topics
}

// NumSubs returns the number of registered subscriptions.
func (b *Broker[T, P]) NumSubs() int {
	return len(b.subs)
}

// NumTopics returns the total number of topics registered on the broker.
func (b *Broker[T, P]) NumTopics() int {
	return len(b.topics)
}

// Subscribe creates a subscription for the specified topics.
//
// The created subscription channel is unbuffered (capacity = 0).
func (b *Broker[T, P]) Subscribe(topics ...T) <-chan Message[T, P] {
	return b.SubscribeWithCapacity(0, topics...)
}

// Subscribe creates a subscription for the specified topics with the specified capacity.
//
// The capacity specifies the subscription channel's buffer capacity.
func (b *Broker[T, P]) SubscribeWithCapacity(capacity int, topics ...T) <-chan Message[T, P] {
	sub := make(chan Message[T, P], capacity)

	for _, topic := range topics {
		b.topics[topic] = append(b.topics[topic], sub)
	}

	b.subs = append(b.subs, sub)

	return sub
}

// Unsubscribe removes a subscription for the specified topics.
//
// All topic subscriptions are removed if none are specified.
func (b *Broker[T, P]) Unsubscribe(sub <-chan Message[T, P], topics ...T) {
	for _, topic := range topics {
		// Topic subscribers.
		subscribers := b.topics[topic]
		for i, s := range subscribers {
			if s == sub {
				// Remove the topic if this is the only subscription.
				if len(subscribers) == 1 {
					delete(b.topics, topic)
				} else {
					// Remove the subscription channel.
					b.topics[topic] = append(subscribers[:i], subscribers[i+1:]...)
				}
			}
		}
	}

	// TODO: only remove if subscriptions on all topics are removed.
	for i, s := range b.subs {
		if s == sub {
			b.subs = append(b.subs[:i], b.subs[i+1:]...)
		}
	}
}

// Publish publishes a message on the topic with the specified payload.
//
// The payload will be discarded if the there are no subscribers on the topic.
//
// This method will block if any of the subscription channels buffer is full.
// This can be used to guarantee message delivery.
func (b *Broker[T, P]) Publish(topic T, payload P) {
	subs := b.topics[topic]

	for _, sub := range subs {
		sub <- Message[T, P]{Topic: topic, Payload: payload}
	}
}

// TryPublish publishes a message on the topic with the specified payload if the subscription's
// channel buffer is not full.
func (b *Broker[T, P]) TryPublish(topic T, payload P) {}

// Broadcast publishes a message with the specified payload on all the subscriptions.
func (b *Broker[T, P]) Broadcast(payload P) {}

// TryBroadcast publishes a message with the specified payload on all the subscriptions if the
// subscription's buffer is not full.
func (b *Broker[T, P]) TryBroadcast(payload P) {}
