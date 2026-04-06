package codegen

import "github.com/monkfromearth/monk-lang/syntax"

// freeVars returns the set of variable names referenced inside a FuncExpr
// that are NOT declared as parameters or local variables within that function.
// These are the variables that need to be captured for closure support.
func freeVars(fn *syntax.FuncExpr) []string {
	// Collect param names as the initial local scope.
	locals := make(map[string]bool)
	for _, p := range fn.Params {
		locals[p.Name] = true
	}
	refs := make(map[string]bool)
	collectRefs(fn.Body.Stmts, locals, refs)

	// Convert to sorted slice for deterministic output.
	result := make([]string, 0, len(refs))
	for name := range refs {
		result = append(result, name)
	}
	// Sort for deterministic C output.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j] < result[i] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// collectRefs walks statements, tracking local declarations and collecting
// references to variables not in locals.
func collectRefs(stmts []syntax.Stmt, locals map[string]bool, refs map[string]bool) {
	for _, stmt := range stmts {
		collectRefsStmt(stmt, locals, refs)
	}
}

func collectRefsStmt(stmt syntax.Stmt, locals map[string]bool, refs map[string]bool) {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		// The value expression may reference outer vars.
		collectRefsExpr(s.Value, locals, refs)
		// Then declare the variable locally.
		locals[s.Name] = true
	case *syntax.AssignStmt:
		collectRefsExpr(s.Target, locals, refs)
		collectRefsExpr(s.Value, locals, refs)
	case *syntax.ExprStmt:
		collectRefsExpr(s.Expr, locals, refs)
	case *syntax.ReturnStmt:
		if s.Value != nil {
			collectRefsExpr(s.Value, locals, refs)
		}
	case *syntax.IfStmt:
		collectRefsExpr(s.Condition, locals, refs)
		collectRefs(s.Then.Stmts, locals, refs)
		if s.Else != nil {
			collectRefsStmt(s.Else, locals, refs)
		}
	case *syntax.WhileStmt:
		collectRefsExpr(s.Condition, locals, refs)
		collectRefs(s.Body.Stmts, locals, refs)
	case *syntax.ForStmt:
		collectRefsExpr(s.Iterable, locals, refs)
		// Loop variable is local to the body.
		saved := locals[s.VarName]
		locals[s.VarName] = true
		collectRefs(s.Body.Stmts, locals, refs)
		if !saved {
			delete(locals, s.VarName)
		}
	case *syntax.GuardStmt:
		collectRefsExpr(s.Expr, locals, refs)
		locals[s.VarName] = true
		collectRefs(s.Against.Stmts, locals, refs)
	case *syntax.BlockStmt:
		collectRefs(s.Stmts, locals, refs)
	}
}

func collectRefsExpr(expr syntax.Expr, locals map[string]bool, refs map[string]bool) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *syntax.IdentExpr:
		if !locals[e.Name] {
			refs[e.Name] = true
		}
	case *syntax.UnaryExpr:
		collectRefsExpr(e.Operand, locals, refs)
	case *syntax.BinaryExpr:
		collectRefsExpr(e.Left, locals, refs)
		collectRefsExpr(e.Right, locals, refs)
	case *syntax.CallExpr:
		collectRefsExpr(e.Callee, locals, refs)
		for _, arg := range e.Args {
			collectRefsExpr(arg, locals, refs)
		}
	case *syntax.IndexExpr:
		collectRefsExpr(e.Object, locals, refs)
		collectRefsExpr(e.Index, locals, refs)
	case *syntax.PropertyExpr:
		collectRefsExpr(e.Object, locals, refs)
	case *syntax.ArrayExpr:
		for _, elem := range e.Elements {
			collectRefsExpr(elem, locals, refs)
		}
	case *syntax.RecordExpr:
		for _, f := range e.Fields {
			collectRefsExpr(f.Value, locals, refs)
		}
	case *syntax.FuncExpr:
		// Nested function — collect its free vars but don't add its params
		// to the current scope. Its own free vars become refs for us.
		inner := make(map[string]bool)
		for _, p := range e.Params {
			inner[p.Name] = true
		}
		collectRefs(e.Body.Stmts, inner, refs)
	case *syntax.ThrowExpr:
		collectRefsExpr(e.Value, locals, refs)
	}
}
