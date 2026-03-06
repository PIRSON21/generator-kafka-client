package schema

import "context"

type AddContractRequest struct {
	ContractID string
	Amount     int
}

type NotifyUserRequest struct {
	UserID  string
	Message string
}

type PingRequest struct{}

type BillingService interface {
	AddContract(ctx context.Context, req AddContractRequest) error
	NotifyUser(ctx context.Context, req NotifyUserRequest) error
	Ping(ctx context.Context, req PingRequest) error
}
