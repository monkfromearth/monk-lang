// Package codegen — static bounds analysis for bounds-check elision.
//
// For typed arrays (int[]/float[]/bool[]), every element access emits a
// bounds check by default. This file implements a lightweight static analysis
// that proves certain accesses are always in-bounds, allowing the check to
// be elided.
//
// Two data sources are combined:
//  1. constVals — compile-time constant integer variables (e.g. `let N = 400`).
//     Populated by emitVarDecl when the RHS is a constant expression.
//  2. arrayLens — statically known length of typed array variables
//     (e.g. `let arr int[] = range(N)` → arrayLens["arr"] = N).
//     Populated by emitVarDecl for range()-initialized typed arrays.
//  3. varBounds — tight [lo, hi] inclusive range for loop-counter variables,
//     set when emitWhile detects a simple `i < BOUND` condition.
//     Cleared when the while loop exits (restoreVarBounds).
//
// The elision check (isBoundedSafe) evaluates the index expression's range
// via evalRange and confirms: min >= 0 && max < arr.length.
// If both hold, the bounds check is skipped and direct data[] access is emitted.
package codegen

import (
	"github.com/monkfromearth/monk-lang/syntax"
)

// initBounds initializes the bound-tracking maps. Called from GenerateWithTypes.
func (g *generator) initBounds() {
	g.constVals = make(map[string]int64)
	g.arrayLens = make(map[string]int64)
	g.varBounds = make(map[string][2]int64)
}

// tryConst evaluates expr as a compile-time integer constant.
// Returns (value, true) for integer literals and tracked constant variables.
func (g *generator) tryConst(expr syntax.Expr) (int64, bool) {
	switch e := expr.(type) {
	case *syntax.NumberExpr:
		if e.IsInt {
			// Parse the integer literal (strip underscores first).
			val := int64(0)
			s := e.Value
			for i := 0; i < len(s); i++ {
				if s[i] == '_' {
					continue
				}
				// Handle hex/octal/binary prefixes by falling back to zero.
				if s[0] == '0' && len(s) > 1 {
					return 0, false
				}
				if s[i] < '0' || s[i] > '9' {
					return 0, false
				}
				val = val*10 + int64(s[i]-'0')
			}
			return val, true
		}
	case *syntax.IdentExpr:
		if v, ok := g.constVals[e.Name]; ok {
			return v, true
		}
	case *syntax.BinaryExpr:
		lv, lok := g.tryConst(e.Left)
		rv, rok := g.tryConst(e.Right)
		if lok && rok {
			switch e.Op {
			case syntax.Plus:
				return lv + rv, true
			case syntax.Minus:
				return lv - rv, true
			case syntax.Star:
				return lv * rv, true
			case syntax.Slash:
				if rv != 0 {
					return lv / rv, true
				}
			}
		}
	}
	return 0, false
}

// recordConst stores a constant value for a variable if expr is a constant expression.
func (g *generator) recordConst(name string, expr syntax.Expr) {
	if v, ok := g.tryConst(expr); ok {
		g.constVals[name] = v
	}
}

// recordArrayLen stores the known length of a typed array variable, if the
// initializer is range(CONST_EXPR). Only called for typed array declarations.
func (g *generator) recordArrayLen(name string, expr syntax.Expr) {
	// Pattern: [e1, e2, ...] → length = number of elements
	if arr, ok := expr.(*syntax.ArrayExpr); ok {
		g.arrayLens[name] = int64(len(arr.Elements))
		return
	}
	// Pattern: range(CONST_EXPR) or fill(CONST_EXPR, value) → length = first arg
	call, ok := expr.(*syntax.CallExpr)
	if !ok {
		return
	}
	callee, ok := call.Callee.(*syntax.IdentExpr)
	if !ok {
		return
	}
	switch callee.Name {
	case "range":
		switch len(call.Args) {
		case 1:
			// range(N) → elements 0..N-1, length = N
			if n, ok := g.tryConst(call.Args[0]); ok {
				g.arrayLens[name] = n
			}
		case 2:
			// range(start, end) → elements start..end-1, length = end - start
			if lo, ok1 := g.tryConst(call.Args[0]); ok1 {
				if hi, ok2 := g.tryConst(call.Args[1]); ok2 {
					g.arrayLens[name] = hi - lo
				}
			}
		}
	case "fill":
		// fill(N, value) → length = N. Same as range(N) for bounds analysis.
		// Pass: `fill(N + 1, true)` with const N → length = N + 1.
		if len(call.Args) >= 1 {
			if n, ok := g.tryConst(call.Args[0]); ok {
				g.arrayLens[name] = n
			}
		}
	}
}

// evalRange computes the inclusive [min, max] integer range of expr based on
// tracked constants and variable bounds. Returns (0, 0, false) if unknown.
// All values are assumed to be non-negative for unsigned-safe reasoning.
func (g *generator) evalRange(expr syntax.Expr) (lo, hi int64, ok bool) {
	switch e := expr.(type) {
	case *syntax.NumberExpr:
		if v, ok := g.tryConst(e); ok {
			return v, v, true
		}
	case *syntax.IdentExpr:
		// Loop-bounded variable takes precedence over constVals.
		// Inside `while i < N`, varBounds["i"] = [0, N-1] is the live range,
		// but constVals["i"] = 0 (from `let i int = 0`). Checking constVals
		// first would return [0, 0], causing unsound elision when N > arrLen.
		// Pass: `while i < 3 { arr[i] }` with arr=range(3) → [0, 2] < 3.
		// Fail (before fix): `while i < 10 { arr[i] }` with arr=range(3) → [0, 0] < 3 → elided → OOB.
		if b, boundOK := g.varBounds[e.Name]; boundOK {
			return b[0], b[1], true
		}
		// Constant variable → tight range (only used outside loops).
		if v, constOK := g.constVals[e.Name]; constOK {
			return v, v, true
		}
	case *syntax.BinaryExpr:
		llo, lhi, lok := g.evalRange(e.Left)
		rlo, rhi, rok := g.evalRange(e.Right)
		if lok && rok && llo >= 0 && rlo >= 0 {
			// Only reason about non-negative values to keep range math safe.
			switch e.Op {
			case syntax.Plus:
				return llo + rlo, lhi + rhi, true
			case syntax.Star:
				return llo * rlo, lhi * rhi, true
			}
		}
	}
	return 0, 0, false
}

// isBoundedSafe reports whether arr[idxExpr] is provably in-bounds.
// arrName is the mangled Monk variable name (e.g. "mk_arr" → looks up "arr").
// If true, the caller can skip the runtime bounds check.
func (g *generator) isBoundedSafe(arrIdent *syntax.IdentExpr, idxExpr syntax.Expr) bool {
	arrLen, ok := g.arrayLens[arrIdent.Name]
	if !ok || arrLen <= 0 {
		return false
	}
	lo, hi, rangeOK := g.evalRange(idxExpr)
	if !rangeOK {
		return false
	}
	return lo >= 0 && hi < arrLen
}

// whileBoundsEntry detects a simple bounding condition on a while loop and
// returns the loop variable name and its exclusive upper bound.
//
// Recognized patterns:
//   - `i < CONST`   → i ∈ [0, CONST-1] inclusive, exclusive bound = CONST
//   - `i <= CONST`  → i ∈ [0, CONST] inclusive, exclusive bound = CONST+1
//   - `i < varname` where varname is a compile-time constant → same as above
//   - `i <= varname` → same as above
//
// Returns ("", 0, 0, false) if the condition doesn't match any recognized pattern.
func (g *generator) whileBoundsEntry(cond syntax.Expr) (varName string, loInc, hiExc int64, found bool) {
	bin, ok := cond.(*syntax.BinaryExpr)
	if !ok {
		return "", 0, 0, false
	}
	ident, ok := bin.Left.(*syntax.IdentExpr)
	if !ok {
		return "", 0, 0, false
	}
	boundVal, ok := g.tryConst(bin.Right)
	if !ok {
		return "", 0, 0, false
	}
	// Use the variable's known initial value as the lower bound instead of
	// hardcoding 0. If `let i int = 5` then `while i < N` has range [5, N-1],
	// not [0, N-1]. If the initial value isn't a compile-time constant (e.g.
	// `let i int = arr[0]`), we can't prove the range — bail entirely.
	// Pass: `let i int = 0; while i < N` → lo=0.
	// Fail: `let i int = some_func(); while i < N` → bail, no elision.
	lo, hasInit := g.constVals[ident.Name]
	if !hasInit {
		return "", 0, 0, false
	}
	switch bin.Op {
	case syntax.Less:
		// i < N → i ∈ [lo, N-1]
		return ident.Name, lo, boundVal - 1, true
	case syntax.LessEqual:
		// i <= N → i ∈ [lo, N]
		return ident.Name, lo, boundVal, true
	}
	return "", 0, 0, false
}

// setVarBound records a tight [lo, hi] inclusive range for a loop variable.
// Called before emitting a while-loop body. The previous bound (if any) is
// returned so the caller can restore it with restoreVarBound.
func (g *generator) setVarBound(name string, lo, hi int64) (prev [2]int64, hadPrev bool) {
	prev, hadPrev = g.varBounds[name]
	g.varBounds[name] = [2]int64{lo, hi}
	return prev, hadPrev
}

// restoreVarBound removes or restores a loop variable's bound after the loop body.
func (g *generator) restoreVarBound(name string, prev [2]int64, hadPrev bool) {
	if hadPrev {
		g.varBounds[name] = prev
	} else {
		delete(g.varBounds, name)
	}
}
