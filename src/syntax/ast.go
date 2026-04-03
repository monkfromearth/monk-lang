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

// Program is the root AST node — a list of statements.
type Program struct {
	Stmts []Stmt
}

func (p *Program) nodeKind() string { return "Program" }

// --- Expressions ---

type NumberExpr struct {
	Value string // raw text: "42", "3.14", "0xFF"
	IsInt bool   // true for int, false for float
}

type StringExpr struct {
	Value string // contents without quotes
}

type TemplateExpr struct {
	Value string // contents without backticks
}

type BoolExpr struct {
	Value bool
}

type NoneExpr struct{}

type IdentExpr struct {
	Name string
}

type UnaryExpr struct {
	Op      TokenKind
	Operand Expr
}

type BinaryExpr struct {
	Left  Expr
	Op    TokenKind
	Right Expr
}

type AssignExpr struct {
	Target Expr      // IdentExpr, IndexExpr, or PropertyExpr
	Op     TokenKind // Equal, PlusEqual, MinusEqual, etc.
	Value  Expr
}

type CallExpr struct {
	Callee Expr
	Args   []Expr
}

type IndexExpr struct {
	Object Expr
	Index  Expr
}

type PropertyExpr struct {
	Object   Expr
	Property string
}

type ArrayExpr struct {
	Elements []Expr
}

type RecordExpr struct {
	Fields []RecordField
}

type RecordField struct {
	Key   string
	Value Expr
}

type FuncExpr struct {
	Params     []Param
	ReturnType TypeExpr // may be nil
	Body       *BlockStmt
}

type Param struct {
	Name    string
	Type    TypeExpr
	Default Expr // nil if no default
}

type TypeExpr struct {
	Name     string // "int", "string", custom type name
	IsArray  bool   // true for "int[]"
	Optional bool   // true for "int?"
}

type ThrowExpr struct {
	Value Expr
}

// --- Statements ---

type VarDeclStmt struct {
	Name    string
	Type    *TypeExpr // nil if no annotation
	Value   Expr
	IsConst bool
}

type ExprStmt struct {
	Expr Expr
}

type BlockStmt struct {
	Stmts []Stmt
}

type IfStmt struct {
	Condition Expr
	Then      *BlockStmt
	Else      Stmt // *BlockStmt or *IfStmt (else-if chain) or nil
}

type WhileStmt struct {
	Condition Expr
	Body      *BlockStmt
}

type ForStmt struct {
	VarName  string
	Iterable Expr
	Body     *BlockStmt
}

type ReturnStmt struct {
	Value Expr // nil for bare return
}

type BreakStmt struct{}
type ContinueStmt struct{}

type GuardStmt struct {
	VarName   string
	Expr      Expr
	ErrorName string
	Against   *BlockStmt
}

type TypeDeclStmt struct {
	Name       string
	Definition TypeDefExpr
}

type TypeDefExpr struct {
	AliasOf *TypeExpr     // for "type UserId = int"
	Fields  []RecordField // for "type Point = { x: int, y: int }" (uses key as name, value as type)
}

type UseStmt struct {
	Names  []string // imported names; nil for wildcard
	Star   bool     // true for "use * from"
	Alias  string   // "use X as Alias from"
	Source string   // the module path string
}

type ExportStmt struct {
	Stmt Stmt // the declaration being exported
}

// --- Marker methods (satisfy interfaces) ---

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
