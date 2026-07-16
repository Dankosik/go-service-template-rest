package evalfixture

import (
	"context"
	"errors"
)

var (
	ErrMissingID = errors.New("missing order id")
	ErrInactive  = errors.New("inactive order")
)

type Order struct {
	ID     string
	Active bool
	Status string
}

type orderRepository interface {
	Load(context.Context, string) (*Order, error)
}

type charger interface {
	Charge(context.Context, *Order) error
}

type auditor interface {
	Record(context.Context, string, error) error
}

type Handler struct {
	repo    orderRepository
	charger charger
	auditor auditor
}

func (h *Handler) Handle(ctx context.Context, id string) (*Order, error) {
	var order *Order
	var mainErr error
	var auditErr error

	if id != "" {
		order, mainErr = h.repo.Load(ctx, id)
		if mainErr == nil {
			if order.Active {
				mainErr = h.charger.Charge(ctx, order)
				if mainErr == nil {
					order.Status = "charged"
				}
			} else {
				mainErr = ErrInactive
			}
		}
	} else {
		mainErr = ErrMissingID
	}

	auditErr = h.auditor.Record(ctx, id, mainErr)
	if mainErr != nil {
		return nil, mainErr
	}
	if auditErr != nil {
		return nil, auditErr
	}
	return order, nil
}
