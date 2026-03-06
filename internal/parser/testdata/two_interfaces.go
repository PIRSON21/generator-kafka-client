package schema

import "context"

type PingRequest struct{}
type FooRequest struct{}

type FirstService interface {
	Ping(ctx context.Context, req PingRequest) error
}

type SecondService interface {
	Foo(ctx context.Context, req FooRequest) error
}
