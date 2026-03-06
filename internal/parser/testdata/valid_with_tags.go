package schema

import "context"

type CreateOrderRequest struct {
	OrderID string `json:"order_id,omitempty"`
	Amount  int    `json:"amount"`
}

type CancelOrderRequest struct {
	OrderID string
}

type OrderService interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) error
	CancelOrder(ctx context.Context, req CancelOrderRequest) error
}
