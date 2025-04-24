package chsub

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestSub global function default description
func TestSub(t *testing.T) {
	tests := []struct {
		name       string
		reg        []string
		sub        map[string]string
		regPublish map[string][]interface{}
		subValue   map[string]struct {
			v   []interface{}
			err error
		}
	}{
		{
			name: "test-1",
			reg:  []string{"test_1_chan"},
			sub: map[string]string{
				"test_1_sub_1": "test_1_chan",
				"test_1_sub_2": "test_1_chan",
				"test_1_sub_3": "test_1_chan_3",
			},
			regPublish: map[string][]interface{}{
				"test_1_chan": {true, "ok"},
			},
			subValue: map[string]struct {
				v   []interface{}
				err error
			}{
				"test_1_sub_1": {[]interface{}{true, "ok"}, nil},
				"test_1_sub_2": {[]interface{}{true, "ok"}, nil},
				"test_1_sub_3": {[]interface{}{}, ErrNotRegistered},
			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := NewSub()
				reg := make(map[string]Publish)
				for _, name := range tt.reg {
					publish, err := got.NewTopic(name)
					reg[name] = publish
					if err != nil {
						t.Error(err)
						return
					}
				}
				for name, regName := range tt.sub {
					tmp := make(chan interface{}, 1)
					err := got.Sub(regName, tmp)
					if err != nil && errors.Is(err, tt.subValue[name].err) {
						t.Logf("name: %s, err %+v", name, err)
						continue
					}
					if err != nil {
						t.Error(err)
						continue
					}
					go func(ch chan interface{}, name string) {
						ctx, cancel := context.WithTimeout(context.Background(), time.Second)
						defer cancel()
						i := 0
						for {
							select {
							case v := <-tmp:
								if v != tt.subValue[name].v[i] {
									t.Error(err)
									return
								}
								i++
								t.Logf("name: %s, v: %+v", name, v)
							case <-ctx.Done():
								return
							}
						}
					}(tmp, name)
				}
				for name, v := range tt.regPublish {
					publish := reg[name]
					for _, vv := range v {
						if err := publish(vv); err != nil {
							t.Error(err)
							return
						}
						time.Sleep(200 * time.Millisecond)
					}
				}
			},
		)
		time.Sleep(2 * time.Second)
	}
}

// TestTopicOptions tests the WithLength option for topics.
func TestTopicOptions(t *testing.T) {
	sub := NewSub()
	publish, err := sub.NewTopic("test_topic", WithLength(3))
	if err != nil {
		t.Fatal(err)
	}

	ch := make(chan interface{}, 1)
	err = sub.Sub("test_topic", ch)
	if err != nil {
		t.Fatal(err)
	}

	// Test buffer full
	for _, msg := range []string{"msg1", "msg2", "msg3"} {
		if err = publish(msg); err != nil {
			t.Error(err)
		}
	}

	select {
	case v := <-ch:
		t.Logf("Received: %v", v)
	case <-time.After(time.Second):
		t.Error("Expected message not received")
	}
}

// TestSubscriberOptions tests the WithSubOnDrop option for subscribers.
func TestSubscriberOptions(t *testing.T) {
	sub := NewSub()
	publish, err := sub.NewTopic("test_topic")
	if err != nil {
		t.Fatal(err)
	}

	dropped := false
	dropHandler := func(v any) {
		dropped = true
		t.Logf("Message dropped: %v", v)
	}

	ch := make(chan interface{}, 1)
	err = sub.Sub("test_topic", ch, WithSubOnDrop(dropHandler))
	if err != nil {
		t.Fatal(err)
	}

	// Fill the channel
	if err = publish("msg1"); err != nil {
		t.Error(err)
	}

	// Wait for the message to be received
	time.Sleep(500 * time.Millisecond)

	// This message should be dropped
	if err = publish("msg2"); err != nil {
		t.Error(err)
	}

	// Wait for the drop handler to be called
	time.Sleep(500 * time.Millisecond)

	if !dropped {
		t.Error("Expected message to be dropped")
	}
}

// TestConcurrentSub tests concurrent subscription and publishing.
func TestConcurrentSub(t *testing.T) {
	sub := NewSub()
	publish, err := sub.NewTopic("test_topic")
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		ch := make(chan any, 1)
		err := sub.Sub("test_topic", ch)
		if err != nil {
			t.Error(err)
			return
		}
		go func(i int, ch chan any) {
			defer wg.Done()
			select {
			case v := <-ch:
				t.Logf("Subscriber %d received: %v", i, v)
			case <-time.After(time.Second):
				t.Error("Expected message not received")
			}
		}(i, ch)
	}

	if err = publish("concurrent_msg"); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

// TestErrorHandling tests error cases like duplicate registration and unregistered topics.
func TestErrorHandling(t *testing.T) {
	sub := NewSub()
	_, err := sub.NewTopic("test_topic")
	if err != nil {
		t.Fatal(err)
	}

	// Duplicate registration
	_, err = sub.NewTopic("test_topic")
	if !errors.Is(err, ErrRegistered) {
		t.Errorf("Expected ErrRegistered, got %v", err)
	}

	// Unregistered topic
	err = sub.Sub("unregistered_topic", make(chan interface{}))
	if !errors.Is(err, ErrNotRegistered) {
		t.Errorf("Expected ErrNotRegistered, got %v", err)
	}
}
