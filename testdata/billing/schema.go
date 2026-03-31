package schema

import "context"

type AddContractRequest struct {
	ContractID int `json:"contract_id" validate:"required,gte=1"`
}

type AddTariffRequest struct {
	TariffID int    `json:"tariff_id" validate:"required,neq=1"`
	Name     string `json:"name" validate:"required,neq=aboba"`
}

type Contract interface {
	AddContract(ctx context.Context, req AddContractRequest) error
}

type Tariff interface {
	AddTariff(ctx context.Context, req AddTariffRequest) error
}
