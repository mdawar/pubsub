# pubsub

[![Go Reference](https://pkg.go.dev/badge/github.com/mdawar/pubsub.svg)](https://pkg.go.dev/github.com/mdawar/pubsub)
[![Go Report Card](https://goreportcard.com/badge/github.com/mdawar/pubsub)](https://goreportcard.com/report/github.com/mdawar/pubsub)
[![Tests](https://github.com/mdawar/pubsub/actions/workflows/test.yml/badge.svg)](https://github.com/mdawar/pubsub/actions/workflows/test.yml)

`pubsub` is a simple and generic **topic-based** **publish/subscribe** package for in-process communication in **Go**.

## Installation

```sh
go get -u github.com/mdawar/pubsub
```

#### Import

```go
import "github.com/mdawar/pubsub"
```

## Usage

#### Create a Broker

```go
// Create a broker and specify the topics type and the message payloads type.
broker := pubsub.NewBroker[string, string]()

// The message payload type can be any type.
events := pubsub.NewBroker[string, Event]()
```

#### Subscriptions

A subscription is simply a channel, everything that you know about [channels](https://go.dev/ref/spec#Channel_types) can be applied to subscriptions.

```go
// Create a subscription to a topic.
// A subscription is a channel that receives messages published to the topic.
// By default the channel is unbuffered (capacity = 0).
sub1 := broker.Subscribe("events")

// A subscription can be created to multiple topics.
// The channel will receive all the messages published to all the specified topics.
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
// Publish a message with the specified payload.
// The payload can be of any type that is specified when creating the broker.
//
// The message will be sent concurrently to the subscribers, ensuring that a slow
// consumer won't affect the other subscribers.
//
// This call will block until all the subscription channels receive the message
// or until the context is canceled.
//
// A nil return value indicates that all the subscribers received the message.
broker.Publish(context.TODO(), "events", "Sample message")
```

```go
// Publish a message and timeout after 1 second.
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
// In this case, Publish will deliver the message to subscribers that are
// ready and will wait for the others for up to the timeout duration.
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
// Non blocking publish (Fire and forget).
//
// The message is sent sequentially to the subscribers that are ready to receive it
// and the others are skipped.
//
// Note: Message delivery is not guaranteed.
broker.TryPublish("events", "A message that may not be delivered")

// Buffered subscriptions can be used for guaranteed delivery with a non-blocking publish.
//
// Publish will still block if any subscription's channel buffer is full, or any of the
// subscriptions is an unbuffered channel.
sub := broker.SubscribeWithCapacity(1, "events")
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
count := broker.Subscribers("events")
```

## Tests

```sh
go test -race -cover
# If you have "just" installed.
just test
# Or using make.
make test
```

## Benchmarks

```sh
go test -bench .
# Or Using "just".
just benchmark
# Or using make.
make benchmark
```
