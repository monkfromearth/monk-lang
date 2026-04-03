package syntax

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodeKind() string
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

// --- Expression nodes ---

// NumberExpr represents an integer or float literal.
type NumberExpr struct {
	Value string // raw text: "42", "3.14", "0xFF"
	IsInt bool   // true = int, false = float
}

// StringExpr represents a double-quoted string literal (contents without quotes).
type StringExpr struct {
	Value string
}

// TemplateExpr represents a backtick template literal (contents without backticks).
type TemplateExpr struct {
	Value string
}

// BoolExpr represents a boolean literal (true or false).
type BoolExpr struct {
	Value bool
}

// NoneExpr represents the none literal.
type NoneExpr struct{}

// IdentExpr represents an identifier (variable name, function name, type name).
type IdentExpr struct {
	Name string
}

// UnaryExpr represents a unary operation: -x, not x, !x, ~x.
type UnaryExpr struct {
	Op      TokenKind
	Operand Expr
}

// BinaryExpr represents a binary operation: a + b, x == y, etc.
type BinaryExpr struct {
	Left  Expr
	Op    TokenKind
	Right Expr
}

// AssignExpr represents assignment: x = 1, arr[0] = 1, record.field = 1.
// Also compound assignment: x += 1, x -= 1, etc.
type AssignExpr struct {
	Target Expr      // IdentExpr, IndexExpr, or PropertyExpr
	Op     TokenKind // Equal, PlusEqual, MinusEqual, etc.
	Value  Expr
}

// CallExpr represents a function call: f(a, b).
type CallExpr struct {
	Callee Expr
	Args   []Expr
}

// IndexExpr represents index access: arr[i], "hello"[0].
type IndexExpr struct {
	Object Expr
	Index  Expr
}

// PropertyExpr represents property access: record.field.
type PropertyExpr struct {
	Object   Expr
	Property string
}

// ArrayExpr represents an array literal: [1, 2, 3].
type ArrayExpr struct {
	Elements []Expr
}

// RecordExpr represents a record literal: {name: "Alice", age: 30}.
type RecordExpr struct {
	Fields []RecordField
}

// RecordField is a key-value pair in a record literal or type definition.
type RecordField struct {
	Key   string
	Value Expr
}

// FuncExpr represents a function expression: (a int, b int) int { return a + b }.
type FuncExpr struct {
	Params     []Param
	ReturnType TypeExpr
	Body       *BlockStmt
}

// Param is a function parameter with name, type, and optional default value.
type Param struct {
	Name    string
	Type    TypeExpr
	Default Expr // nil if no default
}

// TypeExpr represents a type annotation: int, string, int[], int?.
type TypeExpr struct {
	Name     string // "int", "string", custom type name
	IsArray  bool   // true for "int[]"
	Optional bool   // true for "int?"
}

// ThrowExpr represents a throw expression: throw "error".
type ThrowExpr struct {
	Value Expr
}

// --- Statement nodes ---

// VarDeclStmt represents a variable declaration: let x = 1 or const y = 2.
type VarDeclStmt struct {
	Name    string
	Type    *TypeExpr // nil if no annotation
	Value   Expr
	IsConst bool
}

// ExprStmt wraps an expression used as a statement.
type ExprStmt struct {
	Expr Expr
}

// BlockStmt represents a block of statements: { stmt1; stmt2 }.
// Created by if/while/for/guard/function bodies, not standalone.
type BlockStmt struct {
	Stmts []Stmt
}

// IfStmt represents an if/else/else-if statement.
type IfStmt struct {
	Condition Expr
	Then      *BlockStmt
	Else      Stmt // *BlockStmt (else) or *IfStmt (else-if) or nil
}

// WhileStmt represents a while loop.
type WhileStmt struct {
	Condition Expr
	Body      *BlockStmt
}

// ForStmt represents a for-in loop: for x in iterable { }.
type ForStmt struct {
	VarName  string
	Iterable Expr
	Body     *BlockStmt
}

// ReturnStmt represents a return statement. Value is nil for bare return.
type ReturnStmt struct {
	Value Expr
}

// BreakStmt represents a break statement inside a loop.
type BreakStmt struct{}

// ContinueStmt represents a continue statement inside a loop.
type ContinueStmt struct{}

// GuardStmt represents guard/against error handling.
type GuardStmt struct {
	VarName   string     // variable that receives the result
	Expr      Expr       // expression that might throw
	ErrorName string     // variable that receives the error
	Against   *BlockStmt // error handling block
}

// TypeDeclStmt represents a type declaration: type Point = { x: int, y: int }.
type TypeDeclStmt struct {
	Name       string
	Definition TypeDefExpr
}

// TypeDefExpr represents the right side of a type declaration.
// Either a type alias (AliasOf) or a record type definition (Fields).
type TypeDefExpr struct {
	AliasOf *TypeExpr     // for "type UserId = int"
	Fields  []RecordField // for "type Point = { x: int, y: int }"
}

// UseStmt represents an import: use helper from "./utils".
type UseStmt struct {
	Names  []string // imported names
	Star   bool     // true for "use * from"
	Alias  string   // "use X as Alias from"
	Source string   // the module path string
}

// ExportStmt represents an export: export helper.
type ExportStmt struct {
	Stmt Stmt // the declaration being exported
}

// --- Interface marker methods ---

func (e *NumberExpr) nodeKind() string   { return "NumberExpr" }
func (e *StringExpr) nodeKind() string   { return "StringExpr" }
func (e *TemplateExpr) nodeKind() string { return "TemplateExpr" }
func (e *BoolExpr) nodeKind() string     { return "BoolExpr" }
func (e *NoneExpr) nodeKind() string     { return "NoneExpr" }
func (e *IdentExpr) nodeKind() string    { return "IdentExpr" }
func (e *UnaryExpr) nodeKind() string    { return "UnaryExpr" }
func (e *BinaryExpr) nodeKind() string   { return "BinaryExpr" }
func (e *AssignExpr) nodeKind() string   { return "AssignExpr" }
func (e *CallExpr) nodeKind() string     { return "CallExpr" }
func (e *IndexExpr) nodeKind() string    { return "IndexExpr" }
func (e *PropertyExpr) nodeKind() string { return "PropertyExpr" }
func (e *ArrayExpr) nodeKind() string    { return "ArrayExpr" }
func (e *RecordExpr) nodeKind() string   { return "RecordExpr" }
func (e *FuncExpr) nodeKind() string     { return "FuncExpr" }
func (e *ThrowExpr) nodeKind() string    { return "ThrowExpr" }

func (s *VarDeclStmt) nodeKind() string  { return "VarDeclStmt" }
func (s *ExprStmt) nodeKind() string     { return "ExprStmt" }
func (s *BlockStmt) nodeKind() string    { return "BlockStmt" }
func (s *IfStmt) nodeKind() string       { return "IfStmt" }
func (s *WhileStmt) nodeKind() string    { return "WhileStmt" }
func (s *ForStmt) nodeKind() string      { return "ForStmt" }
func (s *ReturnStmt) nodeKind() string   { return "ReturnStmt" }
func (s *BreakStmt) nodeKind() string    { return "BreakStmt" }
func (s *ContinueStmt) nodeKind() string { return "ContinueStmt" }
func (s *GuardStmt) nodeKind() string    { return "GuardStmt" }
func (s *TypeDeclStmt) nodeKind() string { return "TypeDeclStmt" }
func (s *UseStmt) nodeKind() string      { return "UseStmt" }
func (s *ExportStmt) nodeKind() string   { return "ExportStmt" }

func (e *NumberExpr) exprNode()   {}
func (e *StringExpr) exprNode()   {}
func (e *TemplateExpr) exprNode() {}
func (e *BoolExpr) exprNode()     {}
func (e *NoneExpr) exprNode()     {}
func (e *IdentExpr) exprNode()    {}
func (e *UnaryExpr) exprNode()    {}
func (e *BinaryExpr) exprNode()   {}
func (e *AssignExpr) exprNode()   {}
func (e *CallExpr) exprNode()     {}
func (e *IndexExpr) exprNode()    {}
func (e *PropertyExpr) exprNode() {}
func (e *ArrayExpr) exprNode()    {}
func (e *RecordExpr) exprNode()   {}
func (e *FuncExpr) exprNode()     {}
func (e *ThrowExpr) exprNode()    {}

func (s *VarDeclStmt) stmtNode()  {}
func (s *ExprStmt) stmtNode()     {}
func (s *BlockStmt) stmtNode()    {}
func (s *IfStmt) stmtNode()       {}
func (s *WhileStmt) stmtNode()    {}
func (s *ForStmt) stmtNode()      {}
func (s *ReturnStmt) stmtNode()   {}
func (s *BreakStmt) stmtNode()    {}
func (s *ContinueStmt) stmtNode() {}
func (s *GuardStmt) stmtNode()    {}
func (s *TypeDeclStmt) stmtNode() {}
func (s *UseStmt) stmtNode()      {}
func (s *ExportStmt) stmtNode()   {}
