package parser

type TableRef struct {
	Name string
}

type ColumnRef struct {
	Table  string
	Column string
	Op     string
}

type SQLIR struct {
	Raw     string
	Tables  []TableRef
	Columns []ColumnRef
}

type Parser interface {
	Parse(sql string) (SQLIR, error)
}
