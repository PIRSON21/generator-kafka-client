package schema

type PingRequest struct{}

type PingService interface {
	Ping(req PingRequest) error
}
