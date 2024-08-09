# pubsub

`pubsub` is a **topic-based** **publish/subscribe** library for in-process communication in **Go**.

## Installation

```sh
go get -u github.com/mdawar/pubsub
```

## Usage

#### Create a Broker

```go
import "github.com/mdawar/pubsub"

// Create a broker and specify the message payload type.
broker := pubsub.NewBroker[string]()
```

#### Create a Subscription

```go
// Create a subscription on a topic.
// A subscription is a channel that receives messages published on the topic.
sub1 := broker.Subscribe("events")

// A subscription can be created on multiple topics.
sub2 := broker.Subscribe("events", "actions", "testing")
```

#### Publish Messages

```go
// Publish a message with the specified payload.
// The payload can be any type that is specified when creating the broker.
broker.Publish("events", "Sample message")
```
