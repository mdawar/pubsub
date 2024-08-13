package pubsub_test

import (
	"context"
	"strconv"
	"testing"
	"time"

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

	subs := make([]<-chan pubsub.Message[string, string], 0, b.N)

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
	// Number of subscriptions to benchmark.
	cases := []int{0, 1, 2, 4, 6, 8, 10}

	for _, count := range cases {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			broker := pubsub.NewBroker[string, string]()
			topic := "testing"

			done := make(chan struct{})

			for range count {
				go func() {
					msgs := broker.Subscribe(topic)
					for {
						select {
						case <-done:
							return
						case <-msgs:
						}
					}
				}()
			}

			ready := waitUntil(time.Second, func() bool {
				return broker.Subscribers(topic) == count
			})

			if !ready {
				b.Fatal("timed out waiting for subscriptions")
			}

			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				broker.Publish(ctx, topic, strconv.Itoa(i))
			}
			b.StopTimer()

			close(done)
		})
	}
}

func BenchmarkBrokerTryPublish(b *testing.B) {
	// Number of subscriptions to benchmark.
	cases := []int{0, 1, 2, 4, 6, 8, 10}

	for _, count := range cases {
		b.Run(strconv.Itoa(count), func(b *testing.B) {
			broker := pubsub.NewBroker[string, string]()
			topic := "testing"

			done := make(chan struct{})

			for range count {
				go func() {
					msgs := broker.Subscribe(topic)
					for {
						select {
						case <-done:
							return
						case <-msgs:
						}
					}
				}()
			}

			ready := waitUntil(time.Second, func() bool {
				return broker.Subscribers(topic) == count
			})

			if !ready {
				b.Fatal("timed out waiting for subscriptions")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				broker.TryPublish(topic, strconv.Itoa(i))
			}
			b.StopTimer()

			close(done)
		})
	}
}
