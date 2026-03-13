package main

import (
	"fmt"

	"go_tool/internal/parser"
	"go_tool/internal/report"
	"go_tool/internal/rules"
	"go_tool/internal/schema"
)

type ExampleRule struct{}

func (r *ExampleRule) ID() string {
	return "EXAMPLE0001"
}

func (r *ExampleRule) Apply(ctx rules.Context, ir parser.SQLIR, sch schema.Schema) ([]report.Diagnostic, error) {
	_ = ir
	_ = sch
	return []report.Diagnostic{{
		File:       ctx.File,
		Line:       ctx.Line,
		Column:     ctx.Column,
		Snippet:    ctx.Snippet,
		Rule:       r.ID(),
		Message:    "example rule fired",
		Suggestion: "replace with real rule",
		Severity:   report.SeverityInfo,
		Code:       "EXAMPLE0001",
		Confidence: report.ConfidenceLow,
	}}, nil
}

// Register is required by the plugin loader.
func Register(reg *rules.Registry) error {
	reg.Register(&ExampleRule{})
	fmt.Println("example plugin registered")
	return nil
}
