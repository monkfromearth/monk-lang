package syntax

// AST node types and shared data structures.
//
// Expression nodes live in ast_expr.go, statement nodes in ast_stmt.go.
// This file holds the interfaces, root Program, and the shared types that
// appear in both (Param, TypeExpr, TypeDefExpr, TypeField, RecordField).

// Pos records where an AST node starts in the source. Used by code generation
// for #line directives and by error messages for source locations.
type Pos struct {
	Line   int
	Column int
}

// Position is embedded (via Pos) on every concrete node, so it satisfies the
// Node interface without each type needing its own method.
func (p Pos) Position() Pos { return p }

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodeKind() string
	Position() Pos
}

// Expr is the interface for all expression nodes.
type Expr interface {
	Node
	exprNode()
}

// Stmt is the interface for all statement nodes.
type Stmt interface {
	Node
	stmtNode()
}

// Program is the root AST node — a list of top-level statements.
type Program struct {
	Stmts []Stmt
}

func (p *Program) nodeKind() string { return "Program" }
func (p *Program) Position() Pos    { return Pos{1, 1} }

// --- Shared data types used by both expression and statement nodes ---

// Param is a function parameter with name, type, and optional default value.
type Param struct {
	Name    string
	Type    TypeExpr
	Default Expr // nil if no default
}

// TypeExpr represents a type annotation: int, string, int[], int?,
// or a function type: (int, int) -> int.
//
// For plain types, Name is populated. For function types, IsFunc is true and
// FuncParams/FuncReturn hold the signature (Name is "" in that case). The
// IsArray and Optional modifiers apply after the base type, so `(int) -> int?`
// means a function returning an optional int, not an optional function.
type TypeExpr struct {
	Name       string     // "int", "string", custom type name
	IsArray    bool       // true for "int[]"
	Optional   bool       // true for "int?"
	IsFunc     bool       // true for "(T, T) -> T"
	FuncParams []TypeExpr // populated when IsFunc is true
	FuncReturn *TypeExpr  // populated when IsFunc is true
}

// TypeDefExpr represents the right side of a type declaration.
// Either a type alias (AliasOf) or a record type definition (Fields).
type TypeDefExpr struct {
	AliasOf *TypeExpr   // for "type UserId = int"
	Fields  []TypeField // for "type Point = { x: int, y: int }"
}

// TypeField is a field in a record type definition (not a record literal).
// Separate from RecordField because type fields have type annotations, not value expressions.
type TypeField struct {
	Name string
	Type TypeExpr
}

// RecordField is a key-value pair in a record literal.
type RecordField struct {
	Key   string
	Value Expr
}
