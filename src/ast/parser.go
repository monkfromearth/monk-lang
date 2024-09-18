package ast

import (
	"github.com/monkfromearth/monk-lang/src/lexer"
)

var CurrentTokens []lexer.Token = []lexer.Token{}

var CurrentIndex int = 0

func GetCurrentToken() lexer.Token {
	return CurrentTokens[CurrentIndex]
}

func MoveToNextToken() {
	CurrentIndex++
}

func MoveWithExpect(kind lexer.TokenKind, message string) {
	token := GetCurrentToken()
	if token.Kind != kind {
		panic(message)
	}
	MoveToNextToken()
}

func IsNotEOF() bool {
	if CurrentIndex+1 >= len(CurrentTokens) {
		return false
	}
	return CurrentTokens[CurrentIndex].Kind != lexer.EOFToken
}

// GenerateAst generates an abstract syntax tree (AST) from the input string
// and returns a Program struct containing the AST
// Precedence is based on the order of precedence in the language specification
// 1. PrimaryExpression
// 2. UnaryExpression
// 3. MultiplicativeExpression
// 4. AdditiveExpression
func GenerateAst(input string) Program {
	CurrentTokens = lexer.Tokenize(input)

	program := Program{
		NodeType:   ProgramNode,
		NodeName:   NodeTypeNames[ProgramNode],
		Statements: []interface{}{},
	}

	for IsNotEOF() {
		statement := ParseStatement()
		program.Statements = append(program.Statements, statement)
	}

	return program
}
