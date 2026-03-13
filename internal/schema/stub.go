package schema

import (
	"context"
	"fmt"
)

type StubLoader struct{}

func (l *StubLoader) Load(ctx context.Context) (Schema, error) {
	_ = ctx
	return Schema{}, fmt.Errorf("stub loader not implemented")
}
