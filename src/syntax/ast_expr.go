package syntax

// Expression nodes.

// NumberExpr represents an integer or float literal.
type NumberExpr struct {
	Pos
	Value string // raw text: "42", "3.14", "0xFF"
	IsInt bool   // true = int, false = float
}

// StringExpr represents a double-quoted string literal (contents without quotes).
type StringExpr struct {
	Pos
	Value string
}

// TemplateExpr represents a backtick template literal (contents without backticks).
type TemplateExpr struct {
	Pos
	Value string
}

// BoolExpr represents a boolean literal (true or false).
type BoolExpr struct {
	Pos
	Value bool
}

// NoneExpr represents the none literal.
type NoneExpr struct {
	Pos
}

// IdentExpr represents an identifier (variable name, function name, type name).
type IdentExpr struct {
	Pos
	Name string
}

// UnaryExpr represents a unary operation: -x, not x, !x, ~x.
type UnaryExpr struct {
	Pos
	Op      TokenKind
	Operand Expr
}

// BinaryExpr represents a binary operation: a + b, x == y, etc.
type BinaryExpr struct {
	Pos
	Left  Expr
	Op    TokenKind
	Right Expr
}

// CallExpr represents a function call: f(a, b).
type CallExpr struct {
	Pos
	Callee Expr
	Args   []Expr
}

// IndexExpr represents index access: arr[i], "hello"[0].
type IndexExpr struct {
	Pos
	Object Expr
	Index  Expr
}

// PropertyExpr represents property access: record.field.
type PropertyExpr struct {
	Pos
	Object   Expr
	Property string
}

// ArrayExpr represents an array literal: [1, 2, 3].
type ArrayExpr struct {
	Pos
	Elements []Expr
}

// RecordExpr represents a record literal: {name: "Alice", age: 30}.
type RecordExpr struct {
	Pos
	Fields []RecordField
}

// FuncExpr represents a function expression: (a int, b int) int { return a + b }.
type FuncExpr struct {
	Pos
	Params     []Param
	ReturnType TypeExpr
	Body       *BlockStmt
}

// ThrowExpr represents a throw expression: throw "error".
type ThrowExpr struct {
	Pos
	Value Expr
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
func (e *CallExpr) nodeKind() string     { return "CallExpr" }
func (e *IndexExpr) nodeKind() string    { return "IndexExpr" }
func (e *PropertyExpr) nodeKind() string { return "PropertyExpr" }
func (e *ArrayExpr) nodeKind() string    { return "ArrayExpr" }
func (e *RecordExpr) nodeKind() string   { return "RecordExpr" }
func (e *FuncExpr) nodeKind() string     { return "FuncExpr" }
func (e *ThrowExpr) nodeKind() string    { return "ThrowExpr" }

func (e *NumberExpr) exprNode()   {}
func (e *StringExpr) exprNode()   {}
func (e *TemplateExpr) exprNode() {}
func (e *BoolExpr) exprNode()     {}
func (e *NoneExpr) exprNode()     {}
func (e *IdentExpr) exprNode()    {}
func (e *UnaryExpr) exprNode()    {}
func (e *BinaryExpr) exprNode()   {}
func (e *CallExpr) exprNode()     {}
func (e *IndexExpr) exprNode()    {}
func (e *PropertyExpr) exprNode() {}
func (e *ArrayExpr) exprNode()    {}
func (e *RecordExpr) exprNode()   {}
func (e *FuncExpr) exprNode()     {}
func (e *ThrowExpr) exprNode()    {}
