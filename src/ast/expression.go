package ast

import (
	"fmt"
	"strconv"

	"github.com/monkfromearth/monk-lang/src/lexer"
)

func ParseExpression() interface{} {
	return ParseAdditiveExpression()
}

func ParsePrimaryExpression() interface{} {

	token := GetCurrentToken()

	switch token.Kind {

	case lexer.NumberToken:
		return ParseNumericLiteral()

	case lexer.IdentifierToken:
		return ParseIdentifier()

	case lexer.OpenParenthesisToken:
		{
			// Skip the opening parenthesis
			MoveToNextToken()

			// Parse the expression inside the parenthesis
			expression := ParseExpression()

			// Expect closing parenthesis
			MoveWithExpect(lexer.CloseParenthesisToken, "Expected ')' after expression")

			return expression
		}
	default:
		fmt.Println("Unexpected token:", token.Value)
		MoveToNextToken()
		return nil
	}

}

// ParseMultiplicativeExpression parses a multiplicative expression
// e.g. 1 * 2
// e.g. (1 * 2) * 3
func ParseMultiplicativeExpression() interface{} {
	left := ParsePrimaryExpression()

	for GetCurrentToken().Kind == lexer.MultiplyToken || GetCurrentToken().Kind == lexer.DivideToken || GetCurrentToken().Kind == lexer.ModuloToken {
		token := GetCurrentToken()
		operator := token.Value

		MoveToNextToken()

		right := ParsePrimaryExpression()

		expression := BinaryExpression{
			Expression: Expression{
				NodeType: BinaryExpressionNode,
				NodeName: NodeTypeNames[BinaryExpressionNode],
			},
			Operator: operator,
			Left:     left,
			Right:    right,
		}

		left = expression
	}

	return left
}

// ParseAdditiveExpression parses an additive expression
// e.g.: 1 + 2
// e.g.: (1 + 2) - 3
func ParseAdditiveExpression() interface{} {
	left := ParseMultiplicativeExpression()

	for GetCurrentToken().Kind == lexer.PlusToken || GetCurrentToken().Kind == lexer.MinusToken {
		token := GetCurrentToken()
		operator := token.Value

		MoveToNextToken()

		right := ParseMultiplicativeExpression()

		expression := BinaryExpression{
			Expression: Expression{
				NodeType: BinaryExpressionNode,
				NodeName: NodeTypeNames[BinaryExpressionNode],
			},
			Operator: operator,
			Left:     left,
			Right:    right,
		}

		left = expression
	}

	return left
}

// ParseNumericLiteral parses a numeric literal expression
// e.g. 42
func ParseNumericLiteral() NumericLiteralExpression {
	token := GetCurrentToken()

	value, err := strconv.Atoi(token.Value)
	if err != nil {
		panic(err)
	}

	expression := NumericLiteralExpression{
		Expression: Expression{
			NodeType: NumericLiteralNode,
			NodeName: NodeTypeNames[NumericLiteralNode],
		},
		Value: value,
	}

	MoveToNextToken()

	return expression
}

// ParseIdentifier parses an identifier expression
// e.g. foo
func ParseIdentifier() IdentifierExpression {
	token := GetCurrentToken()

	expression := IdentifierExpression{
		Expression: Expression{
			NodeType: IdentifierNode,
			NodeName: NodeTypeNames[IdentifierNode],
		},
		Name: token.Value,
	}

	MoveToNextToken()

	return expression
}
