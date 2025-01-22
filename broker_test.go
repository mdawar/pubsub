package pubsub_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/goleak"

	"github.com/mdawar/pubsub"
)

// Message is an alias for [pubsub.Message] with a string type for fields.
type Message = pubsub.Message[string, string, string]

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestBrokerInitialNumTopics(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
	want := 0

	if got := broker.NumTopics(); want != got {
		t.Errorf("want %d topics, got %d", want, got)
	}
}

func TestBrokerInitialTopics(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
	topics := broker.Topics()
	want := 0

	if got := len(topics); want != got {
		t.Errorf("want %d topics length, got %d", want, got)
	}
}

func TestBrokerSubscribeOnSameTopicReturnsNewChannel(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int, string]()

	topic := "testing"
	sub1 := broker.Subscribe(topic)
	sub2 := broker.Subscribe(topic)

	if sub1 == sub2 {
		t.Error("want new subscription channel, got same channel")
	}
}

func TestBrokerSubscribeWithCapacityOnSameTopicReturnsNewChannel(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int, string]()

	topic := "testing"
	sub1 := broker.SubscribeWithCapacity(1, topic)
	sub2 := broker.SubscribeWithCapacity(1, topic)

	if sub1 == sub2 {
		t.Error("want new subscription channel, got same channel")
	}
}

func TestBrokerSubscribeUnbufferedChannelCapacity(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int, string]()

	sub := broker.Subscribe("testing")
	wantCap := 0

	if got := cap(sub); wantCap != got {
		t.Errorf("want channel capacity %d, got %d", wantCap, got)
	}
}

func TestBrokerSubscribeBufferedChannelCapacity(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int, string]()

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
			broker := pubsub.NewBroker[string, string, string]()

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

	broker := pubsub.NewBroker[string, int, string]()
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

	broker := pubsub.NewBroker[string, int, string]()

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

	broker := pubsub.NewBroker[string, string, string]()

	assertTopics := func(want int) {
		t.Helper()
		if got := broker.NumTopics(); want != got {
			t.Fatalf("want topics count %d, got %d", want, got)
		}
	}

	assertTopics(0)

	var subs []<-chan pubsub.Message[string, string, string]
	topic := "testing"
	count := 10

	// Subscribe on the same topic.
	for range count {
		sub := broker.Subscribe(topic)
		subs = append(subs, sub)
		assertTopics(1)
	}

	lastSubIndex := len(subs) - 1

	// Remove all of the subscriptions and keep 1.
	for _, sub := range subs[:lastSubIndex] {
		broker.Unsubscribe(sub, topic)
		assertTopics(1)
	}

	// Remove the last subscription.
	lastSub := subs[lastSubIndex]
	broker.Unsubscribe(lastSub, topic)
	assertTopics(0)
}

func TestBrokerSubscribers(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()

	assertSubs := func(topic string, want int) {
		t.Helper()
		if got := broker.Subscribers(topic); want != got {
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

// assertEqual asserts that the messages want and got are equal.
func assertEqual(t testing.TB, want, got Message) {
	t.Helper()

	if want.Topic != got.Topic {
		t.Errorf("want message topic %q, got %q", want.Topic, got.Topic)
	}

	if want.Payload != got.Payload {
		t.Errorf("want message payload %q, got %q", want.Payload, got.Payload)
	}

	if want.Sender != got.Sender {
		t.Errorf("want message sender %q, got %q", want.Sender, got.Sender)
	}
}

func TestBrokerPublish(t *testing.T) {
	t.Parallel()

	cases := []int{1, 2, 10, 100}

	for _, count := range cases {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			broker := pubsub.NewBroker[string, string, string]()
			want := Message{
				Topic:   "testing",
				Payload: "Test Message",
				Sender:  "test-sender",
			}

			// Subscriptions.
			var subs []<-chan pubsub.Message[string, string, string]

			// Create the subscriptions.
			for range count {
				sub := broker.Subscribe(want.Topic)
				subs = append(subs, sub)
			}

			result := make(chan error)
			go func() {
				// Blocks until all subscribers receive the message.
				result <- broker.Publish(context.Background(), want)
			}()

			// Wait for messages to be received on the subscription channels.
			for _, sub := range subs {
				select {
				case got := <-sub:
					assertEqual(t, want, got)
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

	cases := map[string]Message{
		"with topic": {
			Topic:   "testing",
			Payload: "Message",
		},
		"without topic": {
			Payload: "Message without a topic",
		},
	}

	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			broker := pubsub.NewBroker[string, string, string]()

			result := make(chan error)
			go func() {
				// A publish without any subscriptions should not block.
				result <- broker.Publish(context.Background(), msg)
			}()

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

func TestBrokerPublishWithCanceledContext(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
	topic := "testing"

	// A subscription that we don't receive on.
	broker.Subscribe(topic)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := make(chan error)
	go func() {
		// Publish with a canceled context.
		result <- broker.Publish(ctx, Message{Topic: topic, Payload: "Test"})
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

	broker := pubsub.NewBroker[string, string, string]()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := make(chan error)
	go func() {
		// A publish without any subscriptions should not block.
		result <- broker.Publish(ctx, Message{Topic: "testing", Payload: "Message"})
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

func TestBrokerPublishWithBufferedSubscription(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
	want := Message{
		Topic:   "testing",
		Payload: "Test Message",
		Sender:  "test-sender",
	}

	// Subscription with buffer size of 1.
	sub := broker.SubscribeWithCapacity(1, want.Topic)

	result := make(chan error)
	go func() {
		// Publish with a buffered subscription should not block.
		result <- broker.Publish(context.Background(), want)
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
	assertEqual(t, want, got)
}

func TestBrokerPublishAfterUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
	msg := Message{
		Topic:   "testing",
		Payload: "Test Message",
	}

	sub := broker.Subscribe(msg.Topic)
	// Unsubscribe from the specified topic.
	broker.Unsubscribe(sub, msg.Topic)

	result := make(chan error)
	go func() {
		// Should not block after unsubscribe.
		result <- broker.Publish(context.Background(), msg)
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

	broker := pubsub.NewBroker[string, string, string]()
	msg := Message{
		Topic:   "testing",
		Payload: "Test Message",
	}

	sub := broker.Subscribe(msg.Topic)
	// Unsubscribe from all topics.
	broker.Unsubscribe(sub)

	result := make(chan error)
	go func() {
		// Should not block after unsubscribe.
		result <- broker.Publish(context.Background(), msg)
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

func TestBrokerPublishSlowSubscriber(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
	want := Message{
		Topic:   "testing",
		Payload: "Test Message",
		Sender:  "test-sender",
	}

	// Slow subscriber that will not be ready to receive the message.
	// We will not receive on this channel to simulate a slow subscriber.
	broker.Subscribe(want.Topic)

	// Subscriptions that will receive the message.
	var subs []<-chan pubsub.Message[string, string, string]

	// Create subscriptions that will receive the message after the slow subscriber.
	// This way the channels will be stored after the slow subscriber internally.
	for range 10 {
		subs = append(subs, broker.Subscribe(want.Topic))
	}

	ctx, cancel := context.WithCancel(context.Background())

	result := make(chan error)
	go func() {
		result <- broker.Publish(ctx, want)
	}()

	// Wait for messages to be received on the subscription channels.
	for _, sub := range subs {
		select {
		case got := <-sub:
			assertEqual(t, want, got)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for message")
		}
	}

	// Cancel publishing.
	cancel()

	// Wait for Publish to return.
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Errorf(`want error %q, got "%v"`, context.Canceled, err)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}
}

func TestBrokerTryPublishWithoutSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.TryPublish(Message{Topic: "testing", Payload: "Message"})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("timed out waiting for TryPublish to return")
	}
}

func TestBrokerTryPublishWithUnbufferedSubscription(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()

	want := Message{
		Topic:   "testing",
		Payload: "Test Message",
		Sender:  "test-sender",
	}

	// Unbuffered subscription.
	sub := broker.Subscribe(want.Topic)

	result := make(chan pubsub.Message[string, string, string])
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
		broker.TryPublish(want)
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
		assertEqual(t, want, got)
	case <-time.After(time.Second):
		t.Error("timed out waiting for message")
	}
}

func TestBrokerTryPublishWithUnbufferedSubscriptionNotReady(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()

	topic := "testing"
	// Unbuffered subscription that we don't receive on.
	sub := broker.Subscribe(topic)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Should not block if the subscription channel is not ready to receive.
		broker.TryPublish(Message{Topic: topic, Payload: "Message"})
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

	broker := pubsub.NewBroker[string, string, string]()
	want := Message{
		Topic:   "testing",
		Payload: "Test Message",
		Sender:  "test-sender",
	}

	// Subscription with buffer size of 1.
	sub := broker.SubscribeWithCapacity(1, want.Topic)

	done := make(chan struct{})
	go func() {
		defer close(done)
		broker.TryPublish(want)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}

	got := <-sub
	assertEqual(t, want, got)
}

func TestBrokerConcurrentPublishSubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
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
			total += broker.Subscribers(topic)
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
			results <- broker.Publish(context.Background(), Message{Topic: topic})
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

	broker := pubsub.NewBroker[string, string, string]()
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
			broker.TryPublish(Message{Topic: topic})
		}()
	}

	wg.Wait()
}

func TestBrokerConcurrentSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string, string]()
	topic := "testing"
	totalSubs := 10

	var wg sync.WaitGroup

	subs := make(chan (<-chan pubsub.Message[string, string, string]))

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
