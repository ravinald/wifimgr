package api

import (
	"context"
)

// Wait extends the rateLimiter to accept a context for backward compatibility
// with the existing mock client implementation
func (r *rateLimiter) Wait(ctx context.Context) error {
	// If context is done, return error
	if err := ctx.Err(); err != nil {
		return err
	}

	// Call the underlying wait method
	r.wait()
	return nil
}
