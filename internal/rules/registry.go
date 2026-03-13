package rules

import (
	"go_tool/internal/parser"
	"go_tool/internal/report"
	"go_tool/internal/schema"
)

type Context struct {
	File    string
	Line    int
	Column  int
	Snippet string
}

type Rule interface {
	ID() string
	Apply(ctx Context, ir parser.SQLIR, schema schema.Schema) ([]report.Diagnostic, error)
}

type Registry struct {
	rules map[string]Rule
}

func NewRegistry() *Registry {
	return &Registry{rules: map[string]Rule{}}
}

func (r *Registry) Register(rule Rule) {
	if rule == nil {
		return
	}
	r.rules[rule.ID()] = rule
}

func (r *Registry) All() []Rule {
	items := make([]Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		items = append(items, rule)
	}
	return items
}

func (r *Registry) Filtered(enable, disable []string) []Rule {
	enabled := map[string]bool{}
	disabled := map[string]bool{}
	for _, id := range enable {
		enabled[id] = true
	}
	for _, id := range disable {
		disabled[id] = true
	}

	items := make([]Rule, 0, len(r.rules))
	for id, rule := range r.rules {
		if disabled[id] {
			continue
		}
		if len(enabled) > 0 && !enabled[id] {
			continue
		}
		items = append(items, rule)
	}
	return items
}
