package rules

import (
	"fmt"
	"strings"

	"go_tool/internal/parser"
	"go_tool/internal/report"
	"go_tool/internal/schema"
)

type MissingIndexRule struct {
	SearchPath []string
}

func (r *MissingIndexRule) ID() string {
	return "GORM1001"
}

func (r *MissingIndexRule) Apply(ctx Context, ir parser.SQLIR, sch schema.Schema) ([]report.Diagnostic, error) {
	var diags []report.Diagnostic
	for _, tbl := range ir.Tables {
		table, ok := sch.FindTable(tbl.Name, r.SearchPath)
		if !ok {
			continue
		}

		conditions := columnConditions(ir, tbl.Name)
		if len(conditions) == 0 {
			continue
		}

		conditions = injectSoftDelete(table, conditions)

		if hasHighRiskOp(conditions) {
			diags = append(diags, report.Diagnostic{
				File:       ctx.File,
				Line:       ctx.Line,
				Column:     ctx.Column,
				Snippet:    ctx.Snippet,
				Rule:       r.ID(),
				Message:    "operator may disable index usage",
				Suggestion: "consider rewriting condition or adding computed index",
				Severity:   report.SeverityWarn,
				Code:       "GORM1003",
				Confidence: report.ConfidenceHigh,
			})
		}

		if suggestGIN(table, conditions) {
			diags = append(diags, report.Diagnostic{
				File:       ctx.File,
				Line:       ctx.Line,
				Column:     ctx.Column,
				Snippet:    ctx.Snippet,
				Rule:       r.ID(),
				Message:    "jsonb query without GIN index",
				Suggestion: fmt.Sprintf("CREATE INDEX idx_%s_jsonb_gin ON %s USING GIN (%s);", sanitizeName(table.Name), table.Name, conditions[0].Column),
				Severity:   report.SeverityWarn,
				Code:       "GORM1004",
				Confidence: report.ConfidenceHigh,
			})
		}

		if hasMatchingIndex(table, conditions) {
			continue
		}

		if hasCoveringIndex(table, conditions) {
			continue
		}

		cols := make([]string, 0, len(conditions))
		for _, c := range conditions {
			cols = append(cols, c.Column)
		}

		suggestion := fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s(%s);", sanitizeName(table.Name), strings.Join(cols, "_"), table.Name, strings.Join(cols, ", "))
		diags = append(diags, report.Diagnostic{
			File:       ctx.File,
			Line:       ctx.Line,
			Column:     ctx.Column,
			Snippet:    ctx.Snippet,
			Rule:       r.ID(),
			Message:    fmt.Sprintf("missing index for table %s", table.Name),
			Suggestion: suggestion,
			Severity:   report.SeverityWarn,
			Code:       "GORM1001",
			Confidence: report.ConfidenceMedium,
		})
	}
	return diags, nil
}

type Condition struct {
	Column string
	Op     string
	Type   string
}

func columnConditions(ir parser.SQLIR, table string) []Condition {
	var out []Condition
	for _, col := range ir.Columns {
		if col.Table != "" && col.Table != table {
			continue
		}
		if col.Column == "" {
			continue
		}
		out = append(out, Condition{Column: col.Column, Op: col.Op})
	}
	return out
}

func injectSoftDelete(table schema.Table, conditions []Condition) []Condition {
	if _, ok := table.Columns["deleted_at"]; !ok {
		return conditions
	}
	for _, c := range conditions {
		if c.Column == "deleted_at" {
			return conditions
		}
	}
	return append([]Condition{{Column: "deleted_at", Op: "="}}, conditions...)
}

func hasMatchingIndex(table schema.Table, conditions []Condition) bool {
	for _, idx := range table.Indexes {
		if isLeftPrefixMatch(idx.Columns, conditions) {
			if isOperatorSafe(conditions) {
				return true
			}
		}
	}
	return false
}

func hasCoveringIndex(table schema.Table, conditions []Condition) bool {
	for _, idx := range table.Indexes {
		if isLeftPrefixMatch(idx.Columns, conditions) || isPrefixCovered(idx.Columns, conditions) {
			return true
		}
	}
	return false
}

func isLeftPrefixMatch(indexCols []string, conditions []Condition) bool {
	if len(indexCols) == 0 || len(conditions) == 0 {
		return false
	}
	if len(conditions) > len(indexCols) {
		return false
	}
	for i, cond := range conditions {
		if i >= len(indexCols) {
			return false
		}
		if indexCols[i] != cond.Column {
			return false
		}
	}
	return true
}

func isPrefixCovered(indexCols []string, conditions []Condition) bool {
	if len(indexCols) == 0 || len(conditions) == 0 {
		return false
	}
	for i, col := range indexCols {
		if i >= len(conditions) {
			return true
		}
		if col != conditions[i].Column {
			return false
		}
	}
	return true
}

func isOperatorSafe(conditions []Condition) bool {
	for _, c := range conditions {
		op := strings.ToUpper(strings.TrimSpace(c.Op))
		if op == "!=" || op == "<>" || op == "NOT IN" || strings.Contains(op, "LIKE") {
			return false
		}
	}
	return true
}

func hasHighRiskOp(conditions []Condition) bool {
	for _, c := range conditions {
		op := strings.ToUpper(strings.TrimSpace(c.Op))
		if op == "!=" || op == "<>" || op == "NOT IN" {
			return true
		}
		if strings.Contains(op, "LIKE") && strings.Contains(strings.ToUpper(c.Op), "%") {
			return true
		}
	}
	return false
}

func suggestGIN(table schema.Table, conditions []Condition) bool {
	for _, cond := range conditions {
		col, ok := table.Columns[cond.Column]
		if !ok {
			continue
		}
		if !strings.Contains(strings.ToLower(col.Type), "jsonb") {
			continue
		}
		op := strings.ToUpper(strings.TrimSpace(cond.Op))
		if op == "@>" || op == "?" || op == "?|" || op == "?&" || op == "->" || op == "->>" {
			if !hasGINIndex(table, cond.Column) {
				return true
			}
		}
	}
	return false
}

func hasGINIndex(table schema.Table, column string) bool {
	for _, idx := range table.Indexes {
		if strings.ToUpper(idx.Method) != "GIN" {
			continue
		}
		for _, col := range idx.Columns {
			if col == column {
				return true
			}
		}
	}
	return false
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}
