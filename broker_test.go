package pubsub_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/mdawar/pubsub"
)

func TestBrokerInitialNumTopics(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	want := 0

	if got := broker.NumTopics(); want != got {
		t.Errorf("want %d topics, got %d", want, got)
	}
}

func TestBrokerInitialTopics(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topics := broker.Topics()
	want := 0

	if got := len(topics); want != got {
		t.Errorf("want %d topics length, got %d", want, got)
	}
}

func TestBrokerSubscribeOnSameTopicReturnsNewChannel(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int]()

	topic := "testing"
	sub1 := broker.Subscribe(topic)
	sub2 := broker.Subscribe(topic)

	if sub1 == sub2 {
		t.Error("want new subscription channel, got same channel")
	}
}

func TestBrokerSubscribeWithCapacityOnSameTopicReturnsNewChannel(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int]()

	topic := "testing"
	sub1 := broker.SubscribeWithCapacity(1, topic)
	sub2 := broker.SubscribeWithCapacity(1, topic)

	if sub1 == sub2 {
		t.Error("want new subscription channel, got same channel")
	}
}

func TestBrokerSubscribeUnbufferedChannelCapacity(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int]()

	sub := broker.Subscribe("testing")
	wantCap := 0

	if got := cap(sub); wantCap != got {
		t.Errorf("want channel capacity %d, got %d", wantCap, got)
	}
}

func TestBrokerSubscribeBufferedChannelCapacity(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int]()

	wantCap := 10
	sub := broker.SubscribeWithCapacity(wantCap, "testing")

	if got := cap(sub); wantCap != got {
		t.Errorf("want channel capacity %d, got %d", wantCap, got)
	}
}

func TestBrokerTopics(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		subscribe []string // The topics to subscribe on.
		want      []string // The expected topics to be returned.
	}{
		"no topics": {
			subscribe: nil,
			want:      nil,
		},
		"single topic": {
			subscribe: []string{"a"},
			want:      []string{"a"},
		},
		"multiple topics": {
			subscribe: []string{"a", "b", "c"},
			want:      []string{"a", "b", "c"},
		},
		"single topic multiple subscriptions": {
			subscribe: []string{"a", "a", "a"},
			want:      []string{"a"},
		},
		"multiple topics multiple subscriptions": {
			subscribe: []string{"a", "b", "c", "a", "b", "c"},
			want:      []string{"a", "b", "c"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			broker := pubsub.NewBroker[string, string]()

			// Loop to create multiple subscriptions.
			for _, topic := range tc.subscribe {
				broker.Subscribe(topic)
			}

			got := broker.Topics()

			if !cmp.Equal(tc.want, got, sortStringSlices) {
				t.Errorf("topics do not match (-want, +got):\n%s", cmp.Diff(tc.want, got, sortStringSlices))
			}
		})
	}
}

func TestBrokerNumTopicsAfterSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int]()
	wantTopics := 10

	for i := range wantTopics {
		broker.Subscribe(fmt.Sprint(i))
	}

	if got := broker.NumTopics(); wantTopics != got {
		t.Errorf("want %d topics, got %d", wantTopics, got)
	}

	// Subscriptions on the same topics should not affect the count.
	for i := range wantTopics {
		broker.Subscribe(fmt.Sprint(i))
	}

	if got := broker.NumTopics(); wantTopics != got {
		t.Errorf("want %d topics after multiple subs on same topic, got %d", wantTopics, got)
	}
}

func TestBrokerNumTopicsDecreasesAfterUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int]()

	assertTopics := func(want int) {
		t.Helper()
		if got := broker.NumTopics(); want != got {
			t.Fatalf("want topics count %d, got %d", want, got)
		}
	}

	assertTopics(0)

	sub := broker.Subscribe("a", "b", "c")
	assertTopics(3)

	broker.Unsubscribe(sub, "a")
	assertTopics(2)

	broker.Unsubscribe(sub, "b")
	assertTopics(1)

	broker.Unsubscribe(sub, "c")
	assertTopics(0)
}

func TestBrokerNumTopicsWithSubscribersOnSameTopic(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	assertTopics := func(want int) {
		t.Helper()
		if got := broker.NumTopics(); want != got {
			t.Fatalf("want topics count %d, got %d", want, got)
		}
	}

	assertTopics(0)

	var subs []<-chan pubsub.Message[string, string]
	topic := "testing"
	count := 10

	// Subscribe on the same topic.
	for range count {
		sub := broker.Subscribe(topic)
		subs = append(subs, sub)
		assertTopics(1)
	}

	lastSubIndex := len(subs) - 1

	// Remove all of the subscriptons and keep 1.
	for _, sub := range subs[:lastSubIndex] {
		broker.Unsubscribe(sub, topic)
		assertTopics(1)
	}

	// Remove the last subscription.
	lastSub := subs[lastSubIndex]
	broker.Unsubscribe(lastSub, topic)
	assertTopics(0)
}

func TestBrokerSubs(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	assertSubs := func(topic string, want int) {
		t.Helper()
		if got := broker.Subs(topic); want != got {
			t.Fatalf("want %d subscriptions on topic %q, got %d", want, topic, got)
		}
	}

	t1 := "a"
	t2 := "b"

	assertSubs(t1, 0)
	assertSubs(t2, 0)

	sub1 := broker.Subscribe(t1, t2)
	assertSubs(t1, 1)
	assertSubs(t2, 1)

	sub2 := broker.Subscribe(t1, t2)
	assertSubs(t1, 2)
	assertSubs(t2, 2)

	broker.Unsubscribe(sub1, t1)
	assertSubs(t1, 1)

	broker.Unsubscribe(sub2, t1)
	assertSubs(t1, 0)

	broker.Unsubscribe(sub1, t2)
	assertSubs(t2, 1)

	broker.Unsubscribe(sub2, t2)
	assertSubs(t2, 0)
}

func TestBrokerPublish(t *testing.T) {
	t.Parallel()

	cases := []int{1, 2, 10, 100}

	for _, count := range cases {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			broker := pubsub.NewBroker[string, string]()
			topic := "testing"
			payload := "Test Message"

			// Subscriptions.
			var subs []<-chan pubsub.Message[string, string]

			// Create the subscriptions.
			for range count {
				sub := broker.Subscribe(topic)
				subs = append(subs, sub)
			}

			result := make(chan error)
			go func() {
				// Blocks until all subscribers receive the message.
				result <- broker.Publish(context.Background(), topic, payload)
			}()

			// Wait for messages to be received on the subscription channels.
			for _, sub := range subs {
				select {
				case got := <-sub:
					if topic != got.Topic {
						t.Errorf("want message topic %q, got %q", topic, got.Topic)
					}

					if payload != got.Payload {
						t.Errorf("want message payload %q, got %q", payload, got.Payload)
					}
				case <-time.After(time.Second):
					t.Error("timed out waiting for message")
				}
			}

			// Wait for Publish to return.
			select {
			case err := <-result:
				if err != nil {
					t.Errorf("want nil error, got %q", err)
				}
			case <-time.After(time.Second):
				t.Error("timed out waiting for Publish to return")
			}
		})
	}
}

func TestBrokerPublishWithoutSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	result := make(chan error)
	go func() {
		// A publish without any subscriptions should not block.
		result <- broker.Publish(context.Background(), "testing", "Message")
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("want nil error, got %q", err)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}
}

func TestBrokerPublishWithCanceledContext(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topic := "testing"

	// A subscription that we don't receive on.
	broker.Subscribe(topic)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := make(chan error)
	go func() {
		// Publish with a canceled context.
		result <- broker.Publish(ctx, topic, "")
	}()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Errorf(`want error %q, got "%v"`, context.Canceled, err)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}
}

func TestBrokerPublishWithCanceledContextAndWithoutSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := make(chan error)
	go func() {
		// A publish without any subscriptions should not block.
		result <- broker.Publish(ctx, "testing", "Message")
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("want nil error, got %q", err)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}
}

func TestBrokerPublishWithBufferedSubscription(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topic := "testing"
	payload := "Test Message"
	// Subscription with buffer size of 1.
	sub := broker.SubscribeWithCapacity(1, topic)

	result := make(chan error)
	go func() {
		// Publish with a buffered subscription should not block.
		result <- broker.Publish(context.Background(), topic, payload)
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("want nil error, got %q", err)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}

	got := <-sub

	if topic != got.Topic {
		t.Errorf("want message topic %q, got %q", topic, got.Topic)
	}

	if payload != got.Payload {
		t.Errorf("want message payload %q, got %q", payload, got.Payload)
	}
}

func TestBrokerPublishAfterUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topic := "testing"
	payload := "Test Message"

	sub := broker.Subscribe(topic)
	// Unsubscribe from the specified topic.
	broker.Unsubscribe(sub, topic)

	result := make(chan error)
	go func() {
		// Should not block after unsubscribe.
		result <- broker.Publish(context.Background(), topic, payload)
	}()

	// Wait for Publish to return.
	select {
	case err := <-result:
		if err != nil {
			t.Errorf("want nil error, got %q", err)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}

	// Make sure the message is not delivered after unsubscribe.
	select {
	case <-sub:
		t.Error("received unexpected message after unsubscribe")
	default:
	}
}

func TestBrokerPublishAfterUnsubscribeAllTopics(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topic := "testing"
	payload := "Test Message"

	sub := broker.Subscribe(topic)
	// Unsubscribe from all topics.
	broker.Unsubscribe(sub)

	result := make(chan error)
	go func() {
		// Should not block after unsubscribe.
		result <- broker.Publish(context.Background(), topic, payload)
	}()

	// Wait for Publish to return.
	select {
	case err := <-result:
		if err != nil {
			t.Errorf("want nil error, got %q", err)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}
}

func TestBrokerTryPublishWithoutSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.TryPublish("testing", "Message")
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("timed out waiting for TryPublish to return")
	}
}

func TestBrokerTryPublishWithUnbufferedSubscription(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	topic := "testing"
	payload := "Message"
	// Unbuffered subscription.
	sub := broker.Subscribe(topic)

	result := make(chan pubsub.Message[string, string])
	ready := make(chan struct{})
	go func() {
		close(ready)
		result <- <-sub
	}()

	// Wait until the subscription is ready to receive.
	<-ready

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.TryPublish(topic, payload)
	}()

	// Wait for TryPublish to return.
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TryPublish to return")
	}

	// Check the received message.
	select {
	case got := <-result:
		if topic != got.Topic {
			t.Errorf("want message topic %q, got %q", topic, got.Topic)
		}

		if payload != got.Payload {
			t.Errorf("want message payload %q, got %q", payload, got.Payload)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for message")
	}
}

func TestBrokerTryPublishWithUnbufferedSubscriptionNotReady(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	topic := "testing"
	// Unbuffered subscription that we don't receive on.
	sub := broker.Subscribe(topic)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Should not block if the subscription channel is not ready to receive.
		broker.TryPublish(topic, "Message")
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for TryPublish to return")
	}

	// The message should not be delivered.
	select {
	case <-sub:
		t.Error("received unexpected message")
	default:
	}
}

func TestBrokerTryPublishWithBufferedSubscription(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topic := "testing"
	payload := "Test Message"
	// Subscription with buffer size of 1.
	sub := broker.SubscribeWithCapacity(1, topic)

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.TryPublish(topic, payload)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}

	got := <-sub

	if topic != got.Topic {
		t.Errorf("want message topic %q, got %q", topic, got.Topic)
	}

	if payload != got.Payload {
		t.Errorf("want message payload %q, got %q", payload, got.Payload)
	}
}

func TestBrokerConcurrentPublishSubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	// Topics to subscribe on.
	topics := []string{"a", "b", "c"}
	// Number of subscriptions to create per topic.
	topicSubsCount := 10
	// Total number of expected subscriptions.
	totalSubsCount := topicSubsCount * len(topics)

	var wg sync.WaitGroup

	// Subscriber goroutines.
	for range topicSubsCount {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Create a new subscription for each topic.
			for _, topic := range topics {
				wg.Add(1)
				// Subscribe and wait for message in a new goroutine.
				go func() {
					defer wg.Done()
					<-broker.Subscribe(topic)
				}()
			}
		}()
	}

	// Wait for all of the subscriptions to be ready.
	// This is required to make sure all of the subscriptions receive the messages.
	waitUntil(time.Second, func() bool {
		var total int
		for _, topic := range topics {
			total += broker.Subs(topic)
		}

		return total == totalSubsCount
	})

	// Publish return values.
	results := make(chan error, len(topics))

	// Publisher goroutines.
	for _, topic := range topics {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Blocks until all the subscribers receive the message.
			results <- broker.Publish(context.Background(), topic, "")
		}()
	}

	wg.Wait()

	for range len(topics) {
		select {
		case err := <-results:
			if err != nil {
				t.Errorf("want nil error, got %q", err)
			}
		case <-time.After(time.Second):
			t.Error("timed out waiting to receive Publish return value")
		}
	}
}

func TestBrokerConcurrentTryPublishSubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	// Topics to subscribe on.
	topics := []string{"a", "b", "c"}

	var wg sync.WaitGroup

	// Subscriber goroutines.
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for _, topic := range topics {
				wg.Add(1)
				// Subscribe and wait for message in a new goroutine.
				go func() {
					defer wg.Done()
					// Try to subscribe and receive without waiting.
					select {
					case <-broker.Subscribe(topic):
					default:
					}
				}()
			}
		}()
	}

	// Publisher goroutines.
	for _, topic := range topics {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Does not wait for the subscribers to be ready.
			broker.TryPublish(topic, "")
		}()
	}

	wg.Wait()
}

func TestBrokerConcurrentSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topic := "testing"
	totalSubs := 10

	var wg sync.WaitGroup

	subs := make(chan (<-chan pubsub.Message[string, string]))

	// Subscribe goroutines.
	for range totalSubs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			subs <- broker.Subscribe(topic)
		}()
	}

	// Unsubscribe goroutines.
	for range totalSubs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			sub := <-subs
			broker.Unsubscribe(sub)
		}()
	}

	wg.Wait()
}
