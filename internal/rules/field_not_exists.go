package rules

import (
	"fmt"

	"go_tool/internal/parser"
	"go_tool/internal/report"
	"go_tool/internal/schema"
)

type FieldNotExistsRule struct{}

func (r *FieldNotExistsRule) ID() string {
	return "GORM1002"
}

func (r *FieldNotExistsRule) Apply(ctx Context, ir parser.SQLIR, sch schema.Schema) ([]report.Diagnostic, error) {
	var diags []report.Diagnostic
	for _, col := range ir.Columns {
		table := col.Table
		if table == "" {
			continue
		}
		if !sch.HasTable(table) {
			diags = append(diags, report.Diagnostic{
				File:       ctx.File,
				Line:       ctx.Line,
				Column:     ctx.Column,
				Snippet:    ctx.Snippet,
				Rule:       r.ID(),
				Message:    fmt.Sprintf("table does not exist: %s", table),
				Suggestion: "check table name or update schema",
				Severity:   report.SeverityError,
				Code:       "GORM1002",
				Confidence: report.ConfidenceHigh,
			})
			continue
		}
		if !sch.HasColumn(table, col.Column) {
			diags = append(diags, report.Diagnostic{
				File:       ctx.File,
				Line:       ctx.Line,
				Column:     ctx.Column,
				Snippet:    ctx.Snippet,
				Rule:       r.ID(),
				Message:    fmt.Sprintf("column does not exist: %s.%s", table, col.Column),
				Suggestion: "check column name or update schema",
				Severity:   report.SeverityError,
				Code:       "GORM1002",
				Confidence: report.ConfidenceHigh,
			})
		}
	}
	return diags, nil
}
