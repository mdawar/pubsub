package pubsub_test

import (
	"fmt"
	"testing"

	"github.com/mdawar/pubsub"
)

func TestBrokerInitialState(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[string]()

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

	broker := pubsub.NewBroker[int]()

	topic := "testing"
	sub1 := broker.Subscribe(topic)
	sub2 := broker.Subscribe(topic)

	if sub1 == sub2 {
		t.Error("want new subscription channel, got same channel")
	}
}

func TestBrokerSubscribeWithCapacityOnSameTopicReturnsNewChannel(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[int]()

	topic := "testing"
	sub1 := broker.SubscribeWithCapacity(1, topic)
	sub2 := broker.SubscribeWithCapacity(1, topic)

	if sub1 == sub2 {
		t.Error("want new subscription channel, got same channel")
	}
}

func TestBrokerSubscribeUnbufferedChannelCapacity(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[int]()

	sub := broker.Subscribe("testing")
	wantCap := 0

	if got := cap(sub); wantCap != got {
		t.Errorf("want channel capacity %d, got %d", wantCap, got)
	}
}

func TestBrokerSubscribeBufferedChannelCapacity(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[int]()

	wantCap := 10
	sub := broker.SubscribeWithCapacity(wantCap, "testing")

	if got := cap(sub); wantCap != got {
		t.Errorf("want channel capacity %d, got %d", wantCap, got)
	}
}

func TestBrokerNumTopicsAfterSubscriptions(t *testing.T) {
	t.Parallel()

	broker := pubsub.NewBroker[int]()
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

	broker := pubsub.NewBroker[int]()
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
