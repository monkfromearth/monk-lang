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

// copyLocals returns a shallow copy of the locals map so that declarations in
// one branch (e.g. then-block) don't bleed into sibling branches (else-block).
func copyLocals(m map[string]bool) map[string]bool {
	c := make(map[string]bool, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

// collectRefs walks statements, tracking local declarations and collecting
// references to variables not in locals.
func collectRefs(stmts []syntax.Stmt, locals map[string]bool, refs map[string]bool) {
	for _, stmt := range stmts {
		collectRefsStmt(stmt, locals, refs)
	}
}

// collectRefsStmt walks a single statement, updating locals with any new
// declarations and refs with any out-of-scope variable references.
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
		// Copy locals so declarations in the then-branch don't bleed into the
		// else-branch (and don't leak into the outer scope after the if).
		thenLocals := copyLocals(locals)
		collectRefs(s.Then.Stmts, thenLocals, refs)
		if s.Else != nil {
			elseLocals := copyLocals(locals)
			collectRefsStmt(s.Else, elseLocals, refs)
		}
	case *syntax.WhileStmt:
		collectRefsExpr(s.Condition, locals, refs)
		// Copy locals so declarations inside the loop body don't leak into
		// the outer scope. Without this, `let x = 20` inside a while body
		// adds x to the shared locals map — a closure referencing the OUTER
		// x after the loop would miss the capture (thinks x is local).
		// Pass: outer x captured after while that declares inner x.
		// Fail (before fix): inner x leaks → outer x not in refs → undeclared in C.
		whileLocals := copyLocals(locals)
		collectRefs(s.Body.Stmts, whileLocals, refs)
	case *syntax.ForStmt:
		collectRefsExpr(s.Iterable, locals, refs)
		// Same scoping fix as WhileStmt: copy locals so body-internal
		// declarations don't leak into the enclosing scope's locals map.
		// The loop variable is added to the copy (local to the body only).
		forLocals := copyLocals(locals)
		forLocals[s.VarName] = true
		collectRefs(s.Body.Stmts, forLocals, refs)
	case *syntax.GuardStmt:
		collectRefsExpr(s.Expr, locals, refs)
		locals[s.VarName] = true
		collectRefs(s.Against.Stmts, locals, refs)
	case *syntax.BlockStmt:
		collectRefs(s.Stmts, locals, refs)
	}
}

// collectRefsExpr walks an expression, recording any identifier that is not in
// locals as a free variable reference. For nested FuncExprs, params are added
// to a copy of locals so outer captures are correctly identified.
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
		// Nested function: seed inner with the current scope so that outer
		// locals aren't mistakenly reported as free vars. Outer-scope vars NOT
		// in locals (grandparent captures) remain absent and correctly flow into
		// refs. The nested function's own params override any outer binding.
		inner := copyLocals(locals)
		for _, p := range e.Params {
			inner[p.Name] = true
		}
		collectRefs(e.Body.Stmts, inner, refs)
	case *syntax.ThrowExpr:
		collectRefsExpr(e.Value, locals, refs)
	}
}
