package analyzer

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

var gormMethodSet = map[string]struct{}{
	"Model":  {},
	"Table":  {},
	"Where":  {},
	"Select": {},
	"Joins":  {},
	"Group":  {},
	"Having": {},
	"Order":  {},
	"Limit":  {},
	"Offset": {},
}

type GormCall struct {
	Method string
	Args   []ast.Expr
	Pos    token.Pos
}

type GormChain struct {
	VarName string
	Calls   []GormCall
}

func ExtractGormChains(file *ast.File) []GormChain {
	chains := map[string]*GormChain{}

	register := func(name string, call *ast.CallExpr) {
		callExpr, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		method := callExpr.Sel.Name
		if _, ok := gormMethodSet[method]; !ok {
			return
		}
		chain := chains[name]
		if chain == nil {
			chain = &GormChain{VarName: name}
			chains[name] = chain
		}
		chain.Calls = append(chain.Calls, GormCall{Method: method, Args: call.Args, Pos: call.Pos()})
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			if len(node.Lhs) != 1 || len(node.Rhs) != 1 {
				return true
			}
			lhsIdent, ok := node.Lhs[0].(*ast.Ident)
			if !ok {
				return true
			}
			rhsCall, ok := node.Rhs[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			if !isGormSelectorCall(rhsCall) {
				return true
			}

			if recv, ok := receiverIdent(rhsCall); ok {
				if prev := chains[recv.Name]; prev != nil {
					chain := &GormChain{VarName: lhsIdent.Name}
					chain.Calls = append(chain.Calls, prev.Calls...)
					chains[lhsIdent.Name] = chain
				}
			}

			register(lhsIdent.Name, rhsCall)
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if recv, ok := sel.X.(*ast.Ident); ok {
					if isGormSelectorCall(node) {
						register(recv.Name, node)
					}
				}
			}
		}
		return true
	})

	var out []GormChain
	for _, chain := range chains {
		if len(chain.Calls) == 0 {
			continue
		}
		sort.Slice(chain.Calls, func(i, j int) bool {
			return chain.Calls[i].Pos < chain.Calls[j].Pos
		})
		out = append(out, *chain)
	}
	return out
}

func receiverIdent(call *ast.CallExpr) (*ast.Ident, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil, false
	}
	return ident, true
}

func isGormSelectorCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	method := sel.Sel.Name
	_, ok = gormMethodSet[method]
	if !ok {
		return false
	}
	return true
}

func ExtractWhereFields(arg ast.Expr) (string, string, bool) {
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", "", false
	}
	raw := strings.Trim(lit.Value, "`\"")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", false
	}

	op := detectOperator(raw)
	field := raw
	for i, ch := range raw {
		if ch == ' ' || ch == '\t' || ch == '=' || ch == '>' || ch == '<' {
			field = raw[:i]
			break
		}
	}
	field = strings.TrimSpace(field)
	return field, op, field != ""
}

func detectOperator(raw string) string {
	upper := strings.ToUpper(raw)
	ops := []string{" NOT IN ", " IN ", " LIKE ", " ILIKE ", " != ", " <> ", " >= ", " <= ", " = ", " > ", " < ", " @> ", " ?| ", " ?& ", " ? ", " ->> ", " -> "}
	for _, op := range ops {
		if strings.Contains(upper, strings.TrimSpace(op)) {
			return strings.TrimSpace(op)
		}
	}
	return "="
}

func IsDynamicString(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.BasicLit:
		return false
	case *ast.BinaryExpr:
		return true
	case *ast.CallExpr:
		return true
	case *ast.Ident:
		return true
	default:
		return true
	}
}

func ExtractModelTable(arg ast.Expr) (string, bool) {
	// Table("users")
	if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		name := strings.Trim(lit.Value, "`\"")
		return name, name != ""
	}

	// Model(&User{}) or Model(User{})
	switch expr := arg.(type) {
	case *ast.UnaryExpr:
		if comp, ok := expr.X.(*ast.CompositeLit); ok {
			if ident, ok := comp.Type.(*ast.Ident); ok {
				return toSnakePlural(ident.Name), true
			}
		}
	case *ast.CompositeLit:
		if ident, ok := expr.Type.(*ast.Ident); ok {
			return toSnakePlural(ident.Name), true
		}
	}

	return "", false
}

func toSnakePlural(name string) string {
	var out []rune
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, rune(strings.ToLower(string(r))[0]))
	}
	res := string(out)
	if !strings.HasSuffix(res, "s") {
		res += "s"
	}
	return res
}
