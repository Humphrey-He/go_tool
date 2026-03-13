package schema

type Column struct {
	Name string
}

type Table struct {
	Name    string
	Columns map[string]Column
}

type Schema struct {
	Tables map[string]Table
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
