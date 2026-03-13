package parser

type StubParser struct{}

func (p *StubParser) Parse(sql string) (SQLIR, error) {
	return SQLIR{Raw: sql}, nil
}
