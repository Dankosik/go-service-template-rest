package accountfixture

import (
	"context"
	"errors"
)

type Account struct {
	ID       string
	TenantID string
}

type accountCache interface {
	Get(context.Context, string) (*Account, bool)
	Put(context.Context, string, *Account) error
}

type accountRepository interface {
	Get(context.Context, string) (*Account, error)
}

type AccountReader struct {
	cache accountCache
	repo  accountRepository
}

func (r *AccountReader) Read(ctx context.Context, tenantID, accountID string) (*Account, error) {
	key := "account:" + accountID
	cached, cachedOK := r.cache.Get(ctx, key)
	account, err := r.repo.Get(ctx, accountID)
	if errors.Is(err, context.DeadlineExceeded) && cachedOK {
		return cached, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.cache.Put(ctx, key, account); err != nil {
		return nil, err
	}
	return account, nil
}
