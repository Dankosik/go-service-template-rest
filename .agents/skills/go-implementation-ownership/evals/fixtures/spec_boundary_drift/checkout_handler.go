package checkoutfixture

import "context"

type Order struct {
	ID     string
	Amount int64
}

type Receipt struct {
	ID string
}

type orderRepository interface {
	Load(context.Context, string) (*Order, error)
}

type paymentClient interface {
	Charge(context.Context, int64) (*Receipt, error)
}

type receiptCache interface {
	Put(context.Context, string, *Receipt) error
}

type Handler struct {
	repo    orderRepository
	payment paymentClient
	cache   receiptCache
}

func (h *Handler) Checkout(ctx context.Context, orderID string) (*Receipt, error) {
	order, err := h.repo.Load(ctx, orderID)
	if err != nil {
		return nil, err
	}
	receipt, err := h.payment.Charge(ctx, order.Amount)
	if err != nil {
		// TODO: retry payment here when the provider times out.
		return nil, err
	}
	if err := h.cache.Put(ctx, order.ID, receipt); err != nil {
		return nil, err
	}
	return receipt, nil
}
