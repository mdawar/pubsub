package pubsub_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/mdawar/pubsub"
)

func TestBrokerInitialNumSubs(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	want := 0

	if got := broker.NumSubs(); want != got {
		t.Errorf("want %d subscriptions, got %d", want, got)
	}
}

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

func TestBrokerNumSubsAfterSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, int]()
	wantSubs := 10
	wantFinalSubs := wantSubs * 2

	for i := range wantSubs {
		broker.Subscribe(fmt.Sprint(i))
	}

	if got := broker.NumSubs(); wantSubs != got {
		t.Errorf("want %d subscriptions, got %d", wantSubs, got)
	}

	// Subscriptions on the same topics should create new subscriptions and increase the count.
	for i := range wantSubs {
		broker.Subscribe(fmt.Sprint(i))
	}

	if got := broker.NumSubs(); wantFinalSubs != got {
		t.Errorf("want %d final subscriptions, got %d", wantFinalSubs, got)
	}
}

func TestBrokerNumSubsDecreasesAfterUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	assertSubs := func(want int) {
		t.Helper()
		if got := broker.NumSubs(); want != got {
			t.Fatalf("want subs count %d, got %d", want, got)
		}
	}

	topic := "testing"

	// First subscription.
	sub1 := broker.Subscribe(topic)
	assertSubs(1)

	// Add 10 more subscriptions on the same topic.
	for range 10 {
		broker.Subscribe(topic)
	}
	assertSubs(11)

	// Remove 1 subscription.
	broker.Unsubscribe(sub1)
	assertSubs(10)
}

func TestBrokerPublish(t *testing.T) {
	t.Parallel()

	cases := map[string]int{
		"1 subscriber":    1,
		"2 subscribers":   2,
		"10 subscribers":  10,
		"100 subscribers": 100,
	}

	for name, count := range cases {
		t.Run(name, func(t *testing.T) {
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

			done := make(chan struct{})
			go func() {
				defer close(done)
				// Blocks until all subscribers receive the message.
				broker.Publish(topic, payload)
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
			case <-done:
			case <-time.After(time.Second):
				t.Error("timed out waiting for Publish to return")
			}
		})
	}
}

func TestBrokerPublishWithoutSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	done := make(chan struct{})
	go func() {
		defer close(done)
		// A publish without any subscriptions should not block.
		broker.Publish("testing", "Message")
	}()

	select {
	case <-done:
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

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Publish with a buffered subscription should not block.
		broker.Publish(topic, payload)
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

func TestBrokerPublishAfterUnsubscribe(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()
	topic := "testing"
	payload := "Test Message"

	sub := broker.Subscribe(topic)
	broker.Unsubscribe(sub, topic)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Should not block after unsubscribe.
		broker.Publish(topic, payload)
	}()

	// Wait for Publish to return.
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("timed out waiting for Publish to return")
	}
}

// TODO: test NumSubs does not decrease if not all subscriptions are removed.
// TODO: test NumSubs decreases if all subscriptions are removed manually.
