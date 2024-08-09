package pubsub_test

import (
	"fmt"
	"testing"

	"github.com/mdawar/pubsub"
)

func TestBrokerInitialState(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string, string]()

	wantSubs := 0
	wantTopics := 0

	if got := broker.NumSubs(); wantSubs != got {
		t.Errorf("want %d initial broker subscriptions, got %d", wantSubs, got)
	}

	if got := broker.NumTopics(); wantTopics != got {
		t.Errorf("want %d initial broker topics, got %d", wantTopics, got)
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

// TODO: test NumTopics decreases after all subscriptions are removed.
// TODO: test NumSubs does not decrease if not all subscriptions are removed.
// TODO: test NumSubs decreases if all subscriptions are removed manually.
