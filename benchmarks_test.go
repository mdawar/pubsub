package pubsub_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/mdawar/pubsub"
)

func BenchmarkBrokerSubscribe(b *testing.B) {
	broker := pubsub.NewBroker[string, string]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Subscribe(strconv.Itoa(i))
	}
}

func BenchmarkBrokerSubscribeWithCapacity(b *testing.B) {
	broker := pubsub.NewBroker[string, string]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.SubscribeWithCapacity(1, strconv.Itoa(i))
	}
}

func BenchmarkBrokerUnsubscribe(b *testing.B) {
	broker := pubsub.NewBroker[string, string]()

	subs := make([]<-chan pubsub.Message[string, string], b.N)

	for i := 0; i < b.N; i++ {
		sub := broker.Subscribe(strconv.Itoa(i))
		subs = append(subs, sub)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Unsubscribe(subs[i], strconv.Itoa(i))
	}
}

func BenchmarkBrokerPublish(b *testing.B) {
	broker := pubsub.NewBroker[string, string]()
	topic := "testing"

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		msgs := broker.Subscribe(topic)
		close(started)
		for {
			select {
			case <-done:
				return
			case <-msgs:
			}
		}
	}()

	<-started

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.Publish(ctx, topic, strconv.Itoa(i))
	}
	b.StopTimer()

	close(done)
}

func BenchmarkBrokerTryPublish(b *testing.B) {
	broker := pubsub.NewBroker[string, string]()
	topic := "testing"

	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		msgs := broker.Subscribe(topic)
		close(started)
		for {
			select {
			case <-done:
				return
			case <-msgs:
			}
		}
	}()

	<-started

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.TryPublish(topic, strconv.Itoa(i))
	}
	b.StopTimer()

	close(done)
}
