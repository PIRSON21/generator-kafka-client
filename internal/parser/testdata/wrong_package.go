package notschema

import "context"

type PingRequest struct{}

type PingService interface {
	Ping(ctx context.Context, req PingRequest) error
}
