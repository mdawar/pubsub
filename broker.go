// Package pubsub provides a generic and concurrency-safe, topic-based publish/subscribe library for in-process communication.
package pubsub

import (
	"context"
	"sync"
)

// Message represents a message delivered by the broker to a subscriber.
type Message[T any, B any] struct {
	// Topic is the topic on which the message is published.
	Topic T
	// Body holds the message body.
	Body B
}

// Broker represents a message broker.
//
// The Broker is the core component of the pub/sub library.
// It manages the registration of subscribers and handles the publishing
// of messages to specific topics.
//
// The Broker supports concurrent operations.
type Broker[T comparable, B any] struct {
	// Mutex to protect the subs map.
	mu sync.RWMutex
	// subs holds the topics and their subscriptions as a slice.
	subs map[T][]chan Message[T, B]
}

// NewBroker creates a new message [Broker] instance.
func NewBroker[T comparable, B any]() *Broker[T, B] {
	return &Broker[T, B]{
		subs: make(map[T][]chan Message[T, B]),
	}
}

// Topics returns a slice of all the topics registered on the [Broker].
//
// A nil slice is returned if there are no topics.
//
// NOTE: The order of the topics is not guaranteed.
func (b *Broker[T, B]) Topics() []T {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var topics []T
	// The iteration order over maps is not guaranteed.
	for topic := range b.subs {
		topics = append(topics, topic)
	}

	return topics
}

// NumTopics returns the total number of topics registered on the [Broker].
func (b *Broker[T, B]) NumTopics() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

// Subscribers returns the number of subscriptions on the specified topic.
func (b *Broker[T, B]) Subscribers(topic T) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[topic])
}

// Subscribe creates a subscription for the specified topics.
//
// The created subscription channel is unbuffered (capacity = 0).
func (b *Broker[T, B]) Subscribe(topics ...T) <-chan Message[T, B] {
	return b.SubscribeWithCapacity(0, topics...)
}

// Subscribe creates a subscription for the specified topics with the specified capacity.
//
// The capacity specifies the subscription channel's buffer capacity.
func (b *Broker[T, B]) SubscribeWithCapacity(capacity int, topics ...T) <-chan Message[T, B] {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := make(chan Message[T, B], capacity)

	for _, topic := range topics {
		b.subs[topic] = append(b.subs[topic], sub)
	}

	return sub
}

// Unsubscribe removes a subscription for the specified topics.
//
// All topic subscriptions are removed if none are specified.
//
// The channel will not be closed, it will only stop receiving messages.
//
// NOTE: Specifying the topics to unsubscribe from can be more efficient.
func (b *Broker[T, B]) Unsubscribe(sub <-chan Message[T, B], topics ...T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(topics) > 0 {
		// Unsubscribe from the specified topics.
		for _, topic := range topics {
			b.removeSubscription(sub, topic)
		}
		return
	}

	// Unsubscribe from all topics.
	for topic := range b.subs {
		b.removeSubscription(sub, topic)
	}
}

// removeSubscription removes a subscription channel from a topic.
//
// The topic will be removed if there are no other subscriptions.
func (b *Broker[T, B]) removeSubscription(sub <-chan Message[T, B], topic T) {
	subscribers := b.subs[topic]
	for i, s := range subscribers {
		if s == sub {
			// Remove the topic if this is the only subscription.
			if len(subscribers) == 1 {
				delete(b.subs, topic)
			} else {
				// Remove the subscription channel form the slice.
				b.subs[topic] = append(subscribers[:i], subscribers[i+1:]...)
			}
		}
	}
}

// Publish publishes a [Message] to the topic with the specified body.
//
// The message is sent concurrently to the subscribers, ensuring that a slow
// consumer won't affect the other subscribers.
//
// This method will block and wait for all the subscriptions to receive
// the message or until the context is canceled.
//
// The value of [context.Context.Err] will be returned.
//
// A nil return value indicates that all the subscribers received the message.
//
// If there are no subscribers to the topic, the message will be discarded.
func (b *Broker[T, B]) Publish(ctx context.Context, msg Message[T, B]) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs := b.subs[msg.Topic]

	switch len(subs) {
	case 0:
		// Do nothing.
	case 1:
		select {
		case <-ctx.Done():
		case subs[0] <- msg:
		}
	default:
		var wg sync.WaitGroup

		for _, sub := range subs {
			select {
			case <-ctx.Done():
			// Try to send if ready.
			case sub <- msg:
			// Send in a goroutine if not ready.
			default:
				wg.Add(1)
				go func() {
					defer wg.Done()

					select {
					case <-ctx.Done():
						return
					case sub <- msg:
					}
				}()
			}
		}

		wg.Wait()
	}

	return ctx.Err()
}

// TryPublish publishes a message to the topic with the specified body if the subscription's
// channel buffer is not full.
//
// The message is sent sequentially to the subscribers that are ready to receive it and the others
// are skipped.
//
// NOTE: Use the [Broker.Publish] method for guaranteed delivery.
func (b *Broker[T, B]) TryPublish(msg Message[T, B]) {
	b.mu.RLock()
	for _, sub := range b.subs[msg.Topic] {
		select {
		case sub <- msg:
		default:
		}
	}
	b.mu.RUnlock()
}
