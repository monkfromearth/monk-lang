package ast

import (
	"github.com/monkfromearth/monk-lang/src/lexer"
)

func ParseStatement() interface{} {
	token := GetCurrentToken()

	switch token.Kind {
	case lexer.LetToken:
		return ParseVariableDeclarationStatement()
	case lexer.ConstToken:
		return ParseVariableDeclarationStatement()
	default:
		{
			result := ParseExpression()
			MoveToNextToken()
			return result
		}
	}
}

func ParseVariableDeclarationStatement() interface{} {
	token := GetCurrentToken()
	isConstant := token.Kind == lexer.ConstToken

	token = MoveNextWith(lexer.IdentifierToken, "Expected variable name after 'let' or 'const'")

	symbol := token.Value

	MoveNextWith(lexer.AssignmentToken, "Expected '=' after variable declaration")

	MoveToNextToken() // Skip the '='

	value := ParseExpression()

	statement := VariableDeclarationStatement{
		Statement: Statement{
			NodeType: VariableDeclarationStatementNode,
			NodeName: NodeTypeNames[VariableDeclarationStatementNode],
		},
		Symbol:     symbol,
		Value:      value,
		IsConstant: isConstant,
	}

	// if we have a newline, skip it and consider the next token as the start of the next statement
	// if file ends we consider the variable declared
	if !IsNextToken(lexer.EOFToken) {
		MoveNextWith(lexer.NewlineToken, "Expected newline after variable declaration") // skip newline
	} else {
		MoveToNextToken()
	}

	return statement
}
