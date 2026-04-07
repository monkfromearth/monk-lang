package codegen

import "github.com/monkfromearth/monk-lang/syntax"

type stackFuncInfo struct {
	cName        string
	selfName     string
	capArrayName string // C name of the MonkValue captures[] stack array (empty if no captures)
	capCount     int    // number of captures (for cleanup loop)
}

type stackFuncAnalysis struct {
	refs    []*syntax.CallExpr
	escapes bool
}

// analyzeStackFuncDecls finds function literals whose variable is only used as
// a direct callee inside its lexical lifetime. Those closures can use a stack
// MonkFunction frame instead of monk_make_function heap allocation.
//
// Pass: `let f = (x int) int { return x+n }; y = f(1)` stack-allocates f.
// Fail: `callbacks = append(callbacks, f)` stays heap-allocated because f escapes.
func analyzeStackFuncDecls(prog *syntax.Program) (map[*syntax.VarDeclStmt]bool, map[*syntax.CallExpr]*syntax.VarDeclStmt) {
	decls := make(map[*syntax.VarDeclStmt]bool)
	calls := make(map[*syntax.CallExpr]*syntax.VarDeclStmt)
	if prog == nil {
		return decls, calls
	}
	scanStackFuncBlock(prog.Stmts, decls, calls)
	return decls, calls
}

func scanStackFuncBlock(stmts []syntax.Stmt, decls map[*syntax.VarDeclStmt]bool, calls map[*syntax.CallExpr]*syntax.VarDeclStmt) {
	for i, stmt := range stmts {
		if decl, ok := stmt.(*syntax.VarDeclStmt); ok {
			if fn, isFn := decl.Value.(*syntax.FuncExpr); isFn && !funcBodyMentionsName(fn, decl.Name) {
				usage := &stackFuncAnalysis{}
				stackFuncScanStmts(stmts[i+1:], decl.Name, usage)
				if !usage.escapes && len(usage.refs) > 0 {
					decls[decl] = true
					for _, call := range usage.refs {
						calls[call] = decl
					}
				}
			}
		}
		scanStackFuncChildren(stmt, decls, calls)
	}
}

func funcBodyMentionsName(fn *syntax.FuncExpr, name string) bool {
	if fn == nil || fn.Body == nil {
		return false
	}
	for _, stmt := range fn.Body.Stmts {
		if stmtMentionsName(stmt, name) {
			return true
		}
	}
	return false
}

func stmtMentionsName(stmt syntax.Stmt, name string) bool {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		return exprMentionsName(s.Value, name)
	case *syntax.AssignStmt:
		return exprMentionsName(s.Target, name) || exprMentionsName(s.Value, name)
	case *syntax.ExprStmt:
		return exprMentionsName(s.Expr, name)
	case *syntax.IfStmt:
		return exprMentionsName(s.Condition, name) ||
			blockMentionsName(s.Then, name) ||
			(s.Else != nil && stmtMentionsName(s.Else, name))
	case *syntax.WhileStmt:
		return exprMentionsName(s.Condition, name) || blockMentionsName(s.Body, name)
	case *syntax.ForStmt:
		return exprMentionsName(s.Iterable, name) || (s.VarName != name && blockMentionsName(s.Body, name))
	case *syntax.ReturnStmt:
		return exprMentionsName(s.Value, name)
	case *syntax.GuardStmt:
		return exprMentionsName(s.Expr, name) ||
			(s.VarName != name && s.ErrorName != name && blockMentionsName(s.Against, name))
	case *syntax.ExportStmt:
		return stmtMentionsName(s.Stmt, name)
	}
	return false
}

func blockMentionsName(block *syntax.BlockStmt, name string) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.Stmts {
		if stmtMentionsName(stmt, name) {
			return true
		}
	}
	return false
}

func exprMentionsName(expr syntax.Expr, name string) bool {
	switch e := expr.(type) {
	case nil:
		return false
	case *syntax.IdentExpr:
		return e.Name == name
	case *syntax.UnaryExpr:
		return exprMentionsName(e.Operand, name)
	case *syntax.BinaryExpr:
		return exprMentionsName(e.Left, name) || exprMentionsName(e.Right, name)
	case *syntax.CallExpr:
		if exprMentionsName(e.Callee, name) {
			return true
		}
		for _, arg := range e.Args {
			if exprMentionsName(arg, name) {
				return true
			}
		}
	case *syntax.IndexExpr:
		return exprMentionsName(e.Object, name) || exprMentionsName(e.Index, name)
	case *syntax.PropertyExpr:
		return exprMentionsName(e.Object, name)
	case *syntax.ArrayExpr:
		for _, elem := range e.Elements {
			if exprMentionsName(elem, name) {
				return true
			}
		}
	case *syntax.RecordExpr:
		for _, field := range e.Fields {
			if exprMentionsName(field.Value, name) {
				return true
			}
		}
	case *syntax.FuncExpr:
		return blockMentionsName(e.Body, name)
	case *syntax.ThrowExpr:
		return exprMentionsName(e.Value, name)
	}
	return false
}

func scanStackFuncChildren(stmt syntax.Stmt, decls map[*syntax.VarDeclStmt]bool, calls map[*syntax.CallExpr]*syntax.VarDeclStmt) {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		if fn, ok := s.Value.(*syntax.FuncExpr); ok && fn.Body != nil {
			scanStackFuncBlock(fn.Body.Stmts, decls, calls)
		}
	case *syntax.ExprStmt:
		scanStackFuncExprChildren(s.Expr, decls, calls)
	case *syntax.AssignStmt:
		scanStackFuncExprChildren(s.Target, decls, calls)
		scanStackFuncExprChildren(s.Value, decls, calls)
	case *syntax.IfStmt:
		scanStackFuncExprChildren(s.Condition, decls, calls)
		scanStackFuncBlock(s.Then.Stmts, decls, calls)
		if s.Else != nil {
			scanStackFuncChildren(s.Else, decls, calls)
		}
	case *syntax.WhileStmt:
		scanStackFuncExprChildren(s.Condition, decls, calls)
		scanStackFuncBlock(s.Body.Stmts, decls, calls)
	case *syntax.ForStmt:
		scanStackFuncExprChildren(s.Iterable, decls, calls)
		scanStackFuncBlock(s.Body.Stmts, decls, calls)
	case *syntax.ReturnStmt:
		scanStackFuncExprChildren(s.Value, decls, calls)
	case *syntax.GuardStmt:
		scanStackFuncExprChildren(s.Expr, decls, calls)
		scanStackFuncBlock(s.Against.Stmts, decls, calls)
	case *syntax.ExportStmt:
		scanStackFuncChildren(s.Stmt, decls, calls)
	}
}

func scanStackFuncExprChildren(expr syntax.Expr, decls map[*syntax.VarDeclStmt]bool, calls map[*syntax.CallExpr]*syntax.VarDeclStmt) {
	switch e := expr.(type) {
	case *syntax.UnaryExpr:
		scanStackFuncExprChildren(e.Operand, decls, calls)
	case *syntax.BinaryExpr:
		scanStackFuncExprChildren(e.Left, decls, calls)
		scanStackFuncExprChildren(e.Right, decls, calls)
	case *syntax.CallExpr:
		scanStackFuncExprChildren(e.Callee, decls, calls)
		for _, arg := range e.Args {
			scanStackFuncExprChildren(arg, decls, calls)
		}
	case *syntax.IndexExpr:
		scanStackFuncExprChildren(e.Object, decls, calls)
		scanStackFuncExprChildren(e.Index, decls, calls)
	case *syntax.PropertyExpr:
		scanStackFuncExprChildren(e.Object, decls, calls)
	case *syntax.ArrayExpr:
		for _, elem := range e.Elements {
			scanStackFuncExprChildren(elem, decls, calls)
		}
	case *syntax.RecordExpr:
		for _, field := range e.Fields {
			scanStackFuncExprChildren(field.Value, decls, calls)
		}
	case *syntax.FuncExpr:
		if e.Body != nil {
			scanStackFuncBlock(e.Body.Stmts, decls, calls)
		}
	case *syntax.ThrowExpr:
		scanStackFuncExprChildren(e.Value, decls, calls)
	}
}

func stackFuncScanStmts(stmts []syntax.Stmt, name string, usage *stackFuncAnalysis) {
	for _, stmt := range stmts {
		if usage.escapes {
			return
		}
		if decl, ok := stmt.(*syntax.VarDeclStmt); ok && decl.Name == name {
			stackFuncScanExpr(decl.Value, name, usage, false, false)
			return
		}
		stackFuncScanStmt(stmt, name, usage)
	}
}

func stackFuncScanBlock(block *syntax.BlockStmt, name string, usage *stackFuncAnalysis) {
	if block == nil {
		return
	}
	stackFuncScanStmts(block.Stmts, name, usage)
}

func stackFuncScanStmt(stmt syntax.Stmt, name string, usage *stackFuncAnalysis) {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		stackFuncScanExpr(s.Value, name, usage, false, false)
	case *syntax.AssignStmt:
		stackFuncScanExpr(s.Target, name, usage, false, false)
		stackFuncScanExpr(s.Value, name, usage, false, false)
	case *syntax.ExprStmt:
		stackFuncScanExpr(s.Expr, name, usage, false, false)
	case *syntax.IfStmt:
		stackFuncScanExpr(s.Condition, name, usage, false, false)
		stackFuncScanBlock(s.Then, name, usage)
		if s.Else != nil {
			stackFuncScanStmt(s.Else, name, usage)
		}
	case *syntax.WhileStmt:
		stackFuncScanExpr(s.Condition, name, usage, false, false)
		stackFuncScanBlock(s.Body, name, usage)
	case *syntax.ForStmt:
		stackFuncScanExpr(s.Iterable, name, usage, false, false)
		if s.VarName != name {
			stackFuncScanBlock(s.Body, name, usage)
		}
	case *syntax.ReturnStmt:
		stackFuncScanExpr(s.Value, name, usage, false, false)
	case *syntax.GuardStmt:
		stackFuncScanExpr(s.Expr, name, usage, false, false)
		if s.VarName != name && s.ErrorName != name {
			stackFuncScanBlock(s.Against, name, usage)
		}
	case *syntax.ExportStmt:
		stackFuncScanStmt(s.Stmt, name, usage)
	}
}

func stackFuncScanExpr(expr syntax.Expr, name string, usage *stackFuncAnalysis, directCallee bool, nestedFunc bool) {
	if expr == nil || usage.escapes {
		return
	}
	switch e := expr.(type) {
	case *syntax.IdentExpr:
		if e.Name == name {
			if directCallee && !nestedFunc {
				return
			}
			usage.escapes = true
		}
	case *syntax.UnaryExpr:
		stackFuncScanExpr(e.Operand, name, usage, false, nestedFunc)
	case *syntax.BinaryExpr:
		stackFuncScanExpr(e.Left, name, usage, false, nestedFunc)
		stackFuncScanExpr(e.Right, name, usage, false, nestedFunc)
	case *syntax.CallExpr:
		if ident, ok := e.Callee.(*syntax.IdentExpr); ok && ident.Name == name && !nestedFunc {
			usage.refs = append(usage.refs, e)
		} else {
			stackFuncScanExpr(e.Callee, name, usage, false, nestedFunc)
		}
		for _, arg := range e.Args {
			stackFuncScanExpr(arg, name, usage, false, nestedFunc)
		}
	case *syntax.IndexExpr:
		stackFuncScanExpr(e.Object, name, usage, false, nestedFunc)
		stackFuncScanExpr(e.Index, name, usage, false, nestedFunc)
	case *syntax.PropertyExpr:
		stackFuncScanExpr(e.Object, name, usage, false, nestedFunc)
	case *syntax.ArrayExpr:
		for _, elem := range e.Elements {
			stackFuncScanExpr(elem, name, usage, false, nestedFunc)
		}
	case *syntax.RecordExpr:
		for _, field := range e.Fields {
			stackFuncScanExpr(field.Value, name, usage, false, nestedFunc)
		}
	case *syntax.FuncExpr:
		if e.Body != nil {
			for _, stmt := range e.Body.Stmts {
				stackFuncScanStmtNested(stmt, name, usage)
			}
		}
	case *syntax.ThrowExpr:
		stackFuncScanExpr(e.Value, name, usage, false, nestedFunc)
	}
}

func stackFuncScanStmtNested(stmt syntax.Stmt, name string, usage *stackFuncAnalysis) {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		stackFuncScanExpr(s.Value, name, usage, false, true)
	case *syntax.AssignStmt:
		stackFuncScanExpr(s.Target, name, usage, false, true)
		stackFuncScanExpr(s.Value, name, usage, false, true)
	case *syntax.ExprStmt:
		stackFuncScanExpr(s.Expr, name, usage, false, true)
	case *syntax.IfStmt:
		stackFuncScanExpr(s.Condition, name, usage, false, true)
		for _, stmt := range s.Then.Stmts {
			stackFuncScanStmtNested(stmt, name, usage)
		}
		if s.Else != nil {
			stackFuncScanStmtNested(s.Else, name, usage)
		}
	case *syntax.WhileStmt:
		stackFuncScanExpr(s.Condition, name, usage, false, true)
		for _, stmt := range s.Body.Stmts {
			stackFuncScanStmtNested(stmt, name, usage)
		}
	case *syntax.ForStmt:
		stackFuncScanExpr(s.Iterable, name, usage, false, true)
		for _, stmt := range s.Body.Stmts {
			stackFuncScanStmtNested(stmt, name, usage)
		}
	case *syntax.ReturnStmt:
		stackFuncScanExpr(s.Value, name, usage, false, true)
	case *syntax.GuardStmt:
		stackFuncScanExpr(s.Expr, name, usage, false, true)
		for _, stmt := range s.Against.Stmts {
			stackFuncScanStmtNested(stmt, name, usage)
		}
	case *syntax.ExportStmt:
		stackFuncScanStmtNested(s.Stmt, name, usage)
	}
}
