// Package chsub implements a channel subscriber pattern.
package chsub

import (
	"errors"
	"sync"
)

var (
	// ErrRegistered indicates that the topic has already been registered.
	ErrRegistered = errors.New("name has been registered")
	// ErrNotRegistered indicates that the subscribed topic is not registered.
	ErrNotRegistered = errors.New("name has not been registered")
	// ErrTopicClosed indicates that the topic is closed.
	ErrTopicClosed = errors.New("topic is closed")
	// ErrChannelNotFound indicates that the channel is not found in subscribers.
	ErrChannelNotFound = errors.New("channel not found in subscribers")
	// ErrChannelFull indicates that the channel is full.
	ErrChannelFull = errors.New("channel is full")
)

// Publish defines a function type for publishing data to a topic.
type Publish func(any) error

// subscriber represents a single subscriber to a topic.
// It holds the channel for receiving messages and the subscriber's configuration.
type subscriber struct {
	ch  chan any
	cfg SubscriberConfig
}

// topic represents a topic instance with a source channel and destination channels.
type topic struct {
	source      chan any
	subscribers []subscriber
	subLock     sync.RWMutex
	closeChan   chan struct{}
	TopicOptions
}

func newTopic(opts ...TopicOption) *topic {
	options := TopicOptions{
		length: 1, // Default length
	}
	for _, opt := range opts {
		opt(&options)
	}
	ch := make(chan any, options.length)
	t := &topic{
		source:       ch,
		subscribers:  make([]subscriber, 0),
		closeChan:    make(chan struct{}),
		TopicOptions: options,
	}
	go t.run()
	return t
}

func (t *topic) run() {
	defer close(t.source)
	for {
		select {
		case v := <-t.source:
			_ = t.publish(v)
		case <-t.closeChan:
			for _, sub := range t.subscribers {
				close(sub.ch)
			}
			return
		}
	}
}

func (t *topic) publish(v any) error {
	t.subLock.RLock()
	defer t.subLock.RUnlock()
	select {
	case <-t.closeChan:
		return ErrTopicClosed
	default:
	}
	for _, sub := range t.subscribers {
		select {
		case sub.ch <- v:
		default:
			if sub.cfg.onDrop != nil {
				sub.cfg.onDrop(v)
			}
		}
	}
	return nil
}

// Sub implements a simple publish/subscribe model.
type Sub struct {
	topics map[string]*topic
	lock   sync.RWMutex
}

// NewSub creates a new Sub instance.
func NewSub() *Sub {
	return &Sub{
		topics: map[string]*topic{},
	}
}

// NewTopic registers a new upstream channel for a topic.
// Returns a Publish function for publishing data to the topic and an error if the topic is already registered.
func (s *Sub) NewTopic(topic string, opts ...TopicOption) (Publish, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.topics[topic]; ok {
		return nil, ErrRegistered
	}
	t := newTopic(opts...)
	s.topics[topic] = t
	return publishFunc(t.source), nil
}

// Sub subscribes a downstream channel to a specified topic.
// Returns the subscribed channel and an error if the topic is not registered.
func (s *Sub) Sub(topic string, ch chan any, opts ...SubOption) error {
	s.lock.RLock()
	t, ok := s.topics[topic]
	s.lock.RUnlock()
	if !ok {
		return ErrNotRegistered
	}

	cfg := SubscriberConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	t.subLock.Lock()
	t.subscribers = append(t.subscribers, subscriber{ch, cfg})
	t.subLock.Unlock()
	return nil
}

// Unsub removes a subscriber channel from a specified topic.
// Returns an error if the topic or channel is not found.
func (s *Sub) Unsub(topic string, ch chan any) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	t, ok := s.topics[topic]
	if !ok {
		return ErrNotRegistered
	}
	t.subLock.Lock()
	defer t.subLock.Unlock()
	for i, sub := range t.subscribers {
		if sub.ch == ch {
			t.subscribers = append(t.subscribers[:i], t.subscribers[i+1:]...)
			return nil
		}
	}
	return ErrChannelNotFound
}

// UnsubAll unsubscribes the given channel from all topics.
func (s *Sub) UnsubAll(ch chan any) {
	s.lock.RLock()
	topics := make([]*topic, 0, len(s.topics))
	for _, t := range s.topics {
		topics = append(topics, t)
	}
	s.lock.RUnlock()

	for _, t := range topics {
		t.subLock.Lock()
		for i, sub := range t.subscribers {
			if sub.ch == ch {
				t.subscribers = append(t.subscribers[:i], t.subscribers[i+1:]...)
				break
			}
		}
		t.subLock.Unlock()
	}
}

// Close terminates all topics and releases associated resources.
func (s *Sub) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, t := range s.topics {
		t.closeChan <- struct{}{}
	}
	s.topics = nil
}

// GetTopicsByChan returns the names of topics subscribed by the given channel.
func (s *Sub) GetTopicsByChan(ch chan any) []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var topics []string
	for name, t := range s.topics {
		t.subLock.RLock()
		for _, sub := range t.subscribers {
			if sub.ch == ch {
				topics = append(topics, name)
				break
			}
		}
		t.subLock.RUnlock()
	}
	return topics
}

// CloseTopic closes a specified topic and releases its resources.
// Returns an error if the topic is not found.
func (s *Sub) CloseTopic(topic string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	t, ok := s.topics[topic]
	if !ok {
		return ErrNotRegistered
	}
	t.closeChan <- struct{}{}
	delete(s.topics, topic)
	return nil
}

func publishFunc(ch chan any) Publish {
	return func(i any) error {
		select {
		case ch <- i:
			return nil
		default:
			return ErrChannelFull
		}
	}
}
