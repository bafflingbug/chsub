package chsub

// TopicOption is a function type used to configure TopicOptions.
type TopicOption func(*TopicOptions)

// TopicOptions contains configuration options for a topic.
type TopicOptions struct {
	length int // length specifies the maximum length of the topic.
}

// WithLength returns a TopicOption that sets the maximum length of a topic.
func WithLength(length int) TopicOption {
	return func(opts *TopicOptions) {
		opts.length = length
	}
}

// SubOption is a function type used to configure SubscriberConfig.
type SubOption func(*SubscriberConfig)

// SubscriberConfig contains configuration options for a subscriber.
type SubscriberConfig struct {
	onDrop func(any) // onDrop is called when a message is dropped due to channel full.
}

// WithSubOnDrop returns a SubOption that sets the onDrop callback for a subscriber.
func WithSubOnDrop(f func(any)) SubOption {
	return func(cfg *SubscriberConfig) {
		cfg.onDrop = f
	}
}
