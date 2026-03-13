package rules

import (
	"fmt"

	"go_tool/internal/parser"
	"go_tool/internal/report"
	"go_tool/internal/schema"
)

type FieldNotExistsRule struct{}

func (r *FieldNotExistsRule) ID() string {
	return "FIELD_NOT_EXISTS"
}

func (r *FieldNotExistsRule) Apply(ctx Context, ir parser.SQLIR, sch schema.Schema) ([]report.Diagnostic, error) {
	var diags []report.Diagnostic
	for _, col := range ir.Columns {
		if col.Table == "" {
			continue
		}
		if !sch.HasTable(col.Table) {
			diags = append(diags, report.Diagnostic{
				File:       ctx.File,
				Line:       ctx.Line,
				Column:     ctx.Column,
				Snippet:    ctx.Snippet,
				Rule:       r.ID(),
				Message:    fmt.Sprintf("table does not exist: %s", col.Table),
				Suggestion: "check table name or update schema",
				Severity:   report.SeverityError,
				Code:       "TABLE_NOT_EXISTS",
			})
			continue
		}
		if !sch.HasColumn(col.Table, col.Column) {
			diags = append(diags, report.Diagnostic{
				File:       ctx.File,
				Line:       ctx.Line,
				Column:     ctx.Column,
				Snippet:    ctx.Snippet,
				Rule:       r.ID(),
				Message:    fmt.Sprintf("column does not exist: %s.%s", col.Table, col.Column),
				Suggestion: "check column name or update schema",
				Severity:   report.SeverityError,
				Code:       "COLUMN_NOT_EXISTS",
			})
		}
	}
	return diags, nil
}
