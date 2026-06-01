//go:build nofirstrun

package app

import (
	"context"
)

func firstRun(ctx context.Context) error {
	return nil
}
