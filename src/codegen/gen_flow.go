package codegen

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Control-flow emission: if/else, while, for-in, return, guard.
// Conditions, loop bodies, and guard/against/throw patterns.

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
// IMPORTANT: never strip parens from GCC statement expressions `({...})` —
// removing the outer `()` turns a valid expression into a bare `{...}` block.
// Pass: `((a == b))` → `(a == b)`. Fail: `({int64_t t=x; t;})` → unchanged.
func stripOuterParens(s string) string {
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return s
	}
	// Statement expression: ({...}) — must keep outer parens.
	if len(s) >= 3 && s[1] == '{' {
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

// emitWhile emits a C `while` loop. The storage snapshot is taken before the
// body and restored after so that variables declared inside the loop don't
// pollute the outer storage map.
func (g *generator) emitWhile(s *syntax.WhileStmt) {
	g.emitLine("    while (%s) {\n", g.emitCondition(s.Condition))
	snap := g.saveStorage()
	// Bounds-check elision: track the loop counter's range so that array
	// accesses inside the body can be proven in-bounds.
	var boundVar string
	var boundPrev [2]int64
	var boundHad bool
	if g.constVals != nil {
		if varName, lo, hi, found := g.whileBoundsEntry(s.Condition); found {
			boundVar = varName
			boundPrev, boundHad = g.setVarBound(varName, lo, hi)
		}
	}
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	if boundVar != "" {
		g.restoreVarBound(boundVar, boundPrev, boundHad)
	}
	g.restoreStorage(snap)
	g.emitLine("    }\n")
}

func (g *generator) emitFor(s *syntax.ForStmt) {
	varName := g.mangledName(s.VarName)

	// Counter-loop fast path: `for x in range(N)` → `for(int64_t x=0; x<N; x++)`.
	// Avoids allocating an N-element array entirely.
	// Pass: `for i in range(10)` → counter. Fail: `for x in arr` → iterate.
	if call, ok := s.Iterable.(*syntax.CallExpr); ok {
		if callee, ok := call.Callee.(*syntax.IdentExpr); ok && callee.Name == "range" && len(call.Args) == 1 {
			nCode, nStore := g.emitExprTyped(call.Args[0])
			var nExpr string
			if nStore == storeInt {
				nExpr = nCode
			} else {
				nExpr = "(" + g.emitExpr(call.Args[0]) + ").int_val"
			}
			g.emitLine("    {\n")
			g.emitLine("        int64_t _n = %s;\n", nExpr)
			g.emitLine("        for (int64_t %s = 0; %s < _n; %s++) {\n", varName, varName, varName)
			g.emitLine("            {\n")
			snap := g.saveStorage()
			g.storage[varName] = storeInt
			for _, stmt := range s.Body.Stmts {
				g.emitStmt(stmt)
			}
			g.restoreStorage(snap)
			g.emitLine("            }\n")
			g.emitLine("        }\n")
			g.emitLine("    }\n")
			return
		}
	}

	iter := g.emitExpr(s.Iterable)

	// Fast path: when we know the iterable is a typed array at compile time,
	// emit a direct loop with a raw scalar loop variable — no boxing, no
	// runtime kind-dispatch. This is what makes for-in over int[] match C.
	if g.info != nil {
		if iterType, ok := g.info.Types[s.Iterable]; ok && iterType != nil {
			iterStore := storageFor(iterType)
			if isArrayStorage(iterStore) {
				ptrField := arrayPtrField(iterStore)
				var elemStore storageKind
				var cType string
				switch iterStore {
				case storeIntArray:
					elemStore = storeInt
					cType = "int64_t"
				case storeFloatArray:
					elemStore = storeFloat
					cType = "double"
				case storeBoolArray:
					elemStore = storeBool
					cType = "bool"
				}
				// Extract raw element from the iterable without conversion.
				// Handles both typed backing store (MONK_INT_ARRAY) and generic
				// MONK_ARRAY (reads .int_val from each MonkValue element).
				// No allocation, no copy — just a branch at loop setup.
				var kindName, genericField string
				switch iterStore {
				case storeIntArray:
					kindName = "MONK_INT_ARRAY"
					genericField = "int_val"
				case storeFloatArray:
					kindName = "MONK_FLOAT_ARRAY"
					genericField = "float_val"
				case storeBoolArray:
					kindName = "MONK_BOOL_ARRAY"
					genericField = "bool_val"
				}
				g.emitLine("    {\n")
				g.emitLine("        MonkValue _iter = %s;\n", iter)
				g.emitLine("        int64_t _len = (_iter.kind == %s) ? _iter.%s->length : _iter.array_val->length;\n",
					kindName, ptrField)
				g.emitLine("        for (int64_t _i = 0; _i < _len; _i++) {\n")
				g.emitLine("            %s %s = (_iter.kind == %s) ? _iter.%s->data[_i] : _iter.array_val->data[_i].%s;\n",
					cType, varName, kindName, ptrField, genericField)
				g.emitLine("            {\n")
				snap := g.saveStorage()
				g.storage[varName] = elemStore
				for _, stmt := range s.Body.Stmts {
					g.emitStmt(stmt)
				}
				g.restoreStorage(snap)
				g.emitLine("            }\n")
				g.emitLine("        }\n")
				g.emitLine("    }\n")
				return
			}
		}
	}

	g.emitLine("    {\n")
	g.emitLine("        MonkValue _iter = %s;\n", iter)
	// Generic MONK_ARRAY path
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
	// Typed backing-store array paths: iterate raw scalars, box each into MonkValue
	// so the loop body always sees a MonkValue (consistent with MONK_ARRAY path).
	g.emitLine("        } else if (_iter.kind == MONK_INT_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.int_array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_int(_iter.int_array_val->data[_i]);\n", varName)
	g.emitLine("                {\n")
	intArrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(intArrSnap)
	g.emitLine("                }\n")
	g.emitLine("            }\n")
	g.emitLine("        } else if (_iter.kind == MONK_FLOAT_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.float_array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_float(_iter.float_array_val->data[_i]);\n", varName)
	g.emitLine("                {\n")
	floatArrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(floatArrSnap)
	g.emitLine("                }\n")
	g.emitLine("            }\n")
	g.emitLine("        } else if (_iter.kind == MONK_BOOL_ARRAY) {\n")
	g.emitLine("            for (int64_t _i = 0; _i < _iter.bool_array_val->length; _i++) {\n")
	g.emitLine("                MonkValue %s = monk_bool(_iter.bool_array_val->data[_i]);\n", varName)
	g.emitLine("                {\n")
	boolArrSnap := g.saveStorage()
	for _, stmt := range s.Body.Stmts {
		g.emitStmt(stmt)
	}
	g.restoreStorage(boolArrSnap)
	g.emitLine("                }\n")
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

// emitReturn saves captured variables back to _self->captures (via
// emitCaptureSaveBack) then emits the C return. For unboxed functions the
// return value is coerced to the declared raw scalar type.
func (g *generator) emitReturn(s *syntax.ReturnStmt) {
	// Save captured variables back to _self->captures before returning.
	g.emitCaptureSaveBack()

	if s.Value == nil {
		if g.retStorage == storeBoxed {
			g.emitLine("    return monk_none();\n")
		} else {
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

// emitCaptureSaveBack writes local capture variables back to _self->captures
// so mutations persist across calls. Only emits if the current function has captures.
func (g *generator) emitCaptureSaveBack() {
	for i, name := range g.currentCaptures {
		mn := g.mangledName(name)
		g.emitLine("    _self->captures[%d] = %s;\n", i, mn)
	}
}

// emitGuard emits the setjmp-based guard construct. The guarded expression runs
// inside monk_guard_begin's if-branch; on throw the else-branch runs with the
// error value bound to errName. The guard variable is pre-initialised to none
// so it has a safe default even if the against block doesn't assign it.
func (g *generator) emitGuard(s *syntax.GuardStmt) {
	varName := g.mangledName(s.VarName)
	errName := g.mangledName(s.ErrorName)

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
