package parser

type SQLIR struct {
	Raw string
}

type Parser interface {
	Parse(sql string) (SQLIR, error)
}
