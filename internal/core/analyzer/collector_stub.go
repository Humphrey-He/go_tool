package analyzer

import (
	"context"
	"go_tool/internal/config"
)

type StubCollector struct{}

func (c *StubCollector) Collect(ctx context.Context, cfg config.Config) ([]Occurrence, error) {
	return []Occurrence{}, nil
}
