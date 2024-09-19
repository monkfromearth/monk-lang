package ast

import "github.com/monkfromearth/monk-lang/src/lexer"

type Node struct {
	Token lexer.Token `json:"token"`
	Value interface{} `json:"value"`
}

const (
	ProgramNode = iota
	IdentifierExpressionNode
	NumericLiteralExpressionNode
	UnaryExpressionNode
	BinaryExpressionNode
	AssignmentExpressionNode
	VariableDeclarationStatementNode
)

var NodeTypeNames = map[int]string{
	ProgramNode:                      "ProgramNode",
	IdentifierExpressionNode:         "IdentifierExpressionNode",
	UnaryExpressionNode:              "UnaryExpressionNode",
	NumericLiteralExpressionNode:     "NumericLiteralExpressionNode",
	BinaryExpressionNode:             "BinaryExpressionNode",
	AssignmentExpressionNode:         "AssignmentExpressionNode",
	VariableDeclarationStatementNode: "VariableDeclarationStatementNode",
}

type Program struct {
	NodeType   int           `json:"nodeType"`
	NodeName   string        `json:"nodeName"`
	Statements []interface{} `json:"statements"`
}

// Statement is a generic statement node
type Statement struct {
	NodeType int    `json:"nodeType"`
	NodeName string `json:"nodeName"`
}

type VariableDeclarationStatement struct {
	Statement
	Symbol     string      `json:"symbol"`
	Value      interface{} `json:"value"`
	IsConstant bool        `json:"isConstant"`
}

// Expression is a generic expression node that can be used as a base for more specific expression types
type Expression struct {
	NodeType int    `json:"nodeType"`
	NodeName string `json:"nodeName"`
}

// AssignmentExpression is an assignment expression node that can be used to assign a value to a variable
// e.g. foo = 42
type AssignmentExpression struct {
	Expression
	Symbol string      `json:"symbol"`
	Value  interface{} `json:"value"`
}

// UnaryExpression is a unary expression node that can be used to represent unary operations
// e.g. -42 - Unary Minus (-): This operation negates the value of a number, changing its sign
// e.g. !foo - Boolean Not (!): This operation inverts the boolean value of its operand
// e.g. &foo - Address-of (&): This operation returns the memory address of its operand
// e.g. *foo - Dereference (*): This operation accesses the value stored at a pointer's address
type UnaryExpression struct {
	Expression
	Operator string      `json:"operator"`
	Right    interface{} `json:"right"`
}

// BinaryExpression is a binary expression node that can be used to represent arithmetic operations and comparisons
// e.g. 1 + 2
type BinaryExpression struct {
	Expression
	Operator string      `json:"operator"`
	Left     interface{} `json:"left"`
	Right    interface{} `json:"right"`
}

// Identifier is an identifier node that represents a variable or function name
// e.g. foo
type IdentifierExpression struct {
	Expression
	Symbol string `json:"symbol"`
}

// NumericLiteral is a numeric literal node that represents a numeric value
// e.g. 42
type NumericLiteralExpression struct {
	Expression
	Value int `json:"value"`
}
