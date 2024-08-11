# pubsub

[![Go Reference](https://pkg.go.dev/badge/github.com/mdawar/pubsub.svg)](https://pkg.go.dev/github.com/mdawar/pubsub)
[![Tests](https://github.com/mdawar/pubsub/actions/workflows/test.yml/badge.svg)](https://github.com/mdawar/pubsub/actions/workflows/test.yml)

`pubsub` is a simple and generic **topic-based** **publish/subscribe** library for in-process communication in **Go**.

## Installation

```sh
go get -u github.com/mdawar/pubsub
```

## Usage

#### Create a Broker

```go
import "github.com/mdawar/pubsub"

// Create a broker and specify the topics type and the message payloads type.
broker := pubsub.NewBroker[string, string]()

// The message payload type can be any type.
events := pubsub.NewBroker[string, Event]()
```

#### Subscriptions

A subscription is simply a channel, everything that you know about [channels](https://go.dev/ref/spec#Channel_types) can be applied to subscriptions.

```go
// Create a subscription on a topic.
// A subscription is a channel that receives messages published on the topic.
// By default the channel is unbuffered (capacity = 0).
sub1 := broker.Subscribe("events")

// A subscription can be created on multiple topics.
// The channel will receive all the messages published on all the specified topics.
sub2 := broker.Subscribe("events", "actions", "testing")

// Create a subscription with a buffered channel.
// The channel capacity is specified as the first parameter.
sub3 := broker.SubscribeWithCapacity(10, "events")

// Unsubscribe from a specific topic or topics.
// The channel will still receive messages for the other topics.
broker.Unsubscribe(sub2, "actions", "testing")

// Unsubscribe from all topics.
// The channel will not be closed, it will only stop receiving messages.
// Note: Specifying all the topics is much more efficient if performance is critical.
broker.Unsubscribe(sub2)
```

#### Publishing Messages

```go
// Publish a message with the specified message payload.
// The payload can be of any type that is specified when creating the broker.
// This call will block until all the subscription channels receive the message.
// The context can be used to cancel the operation or set a timeout.
broker.Publish(context.TODO(), "events", "Sample message")
```

```go
// Publish a message and timeout after 1 second.
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
err := broker.Publish(ctx, "events", "Sample message")
// The error is not nil if the context was canceled or the deadline exceeded.
if err != nil {
  if errors.Is(err, context.DeadlineExceeded) {
    // Timed out.
  } else if errors.Is(err, context.Canceled) {
    // Canceled.
  }
}
```

```go
// Non blocking publish.
// The message will be received by subscriptions that are ready (channel buffer is not full).
// Note: Message delivery is not guaranteed.
broker.TryPublish("events", "A message that may not be received")

// Buffered subscriptions can be used for guaranteed delivery with a non-blocking publish.
// Publish will still block if any subscription's channel buffer is full, or any of the
// subscriptions is an unbuffered channel.
bufferedSub := broker.SubscribeWithCapacity(1, "events")
broker.Publish(context.TODO(), "events", "Guaranteed delivery message")
```

#### Messages

```go
sub := broker.Subscribe("events")

// Receive a message from the channel.
msg := <-sub

// The topic that the message was published on.
msg.Topic
// The payload that was published using Publish() or TryPublish().
msg.Payload
```

#### Topics

```go
// Get a slice of all the topics registered on the broker.
// Note: The order of the topics is not guaranteed.
topics := broker.Topics()

// Get the total number of topics.
topicsCount := broker.NumTopics()

// Get the subscribers count on a specific topic.
count := broker.Subs("events")
```

## Tests

```sh
go test -race -cover
# If you have "just" installed.
just test
# Or using make.
make test
```
