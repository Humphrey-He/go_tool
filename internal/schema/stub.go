package schema

import "context"

type StubLoader struct{}

func (l *StubLoader) Load(ctx context.Context) (Schema, error) {
	return Schema{}, nil
}
