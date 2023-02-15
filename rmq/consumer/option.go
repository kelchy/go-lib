package consumer

import (
	"context"
)

type options struct {
	// for alternative data
	Context        context.Context
	optionalParams []kv
}

type kv struct {
	Key   interface{}
	Value interface{}
}

// pass cloption to config struct
type option func(o *options)

func newOptions(opts ...option) options {
	options := options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

	return options
}

type multipleKey struct{}

func withMultiple(a bool) option {
	return func(o *options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, multipleKey{}, a)
	}
}

type requeueKey struct{}

func withRequeue(a bool) option {
	return func(o *options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, requeueKey{}, a)
	}
}
