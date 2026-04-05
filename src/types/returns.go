// Package types — all-paths-return analysis.
//
// A function declared to return a non-none, non-any type must guarantee a
// return value on every execution path. Equivalent to Go's "missing return
// at end of function".
//
// A statement list returns on every path if one of the following holds:
//   - the last statement is `return <expr>` or `throw`, OR
//   - an `if` (with `else`) where both branches return, OR
//   - a guard whose against-block returns AND whose main expr is a throw
//     (too subtle; we don't model it — user can just return in both)
//
// We're conservative — if we can't prove a path returns, we report it.
package types

import (
	"github.com/monkfromearth/monk-lang/syntax"
)

// stmtsAlwaysReturn reports whether every path through this statement list
// ends in a return or throw.
func stmtsAlwaysReturn(stmts []syntax.Stmt) bool {
	for _, s := range stmts {
		if stmtAlwaysReturns(s) {
			return true
		}
	}
	return false
}

// stmtAlwaysReturns reports whether every path through this single statement
// terminates with return/throw. Only `return`, `throw`, an if/else where both
// sides return, and nested blocks are introspected. Loops are not — `while`
// may not iterate, `for` may iterate zero times.
func stmtAlwaysReturns(s syntax.Stmt) bool {
	switch st := s.(type) {
	case *syntax.ReturnStmt:
		return true
	case *syntax.ExprStmt:
		// `throw <expr>` as a bare statement terminates.
		if _, ok := st.Expr.(*syntax.ThrowExpr); ok {
			return true
		}
	case *syntax.BlockStmt:
		return stmtsAlwaysReturn(st.Stmts)
	case *syntax.IfStmt:
		// else-less if: the else path falls through, so NOT always.
		if st.Else == nil {
			return false
		}
		return stmtsAlwaysReturn(st.Then.Stmts) && stmtAlwaysReturns(st.Else)
	case *syntax.GuardStmt:
		// Both the success path (after guard) AND the against path must
		// terminate. The success path falls through to whatever comes after
		// the guard in its enclosing list, so a guard alone never proves
		// "always returns" — unless its against-block and its SUCCESS path
		// both do. We can't see the success continuation from here, so say no.
		// Users who need this can return explicitly.
		return false
	}
	return false
}
