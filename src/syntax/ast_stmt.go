package syntax

// Statement nodes.

// VarDeclStmt represents a variable declaration: let x = 1 or const y = 2.
type VarDeclStmt struct {
	Pos
	Name    string
	Type    *TypeExpr // nil if no annotation
	Value   Expr
	IsConst bool
}

// AssignStmt represents assignment: x = 1, arr[0] = 1, record.field = 1.
// Design decision: assignment is a STATEMENT, not an expression.
// It cannot appear inside conditions, function args, or other expressions.
type AssignStmt struct {
	Pos
	Target Expr      // IdentExpr, IndexExpr, or PropertyExpr
	Op     TokenKind // Equal, PlusEqual, MinusEqual, etc.
	Value  Expr
}

// ExprStmt wraps an expression used as a statement.
type ExprStmt struct {
	Pos
	Expr Expr
}

// BlockStmt represents a block of statements: { stmt1; stmt2 }.
// Created by if/while/for/guard/function bodies, not standalone.
type BlockStmt struct {
	Pos
	Stmts []Stmt
}

// IfStmt represents an if/else/else-if statement.
type IfStmt struct {
	Pos
	Condition Expr
	Then      *BlockStmt
	Else      Stmt // *BlockStmt (else) or *IfStmt (else-if) or nil
}

// WhileStmt represents a while loop.
type WhileStmt struct {
	Pos
	Condition Expr
	Body      *BlockStmt
}

// ForStmt represents a for-in loop: for x in iterable { }.
type ForStmt struct {
	Pos
	VarName  string
	Iterable Expr
	Body     *BlockStmt
}

// ReturnStmt represents a return statement. Value is nil for bare return.
type ReturnStmt struct {
	Pos
	Value Expr
}

// BreakStmt represents a break statement inside a loop.
type BreakStmt struct {
	Pos
}

// ContinueStmt represents a continue statement inside a loop.
type ContinueStmt struct {
	Pos
}

// GuardStmt represents guard/against error handling.
type GuardStmt struct {
	Pos
	VarName   string     // variable that receives the result
	Expr      Expr       // expression that might throw
	ErrorName string     // variable that receives the error
	Against   *BlockStmt // error handling block
}

// TypeDeclStmt represents a type declaration: type Point = { x: int, y: int }.
type TypeDeclStmt struct {
	Pos
	Name       string
	Definition TypeDefExpr
}

// UseStmt represents an import: use helper from "./utils".
type UseStmt struct {
	Pos
	Names  []string // imported names; for "use { a, b } from"
	Star   bool     // true for "use * from"
	Alias  string   // "use X as Alias from"
	Source string   // the module path string
}

// ExportStmt represents an export: export helper.
type ExportStmt struct {
	Pos
	Stmt Stmt // the declaration being exported
}

// --- Interface marker methods ---

func (s *VarDeclStmt) nodeKind() string  { return "VarDeclStmt" }
func (s *AssignStmt) nodeKind() string   { return "AssignStmt" }
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

func (s *VarDeclStmt) stmtNode()  {}
func (s *AssignStmt) stmtNode()   {}
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
