package ast

import "github.com/monkfromearth/monk-lang/src/lexer"

type Node struct {
	Token lexer.Token `json:"token"`
	Value interface{} `json:"value"`
}

const (
	ProgramNode = iota
	IdentifierNode
	NumericLiteralNode
	BinaryExpressionNode
	NoneLiteralNode
)

var NodeTypeNames = map[int]string{
	ProgramNode:          "ProgramNode",
	IdentifierNode:       "IdentifierNode",
	NumericLiteralNode:   "NumericLiteralNode",
	BinaryExpressionNode: "BinaryExpressionNode",
	NoneLiteralNode:      "NoneLiteralNode",
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

// Expression is a generic expression node that can be used as a base for more specific expression types
type Expression struct {
	NodeType int    `json:"nodeType"`
	NodeName string `json:"nodeName"`
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
	Name string `json:"name"`
}

// NumericLiteral is a numeric literal node that represents a numeric value
// e.g. 42
type NumericLiteralExpression struct {
	Expression
	Value int `json:"value"`
}

// NoneLiteral is a none literal node that represents the absence of a value
// e.g. none
type NoneLiteralExpression struct {
	Expression
}
