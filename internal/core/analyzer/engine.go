package analyzer

import (
	"context"
	"go_tool/internal/config"
	"go_tool/internal/parser"
	"go_tool/internal/report"
	"go_tool/internal/rules"
	"go_tool/internal/schema"
)

type OccurrenceKind string

const (
	OccurrenceKindSQL         OccurrenceKind = "sql"
	OccurrenceKindGormWarning OccurrenceKind = "gorm_warning"
	OccurrenceKindGormWhere   OccurrenceKind = "gorm_where"
)

type Occurrence struct {
	File    string
	Line    int
	Column  int
	Snippet string
	SQL     string
	Kind    OccurrenceKind
}

type SourceCollector interface {
	Collect(ctx context.Context, cfg config.Config) ([]Occurrence, error)
}

type Engine struct {
	Collector SourceCollector
	Parser    parser.Parser
	Schema    schema.Loader
	Rules     *rules.Registry
}

func (e *Engine) Analyze(ctx context.Context, cfg config.Config) ([]report.Diagnostic, error) {
	if e.Collector == nil || e.Parser == nil || e.Schema == nil || e.Rules == nil {
		return nil, ErrInvalidEngine
	}

	occurrences, err := e.Collector.Collect(ctx, cfg)
	if err != nil {
		return nil, err
	}

	sch, err := e.Schema.Load(ctx)
	if err != nil {
		return nil, err
	}

	var diags []report.Diagnostic
	for _, occ := range occurrences {
		if occ.Kind == OccurrenceKindGormWarning {
			diags = append(diags, report.Diagnostic{
				File:       occ.File,
				Line:       occ.Line,
				Column:     occ.Column,
				Snippet:    occ.Snippet,
				Rule:       "GORM2001",
				Message:    "dynamic where clause cannot be parsed",
				Suggestion: "rewrite as constant string or enable rule ignore",
				Severity:   report.SeverityWarn,
				Code:       "GORM2001",
				Confidence: report.ConfidenceLow,
			})
			continue
		}

		if occ.Kind == OccurrenceKindGormWhere {
			ir := parser.SQLIR{Raw: occ.SQL, Tables: nil, Columns: []parser.ColumnRef{{Table: "", Column: occ.SQL}}}
			ctxRule := rules.Context{
				File:    occ.File,
				Line:    occ.Line,
				Column:  occ.Column,
				Snippet: occ.Snippet,
			}
			for _, rule := range e.Rules.All() {
				results, err := rule.Apply(ctxRule, ir, sch)
				if err != nil {
					diags = append(diags, report.Diagnostic{
						File:       occ.File,
						Line:       occ.Line,
						Column:     occ.Column,
						Snippet:    occ.Snippet,
						Rule:       rule.ID(),
						Message:    err.Error(),
						Severity:   report.SeverityError,
						Code:       "RULE_ERROR",
						Confidence: report.ConfidenceLow,
					})
					continue
				}
				diags = append(diags, results...)
			}
			continue
		}

		ir, err := e.Parser.Parse(occ.SQL)
		if err != nil {
			diags = append(diags, report.Diagnostic{
				File:       occ.File,
				Line:       occ.Line,
				Column:     occ.Column,
				Snippet:    occ.Snippet,
				Rule:       "PARSER",
				Message:    err.Error(),
				Severity:   report.SeverityError,
				Code:       "PARSER_ERROR",
				Confidence: report.ConfidenceLow,
			})
			continue
		}

		ctxRule := rules.Context{
			File:    occ.File,
			Line:    occ.Line,
			Column:  occ.Column,
			Snippet: occ.Snippet,
		}

		for _, rule := range e.Rules.All() {
			results, err := rule.Apply(ctxRule, ir, sch)
			if err != nil {
				diags = append(diags, report.Diagnostic{
					File:       occ.File,
					Line:       occ.Line,
					Column:     occ.Column,
					Snippet:    occ.Snippet,
					Rule:       rule.ID(),
					Message:    err.Error(),
					Severity:   report.SeverityError,
					Code:       "RULE_ERROR",
					Confidence: report.ConfidenceLow,
				})
				continue
			}
			diags = append(diags, results...)
		}
	}

	return diags, nil
}
