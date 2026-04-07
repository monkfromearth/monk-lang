// Package types — the Checker walks an AST and reports type errors.
package types

import (
	"github.com/monkfromearth/monk-lang/syntax"
)

// Info carries the results of a type-check. Codegen consults it to decide
// which C type to emit for each variable and whether to use raw C arithmetic
// vs. runtime tagged-union dispatch.
//
// Types: maps each expression node to its computed type. Every Expr walked
//
//	by inferExpr ends up here.
//
// Decls: maps each VarDeclStmt to its declared type (the annotation if
//
//	present, otherwise the first-assignment-inferred type).
//
// Funcs: maps each FuncExpr to its full signature. Codegen uses this to
//
//	generate unboxed function signatures when params/return are scalar.
type Info struct {
	Types map[syntax.Expr]*Type
	Decls map[*syntax.VarDeclStmt]*Type
	Funcs map[*syntax.FuncExpr]*Type
}

// Check runs the type checker on a parsed program. Returns the collected
// type Info and the first error encountered (or nil on success).
func Check(prog *syntax.Program) (*Info, error) {
	c := newChecker()
	if err := c.checkProgram(prog); err != nil {
		return nil, err
	}
	return c.info, nil
}

// Binding associates a name with its declared type and mutability in a scope.
type Binding struct {
	Type    *Type
	IsConst bool
}

// scope is a single lexical scope. parent is the enclosing scope, or nil
// for the top-level. Name lookup walks the chain.
type scope struct {
	parent *scope
	names  map[string]*Binding
}

// newScope allocates a fresh scope chained to parent. Pass nil for the top-level scope.
func newScope(parent *scope) *scope {
	return &scope{parent: parent, names: make(map[string]*Binding)}
}

// lookup walks the scope chain and returns the binding for name, or nil if
// the name is not declared in any enclosing scope.
func (s *scope) lookup(name string) *Binding {
	if b, ok := s.names[name]; ok {
		return b
	}
	if s.parent != nil {
		return s.parent.lookup(name)
	}
	return nil
}

// declare adds name to the current scope. Shadowing an outer binding is
// intentional (inner let/const always wins within its block).
func (s *scope) declare(name string, t *Type, isConst bool) {
	s.names[name] = &Binding{Type: t, IsConst: isConst}
}

// checker carries state across the walk.
type checker struct {
	scope      *scope
	typeDefs   map[string]*Type // named types from `type Point = ...`
	returnType *Type            // expected return type of the current function
	inLoop     int              // break/continue legality
	info       *Info            // collected type info, returned to codegen
}

// newChecker creates a checker with a fresh top-level scope pre-populated
// with all builtin function signatures.
func newChecker() *checker {
	c := &checker{
		scope:    newScope(nil),
		typeDefs: make(map[string]*Type),
		info: &Info{
			Types: make(map[syntax.Expr]*Type),
			Decls: make(map[*syntax.VarDeclStmt]*Type),
			Funcs: make(map[*syntax.FuncExpr]*Type),
		},
	}
	c.declareBuiltins()
	return c
}

// declareBuiltins populates the top-level scope with the 40+ builtin functions
// so calls to show/to_string/length/etc. type-check correctly.
//
// We use Any liberally here. A tighter signature (e.g. length((string|array)) -> int)
// would require union types. That's future work — today we accept "builtins
// are correctly typed at runtime" and focus the checker on user code.
func (c *checker) declareBuiltins() {
	// Output & conversion
	c.scope.declare("show", FuncType([]*Type{Any}, None), true)
	c.scope.declare("to_string", FuncType([]*Type{Any}, Str), true)
	c.scope.declare("to_int", FuncType([]*Type{Any}, Int), true)
	c.scope.declare("to_float", FuncType([]*Type{Any}, Float), true)

	// Type checking — all return boolean except typeof
	c.scope.declare("typeof", FuncType([]*Type{Any}, Str), true)
	for _, name := range []string{
		"is_number", "is_string", "is_boolean", "is_array",
		"is_record", "is_function", "is_none",
	} {
		c.scope.declare(name, FuncType([]*Type{Any}, Bool), true)
	}

	// String/array ops that accept "array or string" — we use Any for the
	// input and the "right" type for output until we have union types.
	c.scope.declare("length", FuncType([]*Type{Any}, Int), true)
	c.scope.declare("substring", FuncType([]*Type{Str, Int, Int}, Str), true)
	c.scope.declare("index_of", FuncType([]*Type{Str, Str}, Int), true)
	c.scope.declare("split", FuncType([]*Type{Str, Str}, ArrayOf(Str)), true)
	c.scope.declare("trim", FuncType([]*Type{Str}, Str), true)
	c.scope.declare("to_upper_case", FuncType([]*Type{Str}, Str), true)
	c.scope.declare("to_lower_case", FuncType([]*Type{Str}, Str), true)

	// Array ops — input element type unknown without generics, so we use Any[]
	anyArr := ArrayOf(Any)
	c.scope.declare("append", FuncType([]*Type{anyArr, Any}, anyArr), true)
	c.scope.declare("prepend", FuncType([]*Type{anyArr, Any}, anyArr), true)
	c.scope.declare("pop", FuncType([]*Type{anyArr}, anyArr), true)
	c.scope.declare("drop", FuncType([]*Type{anyArr, Int}, anyArr), true)
	c.scope.declare("take", FuncType([]*Type{anyArr, Int}, anyArr), true)
	c.scope.declare("slice", FuncType([]*Type{anyArr, Int, Int}, anyArr), true)
	c.scope.declare("range", FuncType([]*Type{Int}, ArrayOf(Int)), true)
	// fill(n, value) → T[] where T is the value's type.
	// Declared as (int, any) → any[]; inferCall refines the return type.
	c.scope.declare("fill", FuncType([]*Type{Int, Any}, anyArr), true)

	// Math — accept numeric, return float (most math genuinely returns float).
	for _, name := range []string{
		"floor", "ceil", "round", "sqrt", "log", "log10", "exp",
		"sin", "cos", "tan", "asin", "acos", "atan",
	} {
		c.scope.declare(name, FuncType([]*Type{Any}, Float), true)
	}
	c.scope.declare("pow", FuncType([]*Type{Any, Any}, Float), true)
	// abs/min/max preserve the input type — return Any so int→int, float→float.
	c.scope.declare("abs", FuncType([]*Type{Any}, Any), true)
	c.scope.declare("min", FuncType([]*Type{Any, Any}, Any), true)
	c.scope.declare("max", FuncType([]*Type{Any, Any}, Any), true)

	// Higher-order array functions
	c.scope.declare("map", FuncType([]*Type{ArrayOf(Any), FuncType([]*Type{Any}, Any)}, ArrayOf(Any)), true)
	c.scope.declare("filter", FuncType([]*Type{ArrayOf(Any), FuncType([]*Type{Any}, Bool)}, ArrayOf(Any)), true)
	c.scope.declare("reduce", FuncType([]*Type{ArrayOf(Any), FuncType([]*Type{Any, Any}, Any), Any}, Any), true)

	// File / env
	c.scope.declare("file_read", FuncType([]*Type{Str}, Str), true)
	c.scope.declare("file_write", FuncType([]*Type{Str, Str}, None), true)
	c.scope.declare("file_exists", FuncType([]*Type{Str}, Bool), true)
	c.scope.declare("env_get", FuncType([]*Type{Str}, Str), true)
	c.scope.declare("exit", FuncType([]*Type{Int}, None), true)
	c.scope.declare("args", FuncType(nil, ArrayOf(Str)), true)
}

// ─── Program / statements ──────────────────────────────────────────────────

// checkProgram runs in three sub-passes: (1) resolve named type declarations,
// (2) hoist top-level function signatures for recursive/forward references,
// (3) type-check every statement in order.
func (c *checker) checkProgram(prog *syntax.Program) error {
	// Two-pass: (1) hoist all function declarations and type defs so forward
	// references work, (2) check everything.
	// We actually need FOUR things declared before any check that might use
	// them: type definitions, then function signatures, then the rest.
	for _, stmt := range prog.Stmts {
		if td, ok := stmt.(*syntax.TypeDeclStmt); ok {
			t, err := c.resolveTypeDef(td)
			if err != nil {
				return err
			}
			c.typeDefs[td.Name] = t
		}
	}
	// Hoist top-level `let name = <funcexpr>` bindings, so recursion +
	// forward-references work. (The spec names this as "Self-reference for
	// recursion" and funcs-as-values; we implement it at type-check time.)
	for _, stmt := range prog.Stmts {
		if vd, ok := stmt.(*syntax.VarDeclStmt); ok {
			if fn, ok := vd.Value.(*syntax.FuncExpr); ok {
				sig, err := c.funcSignature(fn)
				if err != nil {
					return err
				}
				c.scope.declare(vd.Name, sig, vd.IsConst)
			}
		}
	}
	for _, stmt := range prog.Stmts {
		if err := c.checkStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

// checkStmt dispatches to the appropriate check function for each statement kind.
func (c *checker) checkStmt(stmt syntax.Stmt) error {
	switch s := stmt.(type) {
	case *syntax.VarDeclStmt:
		return c.checkVarDecl(s)
	case *syntax.AssignStmt:
		return c.checkAssign(s)
	case *syntax.ExprStmt:
		_, err := c.inferExpr(s.Expr)
		return err
	case *syntax.BlockStmt:
		return c.checkBlock(s, true)
	case *syntax.IfStmt:
		return c.checkIf(s)
	case *syntax.WhileStmt:
		return c.checkWhile(s)
	case *syntax.ForStmt:
		return c.checkFor(s)
	case *syntax.ReturnStmt:
		return c.checkReturn(s)
	case *syntax.BreakStmt:
		if c.inLoop == 0 {
			return newTypeError(s.Pos, "break outside of loop")
		}
		return nil
	case *syntax.ContinueStmt:
		if c.inLoop == 0 {
			return newTypeError(s.Pos, "continue outside of loop")
		}
		return nil
	case *syntax.GuardStmt:
		return c.checkGuard(s)
	case *syntax.TypeDeclStmt:
		return nil // already resolved during hoisting
	case *syntax.UseStmt, *syntax.ExportStmt:
		return nil // phase 7
	}
	return nil
}

// checkBlock type-checks all statements in a block. When newScope is true, a
// child scope is pushed for the block and popped on return. Passing false is
// used by guard's against block, which shares the outer scope.
func (c *checker) checkBlock(b *syntax.BlockStmt, newScope bool) error {
	if newScope {
		c.scope = newScopeOf(c.scope)
		defer func() { c.scope = c.scope.parent }()
	}
	for _, s := range b.Stmts {
		if err := c.checkStmt(s); err != nil {
			return err
		}
	}
	return nil
}

// newScopeOf is a named alias for newScope used at call sites where the intent
// ("create a scope OF this parent") aids readability.
func newScopeOf(parent *scope) *scope { return newScope(parent) }
