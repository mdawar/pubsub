// Package pubsub provides a generic and concurrency-safe, topic-based publish/subscribe library for in-process communication.
package pubsub

import (
	"context"
	"sync"
)

// Message represents a message delivered by the broker to a subscriber.
type Message[T any, P any, S any] struct {
	// Topic is the topic on which the message is published.
	Topic T
	// Payload holds the published value.
	Payload P
	// Sender is an identifier for the message's sender.
	Sender S
}

// Broker represents a message broker.
//
// The Broker is the core component of the pub/sub library.
// It manages the registration of subscribers and handles the publishing
// of messages to specific topics.
//
// The Broker supports concurrent operations.
type Broker[T comparable, P any, S any] struct {
	// Mutex to protect the subs map.
	mu sync.RWMutex
	// subs holds the topics and their subscriptions as a slice.
	subs map[T][]chan Message[T, P, S]
}

// NewBroker creates a new message [Broker] instance.
func NewBroker[T comparable, P any, S any]() *Broker[T, P, S] {
	return &Broker[T, P, S]{
		subs: make(map[T][]chan Message[T, P, S]),
	}
}

// Topics returns a slice of all the topics registered on the [Broker].
//
// A nil slice is returned if there are no topics.
//
// NOTE: The order of the topics is not guaranteed.
func (b *Broker[T, P, S]) Topics() []T {
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
func (b *Broker[T, P, S]) NumTopics() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

// Subscribers returns the number of subscriptions on the specified topic.
func (b *Broker[T, P, S]) Subscribers(topic T) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[topic])
}

// Subscribe creates a subscription for the specified topics.
//
// The created subscription channel is unbuffered (capacity = 0).
func (b *Broker[T, P, S]) Subscribe(topics ...T) <-chan Message[T, P, S] {
	return b.SubscribeWithCapacity(0, topics...)
}

// Subscribe creates a subscription for the specified topics with the specified capacity.
//
// The capacity specifies the subscription channel's buffer capacity.
func (b *Broker[T, P, S]) SubscribeWithCapacity(capacity int, topics ...T) <-chan Message[T, P, S] {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := make(chan Message[T, P, S], capacity)

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
func (b *Broker[T, P, S]) Unsubscribe(sub <-chan Message[T, P, S], topics ...T) {
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
func (b *Broker[T, P, S]) removeSubscription(sub <-chan Message[T, P, S], topic T) {
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

// Publish publishes a [Message] to the topic with the specified payload.
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
func (b *Broker[T, P, S]) Publish(ctx context.Context, msg Message[T, P, S]) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// TODO: add test for empty topic.
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

		wg.Add(len(subs))
		for _, sub := range subs {
			go func() {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				case sub <- msg:
				}
			}()
		}

		wg.Wait()
	}

	return ctx.Err()
}

// TryPublish publishes a message to the topic with the specified payload if the subscription's
// channel buffer is not full.
//
// The message is sent sequentially to the subscribers that are ready to receive it and the others
// are skipped.
//
// NOTE: Use the [Broker.Publish] method for guaranteed delivery.
func (b *Broker[T, P, S]) TryPublish(msg Message[T, P, S]) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subs[msg.Topic] {
		select {
		case sub <- msg:
		default:
		}
	}
}
