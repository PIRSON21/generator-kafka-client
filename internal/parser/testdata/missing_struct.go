package schema

import "context"

type PingService interface {
	Ping(ctx context.Context, req PingRequest) error
}
