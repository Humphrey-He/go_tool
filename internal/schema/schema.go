package schema

import "context"

type Column struct {
	Name string `json:"name"`
}

type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Method  string   `json:"method"`
}

type Table struct {
	Name    string            `json:"name"`
	Columns map[string]Column `json:"columns"`
	Indexes map[string]Index  `json:"indexes"`
}

type Schema struct {
	Tables map[string]Table `json:"tables"`
}

func NewSchema() Schema {
	return Schema{Tables: map[string]Table{}}
}

func (s Schema) HasTable(name string) bool {
	_, ok := s.Tables[name]
	return ok
}

func (s Schema) HasColumn(table, column string) bool {
	t, ok := s.Tables[table]
	if !ok {
		return false
	}
	_, ok = t.Columns[column]
	return ok
}

func (s Schema) FindTable(table string, searchPath []string) (Table, bool) {
	if t, ok := s.Tables[table]; ok {
		return t, true
	}
	for _, sp := range searchPath {
		name := sp + "." + table
		if t, ok := s.Tables[name]; ok {
			return t, true
		}
	}
	return Table{}, false
}

type Loader interface {
	Load(ctx context.Context) (Schema, error)
}
