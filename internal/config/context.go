package config

import (
	"context"
	"fmt"
)

// checkContext classifies caller cancellation during config loading.
func checkContext(ctx context.Context) error {
	return checkContextWithError(ctx, ErrLoad)
}

func checkValidateContext(ctx context.Context) error {
	return checkContextWithError(ctx, ErrValidate)
}

func checkContextWithError(ctx context.Context, classification error) error {
	if ctx == nil {
		return fmt.Errorf("%w: nil context", classification)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %w", classification, err)
	}
	return nil
}
