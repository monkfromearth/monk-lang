package ast

import (
	"fmt"
	"strconv"

	"github.com/monkfromearth/monk-lang/src/lexer"
	"github.com/monkfromearth/monk-lang/src/utils"
)

var CurrentTokens []lexer.Token = []lexer.Token{}

var CurrentIndex int = 0

func GetCurrentToken() lexer.Token {
	return CurrentTokens[CurrentIndex]
}

func GetNextToken() lexer.Token {
	return CurrentTokens[CurrentIndex+1]
}

func MoveToNextToken() {
	fmt.Println("Moved to Next Token")
	if CurrentIndex+1 >= len(CurrentTokens) {
		return
	}
	CurrentIndex++
}

func PanicWithDetails(token lexer.Token, message string) {
	panic(message + " (" + strconv.Itoa(token.Line) + ":" + strconv.Itoa(token.Column) + ")")
}

func MoveNextWith(kind lexer.TokenKind, message string) lexer.Token {
	token := GetNextToken()
	fmt.Println("Next token")
	utils.PrettyPrint(token)
	if token.Kind != kind {
		PanicWithDetails(token, message)
	}
	MoveToNextToken()
	return GetCurrentToken()
}

func IsNextToken(kind lexer.TokenKind) bool {
	token := GetNextToken()
	return token.Kind == kind
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
