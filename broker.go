package pubsub

// Message represents a message delivered by the broker to a subscriber.
type Message[T any] struct {
	// Topic is the topic on which the message is published.
	Topic string
	// Payload holds the published value.
	Payload T
}

// Broker represents a message broker.
//
// The Broker is the core component of the pub/sub library.
// It manages the registration of subscribers and handles the publishing
// of messages to specific topics or broadcasting to all subscribers.
//
// The Broker supports concurrent operations.
type Broker[T any] struct {
	// topics holds the topics and their subscriptions as a slice.
	topics map[string][]chan Message[T]
	// subs holds all of the subscription channels.
	subs []chan Message[T]
}

// NewBroker creates a new message broker instance.
func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		topics: make(map[string][]chan Message[T]),
	}
}

// NumSubs returns the number of registered subscriptions.
func (b *Broker[T]) NumSubs() int {
	return len(b.subs)
}

// NumTopics returns the total number of topics registered on the broker.
func (b *Broker[T]) NumTopics() int {
	return len(b.topics)
}

// Subscribe creates a subscription for the specified topics.
//
// The created subscription channel is unbuffered (capacity = 0).
func (b *Broker[T]) Subscribe(topics ...string) <-chan Message[T] {
	return b.SubscribeWithCapacity(0, topics...)
}

// Subscribe creates a subscription for the specified topics with the specified capacity.
//
// The capacity specifies the subscription channel's buffer capacity.
func (b *Broker[T]) SubscribeWithCapacity(capacity int, topics ...string) <-chan Message[T] {
	sub := make(chan Message[T], capacity)

	for _, topic := range topics {
		b.topics[topic] = append(b.topics[topic], sub)
	}

	b.subs = append(b.subs, sub)

	return sub
}

// Unsubscribe removes a subscription for the specified topics.
//
// All topic subscriptions are removed if none are specified.
func (b *Broker[T]) Unsubscribe(sub <-chan Message[T], topics ...string) {}

// Publish publishes a message on the topic with the specified payload.
func (b *Broker[T]) Publish(topic string, payload T) {}

// TryPublish publishes a message on the topic with the specified payload if the subscription's
// channel buffer is not full.
func (b *Broker[T]) TryPublish(topic string, payload T) {}

// Broadcast publishes a message with the specified payload on all the subscriptions.
func (b *Broker[T]) Broadcast(payload T) {}

// TryBroadcast publishes a message with the specified payload on all the subscriptions if the
// subscription's buffer is not full.
func (b *Broker[T]) TryBroadcast(payload T) {}
