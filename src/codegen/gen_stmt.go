package codegen

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Statement emission. Each emit* method writes directly into g.body.

func (g *generator) emitStmt(stmt syntax.Stmt) {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		g.emitVarDecl(s)
	case *syntax.AssignStmt:
		g.emitAssign(s)
	case *syntax.ExprStmt:
		g.emitLine("    %s;\n", g.emitExpr(s.Expr))
	case *syntax.IfStmt:
		g.emitIf(s)
	case *syntax.WhileStmt:
		g.emitWhile(s)
	case *syntax.ForStmt:
		g.emitFor(s)
	case *syntax.ReturnStmt:
		g.emitReturn(s)
	case *syntax.BreakStmt:
		g.emitLine("    break;\n")
	case *syntax.ContinueStmt:
		g.emitLine("    continue;\n")
	case *syntax.GuardStmt:
		g.emitGuard(s)
	case *syntax.BlockStmt:
		for _, inner := range s.Stmts {
			g.emitStmt(inner)
		}
	default:
		g.emitLine("    /* TODO: unhandled statement %T */\n", stmt)
	}
}

func (g *generator) emitVarDecl(s *syntax.VarDeclStmt) {
	// Function declarations: hoist the C function, track the name mapping
	if fnExpr, isFn := s.Value.(*syntax.FuncExpr); isFn {
		g.funcCount++
		cFuncName := fmt.Sprintf("_monk_func_%d", g.funcCount)
		g.funcNames[s.Name] = cFuncName
		g.hoistFunction(cFuncName, fnExpr)
		return
	}

	name := mangleName(s.Name)

	// Decide this variable's storage based on the type checker's verdict.
	// Absent type info (legacy Generate path), stay boxed.
	var store storageKind
	if g.info != nil {
		if t, ok := g.info.Decls[s]; ok {
			store = storageFor(t)
		}
	}
	g.storage[name] = store

	if store == storeBoxed {
		// Classic path — one MonkValue per variable, deep-copied at init.
		value := g.emitExpr(s.Value)
		g.emitLine("    MonkValue %s = monk_deep_copy(%s);\n", name, value)
		return
	}

	// Unboxed path — emit a raw C scalar. The initializer is an expression
	// whose type we know; coerce to match the variable's storage.
	rhsCode, rhsStore := g.emitExprTyped(s.Value)
	init := coerce(rhsCode, rhsStore, store)
	g.emitLine("    %s %s = %s;\n", cTypeName(store), name, init)
}

func (g *generator) emitAssign(s *syntax.AssignStmt) {
	// Unboxed-variable fast path: if the target is a plain ident with raw
	// storage, emit raw C assignment (no deep_copy/free cycle needed — the
	// value is a plain scalar).
	if target, ok := s.Target.(*syntax.IdentExpr); ok {
		name := mangleName(target.Name)
		if store := g.varStorage(name); store != storeBoxed {
			rhsCode, rhsStore := g.emitExprTyped(s.Value)
			rhs := coerce(rhsCode, rhsStore, store)
			// Stash RHS in a temp for ops that need to evaluate it twice
			// (div/mod guards), so a side-effecting RHS like `a /= f()`
			// doesn't call f() both for the zero-check and the divide.
			switch s.Op {
			case syntax.Equal:
				g.emitLine("    %s = %s;\n", name, rhs)
			case syntax.PlusEqual:
				g.emitLine("    %s += %s;\n", name, rhs)
			case syntax.MinusEqual:
				g.emitLine("    %s -= %s;\n", name, rhs)
			case syntax.StarEqual:
				g.emitLine("    %s *= %s;\n", name, rhs)
			case syntax.SlashEqual:
				if store == storeInt {
					tmp := g.newTemp()
					g.emitLine("    { int64_t %s = %s; if (%s==0) monk_panic(\"division by zero\"); %s /= %s; }\n",
						tmp, rhs, tmp, name, tmp)
				} else {
					g.emitLine("    %s /= %s;\n", name, rhs)
				}
			case syntax.PercentEqual:
				if store == storeInt {
					tmp := g.newTemp()
					g.emitLine("    { int64_t %s = %s; if (%s==0) monk_panic(\"modulo by zero\"); %s %%= %s; }\n",
						tmp, rhs, tmp, name, tmp)
				} else {
					// float %=: C's `%` doesn't work on doubles, but spec
					// says `%` applies to numeric. Use fmod() from math.h
					// (already linked via -lm). Zero-check for consistency.
					tmp := g.newTemp()
					g.emitLine("    { double %s = %s; if (%s==0.0) monk_panic(\"modulo by zero\"); %s = fmod(%s, %s); }\n",
						tmp, rhs, tmp, name, name, tmp)
				}
			}
			return
		}
	}

	value := g.emitExpr(s.Value)

	switch target := s.Target.(type) {
	case *syntax.IdentExpr:
		name := mangleName(target.Name)
		tmp := g.newTemp()
		switch s.Op {
		case syntax.Equal:
			// Compute new value BEFORE freeing old (avoids use-after-free
			// when the new value expression references the variable)
			g.emitLine("    { MonkValue %s = monk_deep_copy(%s);\n", tmp, value)
			g.emitLine("      monk_free(%s);\n", name)
			g.emitLine("      %s = %s; }\n", name, tmp)
		case syntax.PlusEqual:
			// += mirrors Plus: dispatch on MONK_STRING for the concat overload
			// so `s += "world"` on a string routes through monk_string_concat
			// instead of monk_add (which would runtime-error).
			g.emitLine("    { MonkValue %s = (%s.kind==MONK_STRING ? monk_string_concat(%s,%s) : monk_add(%s,%s));\n",
				tmp, name, name, value, name, value)
			g.emitLine("      monk_free(%s);\n", name)
			g.emitLine("      %s = %s; }\n", name, tmp)
		default:
			op := compoundToArith(s.Op)
			g.emitLine("    { MonkValue %s = %s(%s, %s);\n", tmp, op, name, value)
			g.emitLine("      monk_free(%s);\n", name)
			g.emitLine("      %s = %s; }\n", name, tmp)
		}
	case *syntax.IndexExpr:
		// Common case — target.Object is a plain identifier, so we can take
		// its address directly and mutate in place. Nested targets like
		// `matrix[i][j]` produce an rvalue from emitExpr (`monk_array_get(…)`),
		// which cannot be the operand of &. Route those through a temp so
		// the generated C compiles; mutating the intermediate copy is a
		// no-op per Monk's value semantics, which is the intended behavior —
		// nested index assignment does NOT propagate to the outer container.
		// Users who need deep mutation must rebind: `let row = m[i]; row[j] = x; m[i] = row`.
		idx := g.emitExpr(target.Index)
		if _, ok := target.Object.(*syntax.IdentExpr); ok {
			obj := g.emitExpr(target.Object)
			g.emitLine("    monk_array_set(&%s, %s, %s);\n", obj, idx, value)
		} else {
			obj := g.emitExpr(target.Object)
			tmp := g.newTemp()
			g.emitLine("    { MonkValue %s = %s; monk_array_set(&%s, %s, %s); }\n",
				tmp, obj, tmp, idx, value)
		}
	case *syntax.PropertyExpr:
		// Same rationale as IndexExpr above.
		if _, ok := target.Object.(*syntax.IdentExpr); ok {
			obj := g.emitExpr(target.Object)
			g.emitLine("    monk_record_set(&%s, \"%s\", %s);\n", obj, target.Property, value)
		} else {
			obj := g.emitExpr(target.Object)
			tmp := g.newTemp()
			g.emitLine("    { MonkValue %s = %s; monk_record_set(&%s, \"%s\", %s); }\n",
				tmp, obj, tmp, target.Property, value)
		}
	}
}

// emitCondition generates the C expression for an `if`/`while` condition.
// Uses typed emission when possible so `while (i < N)` stays as raw C
// comparison instead of going through monk_is_truthy(monk_less(...)).
func (g *generator) emitCondition(e syntax.Expr) string {
	if g.info != nil {
		code, kind := g.emitExprTyped(e)
		// Strip ONE layer of outer parens when the whole expression is wrapped
		// in a single balanced pair. This avoids cc's -Wparentheses-equality
		// warning on code like `if ((a == b))`. Unsafe strips would change
		// meaning (e.g. `(a)+(b)` → `a)+(b`), so we verify balance.
		code = stripOuterParens(code)
		switch kind {
		case storeBool:
			return code
		case storeInt, storeFloat:
			// Truthiness: 0 is falsy, everything else is truthy.
			return "(" + code + ") != 0"
		}
	}
	return "monk_is_truthy(" + g.emitExpr(e) + ")"
}

// stripOuterParens removes ONE layer of enclosing parens if the first `(`
// matches the last `)` directly (i.e. the entire expression is wrapped).
// Returns s unchanged for everything else.
func stripOuterParens(s string) string {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return s
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i < len(s)-1 {
				// The matching `)` for the opening `(` is not at the end —
				// meaning the expression is NOT wrapped in a single balanced
				// pair. Example: "(a)+(b)" — depth hits 0 at index 2.
				return s
			}
		}
	}
	return s[1 : len(s)-1]
}

// emitIf handles if/else-if/else chains recursively.
func (g *generator) emitIf(s *syntax.IfStmt) {
	g.emitLine("    if (%s) {\n", g.emitCondition(s.Condition))
	thenSnap := g.saveStorage()
	for _, stmt := range s.Then.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(thenSnap)
	if s.Else != nil {
		switch e := s.Else.(type) {
		case *syntax.BlockStmt:
			g.emitLine("    } else {\n")
			elseSnap := g.saveStorage()
			for _, stmt := range e.Stmts {
				g.emitStmt(stmt)
			}
			g.restoreStorage(elseSnap)
			g.emitLine("    }\n")
			return
		case *syntax.IfStmt:
			// Recursive: handles arbitrary else-if chain depth
			g.body.WriteString("    } else ")
			g.emitIfInline(e)
			return
		}
	}
	g.emitLine("    }\n")
}

// emitIfInline emits an if statement without the leading indent (for else-if chains).
func (g *generator) emitIfInline(s *syntax.IfStmt) {
	fmt.Fprintf(&g.body, "if (%s) {\n", g.emitCondition(s.Condition))
	thenSnap := g.saveStorage()
	for _, stmt := range s.Then.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(thenSnap)
	if s.Else != nil {
		switch e := s.Else.(type) {
		case *syntax.BlockStmt:
			g.emitLine("    } else {\n")
			elseSnap := g.saveStorage()
			for _, stmt := range e.Stmts {
				g.emitStmt(stmt)
			}
			g.restoreStorage(elseSnap)
			g.emitLine("    }\n")
			return
		case *syntax.IfStmt:
			g.body.WriteString("    } else ")
			g.emitIfInline(e)
			return
		}
	}
	g.emitLine("    }\n")
}

func (g *generator) emitWhile(s *syntax.WhileStmt) {
	g.emitLine("    while (%s) {\n", g.emitCondition(s.Condition))
	snap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(snap)
	g.emitLine("    }\n")
}

func (g *generator) emitFor(s *syntax.ForStmt) {
	iter := g.emitExpr(s.Iterable)
	varName := mangleName(s.VarName)

	g.emitLine("    {\n")
	g.emitLine("        MonkValue _iter = %s;\n", iter)
	g.emitLine("        if (_iter.kind == MONK_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_deep_copy(_iter.array_val->data[_i]);\n", varName)
	// Inner block so user code can safely shadow the loop variable.
	g.emitLine("                {\n")
	arrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(arrSnap)
	g.emitLine("                }\n")
	g.emitLine("                monk_free(%s);\n", varName)
	g.emitLine("            }\n")
	g.emitLine("        } else if (_iter.kind == MONK_STRING) {\n")
	g.emitLine("            for (int64_t _i = 0; _iter.str_val[_i]; ) {\n")
	g.emitLine("                int64_t _clen = 1;\n")
	// Fix 6: bounds check for malformed UTF-8
	g.emitLine("                while (_iter.str_val[_i + _clen] && (_iter.str_val[_i + _clen] & 0xC0) == 0x80) _clen++;\n")
	g.emitLine("                char *_ch = malloc(_clen + 1);\n")
	g.emitLine("                if (!_ch) monk_panic(\"out of memory\");\n")
	g.emitLine("                memcpy(_ch, _iter.str_val + _i, _clen);\n")
	g.emitLine("                _ch[_clen] = '\\0';\n")
	// Note: value-semantics keeps this safe even though _ch is freed after
	// the body runs. Any downstream consumer (emitVarDecl, monk_append,
	// monk_record_set, …) deep-copies the loop variable on store, so a
	// captured reference owns its own string by the time we hit free(_ch).
	g.emitLine("                MonkValue %s = (MonkValue){.kind = MONK_STRING, .str_val = _ch};\n", varName)
	g.emitLine("                {\n")
	strSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(strSnap)
	g.emitLine("                }\n")
	g.emitLine("                free(_ch);\n")
	g.emitLine("                _i += _clen;\n")
	g.emitLine("            }\n")
	g.emitLine("        }\n")
	g.emitLine("    }\n")
}

func (g *generator) emitReturn(s *syntax.ReturnStmt) {
	if s.Value == nil {
		if g.retStorage == storeBoxed {
			g.emitLine("    return monk_none();\n")
		} else {
			// Bare return in a void-scalar fn doesn't really happen (type
			// checker rejects it), but be safe.
			g.emitLine("    return 0;\n")
		}
		return
	}
	if g.retStorage == storeBoxed {
		g.emitLine("    return %s;\n", g.emitExpr(s.Value))
		return
	}
	// Unboxed return: evaluate in typed form and coerce.
	code, kind := g.emitExprTyped(s.Value)
	g.emitLine("    return %s;\n", coerce(code, kind, g.retStorage))
}

func (g *generator) emitGuard(s *syntax.GuardStmt) {
	varName := mangleName(s.VarName)
	errName := mangleName(s.ErrorName)

	g.emitLine("    MonkValue %s = monk_none();\n", varName)
	g.emitLine("    {\n")
	g.emitLine("        MonkGuardContext _guard_ctx;\n")
	g.emitLine("        if (monk_guard_begin(&_guard_ctx) == 0) {\n")
	g.emitLine("            %s = %s;\n", varName, g.emitExpr(s.Expr))
	g.emitLine("            monk_guard_end(&_guard_ctx);\n")
	g.emitLine("        } else {\n")
	g.emitLine("            MonkValue %s = monk_current_error();\n", errName)
	for _, stmt := range s.Against.Stmts {
		g.emitStmt(stmt)
	}
	g.emitLine("            (void)%s;\n", errName)
	g.emitLine("        }\n")
	g.emitLine("    }\n")
}
