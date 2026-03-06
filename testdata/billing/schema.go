package schema

import "context"

type AddContractRequest struct {
	ContractID int
}

type AddTariffRequest struct {
	TariffID int
	Name     string
}

type Contract interface {
	AddContract(ctx context.Context, req AddContractRequest) error
}

type Tariff interface {
	AddTariff(ctx context.Context, req AddTariffRequest) error
}
