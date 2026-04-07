// Package types — statement-level type checks.
package types

import (
	"github.com/monkfromearth/monk-lang/syntax"
)

// resolveTypeDef turns a `type Point = {x: int, y: int}` into a Type.
func (c *checker) resolveTypeDef(td *syntax.TypeDeclStmt) (*Type, error) {
	if td.Definition.AliasOf != nil {
		t, err := c.resolveTypeExpr(td.Definition.AliasOf, td.Pos)
		if err != nil {
			return nil, err
		}
		return t, nil
	}
	// Record type definition.
	fields := make([]RecordTypeField, len(td.Definition.Fields))
	for i, f := range td.Definition.Fields {
		ft, err := c.resolveTypeExpr(&f.Type, td.Pos)
		if err != nil {
			return nil, err
		}
		fields[i] = RecordTypeField{Name: f.Name, Type: ft}
	}
	return &Type{Kind: KindRecord, Fields: fields, RecordName: td.Name}, nil
}

// resolveTypeExpr converts a parser TypeExpr to our Type, looking up named
// types in the typeDefs map. Position is passed in for error messages because
// TypeExpr doesn't carry a position of its own.
func (c *checker) resolveTypeExpr(te *syntax.TypeExpr, pos syntax.Pos) (*Type, error) {
	// Function type: (T1, T2) -> T3
	if te.IsFunc {
		params := make([]*Type, len(te.FuncParams))
		for i := range te.FuncParams {
			pt, err := c.resolveTypeExpr(&te.FuncParams[i], pos)
			if err != nil {
				return nil, err
			}
			params[i] = pt
		}
		ret, err := c.resolveTypeExpr(te.FuncReturn, pos)
		if err != nil {
			return nil, err
		}
		return FuncType(params, ret), nil
	}
	var base *Type
	switch te.Name {
	case "int":
		base = Int
	case "float":
		base = Float
	case "string":
		base = Str
	case "boolean":
		base = Bool
	case "none":
		base = None
	case "any":
		base = Any
	case "array":
		// Unparameterized array — accepts any element type.
		base = ArrayOf(Any)
	case "record":
		// Untyped record — structural, any shape. Reads are graceful (none),
		// writes to unknown fields are runtime errors.
		base = &Type{Kind: KindRecord}
	case "function":
		// Untyped function — we can't check calls against it, but we allow
		// the annotation.
		base = &Type{Kind: KindFunc, Return: Any}
	default:
		if t, ok := c.typeDefs[te.Name]; ok {
			base = t
		} else {
			return nil, newTypeError(pos, "unknown type '%s'", te.Name)
		}
	}
	if te.IsArray {
		base = ArrayOf(base)
	}
	if te.Optional {
		base = OptionalOf(base)
	}
	return base, nil
}

// funcSignature extracts a Func type from a FuncExpr without checking its body.
// Used during hoisting so recursive/forward references typecheck.
func (c *checker) funcSignature(fn *syntax.FuncExpr) (*Type, error) {
	params := make([]*Type, len(fn.Params))
	seenDefault := false
	for i, p := range fn.Params {
		// Distinguish "missing annotation" (Name == "" AND not a function
		// type) from a legitimate function-type annotation which has IsFunc
		// true and Name == "". Untyped params fall back to Any for ergonomics.
		if p.Type.Name == "" && !p.Type.IsFunc {
			params[i] = Any
		} else {
			pt, err := c.resolveTypeExpr(&p.Type, fn.Pos)
			if err != nil {
				return nil, err
			}
			params[i] = pt
		}
		// Validate defaults: must be trailing (no required param after a default).
		if p.Default != nil {
			seenDefault = true
			dt, err := c.inferExpr(p.Default)
			if err != nil {
				return nil, err
			}
			if !AssignableTo(dt, params[i]) {
				return nil, newTypeError(fn.Pos,
					"default value for parameter '%s': cannot assign %s to %s",
					p.Name, dt, params[i])
			}
		} else if seenDefault {
			return nil, newTypeError(fn.Pos,
				"required parameter '%s' cannot follow a parameter with a default value",
				p.Name)
		}
	}
	var ret *Type
	if fn.ReturnType.Name == "" && !fn.ReturnType.IsFunc {
		ret = Any
	} else {
		var err error
		ret, err = c.resolveTypeExpr(&fn.ReturnType, fn.Pos)
		if err != nil {
			return nil, err
		}
	}
	ft := FuncType(params, ret)
	// Compute MinParams: count leading required (non-default) params.
	ft.MinParams = len(params)
	for i := len(fn.Params) - 1; i >= 0; i-- {
		if fn.Params[i].Default != nil {
			ft.MinParams = i
		} else {
			break
		}
	}
	return ft, nil
}

// ─── Var decl & assignment ────────────────────────────────────────────────

// checkVarDecl infers the RHS type, validates it against any explicit annotation,
// and declares the binding in the current scope. The resolved type is stored in
// c.info.Decls for codegen's storage-kind decision.
func (c *checker) checkVarDecl(s *syntax.VarDeclStmt) error {
	valueType, err := c.inferExpr(s.Value)
	if err != nil {
		return err
	}
	var declared *Type
	if s.Type != nil {
		declared, err = c.resolveTypeExpr(s.Type, s.Pos)
		if err != nil {
			return err
		}
		if !AssignableTo(valueType, declared) {
			return newTypeError(s.Pos,
				"cannot assign %s to %s in declaration of '%s'",
				valueType, declared, s.Name)
		}
	} else {
		// First-assignment inference. Strip "anonymous record" down to its
		// structural shape; we don't want every untyped record named "{...}".
		declared = valueType
	}
	// Arrays that start as [] inherit element type from context if possible.
	// Here the context is an explicit annotation, already handled above.

	// Redeclaring an imported name is an error. Without this check, the local
	// `let` silently shadows the import — codegen uses importMap for the name
	// so the local declaration is effectively dead.
	// Pass: `use add from "./m"; let x = 1`  (different names)
	// Fail: `use add from "./m"; let add = 2` (shadows import)
	if c.importedNames[s.Name] {
		return newTypeError(s.Pos, "'%s' is already declared via import", s.Name)
	}

	c.scope.declare(s.Name, declared, s.IsConst)
	c.info.Decls[s] = declared
	return nil
}

// checkAssign validates an assignment statement. Supports three target forms:
// plain identifier, index expression (array element), and property expression
// (record field). Checks compound operators via checkCompoundOp.
func (c *checker) checkAssign(s *syntax.AssignStmt) error {
	switch target := s.Target.(type) {
	case *syntax.IdentExpr:
		b := c.scope.lookup(target.Name)
		if b == nil {
			return newTypeError(s.Pos, "assignment to undefined variable '%s'", target.Name)
		}
		if b.IsConst {
			return newTypeError(s.Pos, "cannot assign to const '%s'", target.Name)
		}
		rhsType, err := c.inferExpr(s.Value)
		if err != nil {
			return err
		}
		if err := c.checkCompoundOp(s, b.Type, rhsType); err != nil {
			return err
		}
		if !AssignableTo(rhsType, b.Type) {
			return newTypeError(s.Pos,
				"cannot assign %s to variable '%s' of type %s",
				rhsType, target.Name, b.Type)
		}
	case *syntax.IndexExpr:
		collType, err := c.inferExpr(target.Object)
		if err != nil {
			return err
		}
		if collType.Kind != KindArray && collType.Kind != KindAny {
			return newTypeError(s.Pos, "index assignment requires an array, got %s", collType)
		}
		idxType, err := c.inferExpr(target.Index)
		if err != nil {
			return err
		}
		if idxType.Kind != KindInt && idxType.Kind != KindAny {
			return newTypeError(s.Pos, "array index must be int, got %s", idxType)
		}
		rhsType, err := c.inferExpr(s.Value)
		if err != nil {
			return err
		}
		if collType.Kind == KindArray {
			if err := c.checkCompoundOp(s, collType.Elem, rhsType); err != nil {
				return err
			}
			if !AssignableTo(rhsType, collType.Elem) {
				return newTypeError(s.Pos,
					"cannot assign %s to %s element", rhsType, collType)
			}
		}
	case *syntax.PropertyExpr:
		recType, err := c.inferExpr(target.Object)
		if err != nil {
			return err
		}
		if recType.Kind != KindRecord && recType.Kind != KindAny {
			return newTypeError(s.Pos, "property assignment requires a record, got %s", recType)
		}
		rhsType, err := c.inferExpr(s.Value)
		if err != nil {
			return err
		}
		if recType.Kind == KindRecord {
			var fieldType *Type
			for i := range recType.Fields {
				if recType.Fields[i].Name == target.Property {
					fieldType = recType.Fields[i].Type
					break
				}
			}
			if fieldType == nil {
				// Records have fixed shape from creation — no adding new fields.
				return newTypeError(s.Pos,
					"%s has no field '%s'", recType, target.Property)
			}
			if err := c.checkCompoundOp(s, fieldType, rhsType); err != nil {
				return err
			}
			if !AssignableTo(rhsType, fieldType) {
				return newTypeError(s.Pos,
					"cannot assign %s to field '%s' of type %s",
					rhsType, target.Property, fieldType)
			}
		}
	default:
		return newTypeError(s.Pos, "invalid assignment target")
	}
	return nil
}

// checkCompoundOp verifies that `target op= value` is a well-typed binary
// operation. For plain `=` (Op == Equal), this is a no-op — only AssignableTo
// matters. For compound ops, the implied operation must itself be legal:
//
//	x -= y  →  x = x - y       (target and value must both be numeric)
//	x += y  →  x = x + y       (numeric-numeric OR string-string)
//	x *= y  →  numeric only
//	x /= y  →  numeric only
//	x %= y  →  numeric only
//
// Without this, `let s = "hi"; s -= "world"` passes the checker and crashes
// at runtime with "cannot subtract these types", violating the spec's
// "explicit over implicit, no hidden errors" rule.
func (c *checker) checkCompoundOp(s *syntax.AssignStmt, targetType, valueType *Type) error {
	if s.Op == syntax.Equal {
		return nil
	}
	// Any on either side silences the check — runtime will handle it.
	if targetType.Kind == KindAny || valueType.Kind == KindAny {
		return nil
	}
	// += has a string-concat overload.
	if s.Op == syntax.PlusEqual {
		if targetType.Kind == KindStr && valueType.Kind == KindStr {
			return nil
		}
		if targetType.Kind == KindStr || valueType.Kind == KindStr {
			return newTypeError(s.Pos,
				"operator += cannot mix string and %s",
				otherNonStr(targetType, valueType))
		}
	}
	// All other compound ops and += on non-strings require numeric both sides.
	if !isNumericOrAny(targetType) || !isNumericOrAny(valueType) {
		return newTypeError(s.Pos,
			"operator %s requires numeric operands, got %s and %s",
			compoundOpString(s.Op), targetType, valueType)
	}
	return nil
}

func compoundOpString(k syntax.TokenKind) string {
	switch k {
	case syntax.PlusEqual:
		return "+="
	case syntax.MinusEqual:
		return "-="
	case syntax.StarEqual:
		return "*="
	case syntax.SlashEqual:
		return "/="
	case syntax.PercentEqual:
		return "%="
	}
	return "<op>="
}

func otherNonStr(a, b *Type) string {
	if a.Kind == KindStr {
		return b.String()
	}
	return a.String()
}

// ─── Control flow ──────────────────────────────────────────────────────────

// checkIf type-checks the condition and both branches of an if statement.
// Any type is legal as a condition — truthiness is a runtime property.
func (c *checker) checkIf(s *syntax.IfStmt) error {
	condType, err := c.inferExpr(s.Condition)
	if err != nil {
		return err
	}
	// Any type is truthy-checkable (truthiness rules apply at runtime).
	_ = condType
	if err := c.checkBlock(s.Then, true); err != nil {
		return err
	}
	if s.Else != nil {
		return c.checkStmt(s.Else)
	}
	return nil
}

// checkWhile type-checks the condition and body. Bumps inLoop so break/continue
// inside the body are legal.
func (c *checker) checkWhile(s *syntax.WhileStmt) error {
	if _, err := c.inferExpr(s.Condition); err != nil {
		return err
	}
	c.inLoop++
	defer func() { c.inLoop-- }()
	return c.checkBlock(s.Body, true)
}

// checkFor type-checks `for varName in iterable { body }`. The loop variable
// is declared const in its own scope (parent of the body), so body-level
// `let i = ...` cannot shadow and reassign it — matching the spec guarantee
// that the for-loop variable is immutable.
func (c *checker) checkFor(s *syntax.ForStmt) error {
	iterType, err := c.inferExpr(s.Iterable)
	if err != nil {
		return err
	}
	var elemType *Type
	switch iterType.Kind {
	case KindArray:
		elemType = iterType.Elem
	case KindStr:
		elemType = Str
	case KindAny:
		elemType = Any
	default:
		return newTypeError(s.Pos, "cannot iterate over %s", iterType)
	}
	// The loop variable lives in its own scope (parent of the body). The body
	// then gets its own child scope — otherwise a `let i = ...` inside the
	// body would replace the const loop-variable binding and let the user
	// mutate it, defeating the spec's "loop variable is const" guarantee.
	c.scope = newScopeOf(c.scope)
	defer func() { c.scope = c.scope.parent }()
	c.scope.declare(s.VarName, elemType, true)
	c.inLoop++
	defer func() { c.inLoop-- }()
	return c.checkBlock(s.Body, true)
}

// checkReturn verifies that the returned type is compatible with the enclosing
// function's declared return type. A bare return is only valid when the
// function returns none or any.
func (c *checker) checkReturn(s *syntax.ReturnStmt) error {
	if c.returnType == nil {
		return newTypeError(s.Pos, "return outside of function")
	}
	if s.Value == nil {
		if c.returnType.Kind != KindNone && c.returnType.Kind != KindAny {
			return newTypeError(s.Pos,
				"bare return in function returning %s", c.returnType)
		}
		return nil
	}
	rt, err := c.inferExpr(s.Value)
	if err != nil {
		return err
	}
	if !AssignableTo(rt, c.returnType) {
		return newTypeError(s.Pos,
			"cannot return %s from function returning %s", rt, c.returnType)
	}
	return nil
}

// checkGuard type-checks a guard statement. The guarded variable is declared
// in the enclosing scope (so it's accessible after the guard block). The error
// variable is bound as Any inside the against block only.
func (c *checker) checkGuard(s *syntax.GuardStmt) error {
	rt, err := c.inferExpr(s.Expr)
	if err != nil {
		return err
	}
	// Declare the guard var in the enclosing scope (per spec). If the against
	// block doesn't reassign it, it defaults to none, so the type is T? —
	// but for simplicity we just use T.
	c.scope.declare(s.VarName, rt, false)
	// The error variable is an Any inside the against block.
	c.scope = newScopeOf(c.scope)
	c.scope.declare(s.ErrorName, Any, false)
	err = c.checkBlock(s.Against, false)
	c.scope = c.scope.parent
	return err
}
