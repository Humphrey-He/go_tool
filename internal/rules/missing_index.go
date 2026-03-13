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
		out = append(out, Condition{Column: col.Column, Op: "="})
	}
	return out
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
		if op == "!=" || op == "NOT IN" || strings.HasPrefix(op, "LIKE %") || strings.Contains(op, "LIKE '%") {
			return false
		}
	}
	return true
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}
