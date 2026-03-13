package parser

import (
	"fmt"

	"github.com/xwb1989/sqlparser"
)

type TableRef struct {
	Name string
}

type ColumnRef struct {
	Table  string
	Column string
}

type SQLIR struct {
	Raw     string
	Tables  []TableRef
	Columns []ColumnRef
}

func ParseVitess(sql string) (SQLIR, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return SQLIR{}, err
	}

	ir := SQLIR{Raw: sql}
	walker := func(node sqlparser.SQLNode) (bool, error) {
		switch n := node.(type) {
		case sqlparser.TableName:
			if n.Name.String() != "" {
				ir.Tables = append(ir.Tables, TableRef{Name: n.Name.String()})
			}
		case *sqlparser.ColName:
			col := n.Name.String()
			if col != "" {
				ir.Columns = append(ir.Columns, ColumnRef{
					Table:  n.Qualifier.Name.String(),
					Column: col,
				})
			}
		case sqlparser.TableExpr:
			_ = n
		case sqlparser.Expr:
			_ = n
		case *sqlparser.AliasedTableExpr:
			_ = n
		case *sqlparser.TableName:
			// no-op
		default:
			_ = n
		}
		return true, nil
	}

	if err := sqlparser.Walk(walker, stmt); err != nil {
		return SQLIR{}, fmt.Errorf("walk sql: %w", err)
	}

	return ir, nil
}

type VitessParser struct{}

func (p *VitessParser) Parse(sql string) (SQLIR, error) {
	return ParseVitess(sql)
}
