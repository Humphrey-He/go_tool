package analyzer

import (
	"go/ast"
	"go/token"
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
			if isGormSelectorCall(rhsCall) {
				register(lhsIdent.Name, rhsCall)
			}
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
		out = append(out, *chain)
	}
	return out
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

func ExtractWhereFields(arg ast.Expr) (string, bool) {
	lit, ok := arg.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	raw := strings.Trim(lit.Value, "`\"")
	// naive parse: take first token before space or operator
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	tok := raw
	for i, ch := range raw {
		if ch == ' ' || ch == '\t' || ch == '=' || ch == '>' || ch == '<' {
			tok = raw[:i]
			break
		}
	}
	return tok, true
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
