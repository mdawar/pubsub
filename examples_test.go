package pubsub_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mdawar/pubsub"
)

func Example() {
	// A broker with topic and message payload of type string.
	broker := pubsub.NewBroker[string, string]()

	// Topic to publish the message to.
	topic := "example"
	// Number of subscribers to the topic.
	subCount := 100

	var wg sync.WaitGroup

	// Run consumer goroutines subscribed to the same topic.
	for range subCount {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Subscribe to topic.
			sub := broker.Subscribe(topic)
			// Wait for message.
			msg := <-sub
			// Message fields.
			_ = msg.Topic
			_ = msg.Payload
		}()
	}

	// Helper function to wait for all consumer goroutines to subscribe.
	// This is just for the example and not needed in production code.
	waitUntil(time.Second, func() bool {
		return broker.Subscribers(topic) == subCount
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Publish a message concurrently to the topic.
	// This call will wait for all subscribers to receive the message or
	// until the context is canceled (e.g. on timeout).
	err := broker.Publish(ctx, topic, "Message to deliver")
	if err != nil {
		// In this case 1 or more subscribers did not receive the message.
		switch {
		case errors.Is(err, context.Canceled):
			fmt.Println("Publishing was canceled")
		case errors.Is(err, context.DeadlineExceeded):
			fmt.Println("Publishing timed out")
		}
	}

	wg.Wait()

	fmt.Println(err)
	// Output: <nil>
}

func ExampleBroker() {
	// A broker with topic and message payload of type string.
	// The payload can of any type.
	pubsub.NewBroker[string, string]()

	// A broker with integer topics.
	pubsub.NewBroker[int, string]()
}

func ExampleBroker_Subscribe() {
	broker := pubsub.NewBroker[string, string]()

	// Create a subscription to a single topic.
	sub1 := broker.Subscribe("events")
	_ = sub1

	// Create a subscription to multiple topics.
	sub2 := broker.Subscribe("events", "actions", "errors")
	_ = sub2
}

func ExampleBroker_SubscribeWithCapacity() {
	broker := pubsub.NewBroker[string, string]()

	// Create a subscription to a single topic with a specific channel capacity.
	sub1 := broker.SubscribeWithCapacity(10, "events")
	_ = sub1

	// Create a subscription to multiple topics with a specific channel capacity.
	sub2 := broker.SubscribeWithCapacity(10, "events", "actions", "errors")
	_ = sub2
}

func ExampleBroker_Unsubscribe() {
	broker := pubsub.NewBroker[string, string]()

	sub := broker.Subscribe("events", "actions", "errors")

	// Unsubscribe from a single topic.
	broker.Unsubscribe(sub, "events")

	// Unsubscribe from all topics.
	// The channel will not be closed, it will only stop receiving messages.
	broker.Unsubscribe(sub)
}

func ExampleBroker_Publish() {
	broker := pubsub.NewBroker[string, string]()

	// Publish a message to the topic concurrently.
	//
	// This call will wait for all the subscribers to receive the message
	// or the context to be canceled.
	//
	// If there are no subscribers the message will be discarded.
	err := broker.Publish(context.TODO(), "events", "Message payload to deliver")

	// A nil error is expected if the context is not canceled.
	fmt.Println(err == nil)
	// Output: true
}

func ExampleBroker_Publish_timeout() {
	broker := pubsub.NewBroker[string, string]()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Publish a message concurrently with a timeout of 1 second.
	//
	// Subscribers that are ready will receive the message, the others will be given
	// up to the timeout value to receive the message.
	//
	// A slow consumer will not affect the other subscribers; the timeout applies
	// individually to each subscriber.
	err := broker.Publish(ctx, "events", "Message payload to deliver")
	if err != nil {
		// In this case 1 or more subscribers did not receive the message.
		switch {
		case errors.Is(err, context.Canceled):
			fmt.Println("Publishing was canceled")
		case errors.Is(err, context.DeadlineExceeded):
			fmt.Println("Publishing timed out")
		}
	}
}

func ExampleBroker_Subscribers() {
	broker := pubsub.NewBroker[string, string]()
	topic := "example"

	for range 10 {
		broker.Subscribe(topic)
	}

	// The number of subscribers to this topic.
	subscribers := broker.Subscribers(topic)
	fmt.Println(subscribers)
	// Output: 10
}

func ExampleBroker_TryPublish() {
	broker := pubsub.NewBroker[string, string]()
	topic := "example"

	// A subscription that will not receive the message.
	// The channel is unbuffered and will not be ready when the message is published.
	broker.Subscribe(topic)
	// A buffered subscription that will receive the message.
	bufferedSub := broker.SubscribeWithCapacity(1, topic)

	// This method will send the message to the subscribers that are ready
	// to receive it (channel buffer not full) and the others will be skipped.
	broker.TryPublish(topic, "abc")

	// Receive the message on the buffered subscription.
	msg := <-bufferedSub

	fmt.Println(msg.Payload)
	// Output: abc
}
