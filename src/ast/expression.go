package ast

import (
	"fmt"
	"strconv"

	"github.com/monkfromearth/monk-lang/src/lexer"
	"github.com/monkfromearth/monk-lang/src/utils"
)

// Precedence:
// 1. PrimaryExpression
// 2. UnaryExpression
// 3. MultiplicativeExpression
// 4. AdditiveExpression
// 5. AssignmentExpression
func ParseExpression() interface{} {
	result := ParseAssignmentExpression()
	return result
}

// ParseAssignmentExpression parses an assignment expression
// e.g.: a = 1
func ParseAssignmentExpression() interface{} {
	fmt.Println("Left - Current Token in ParseAssignmentExpression")
	utils.PrettyPrint(GetCurrentToken())

	left := ParseAdditiveExpression()

	for IsNextToken(lexer.AssignmentToken) {
		MoveToNextToken() // To the `=` token

		MoveToNextToken() // Skip the `=` token

		right := ParseAdditiveExpression()

		var symbol string

		switch left := left.(type) {
		case IdentifierExpression:
			{
				symbol = left.Symbol
			}
		default:
			PanicWithDetails(GetCurrentToken(), "Expected identifier for assignment")
		}

		expression := AssignmentExpression{
			Expression: Expression{
				NodeType: AssignmentExpressionNode,
				NodeName: NodeTypeNames[AssignmentExpressionNode],
			},
			Symbol: symbol,
			Value:  right,
		}

		left = expression
	}

	return left
}

// ParseAdditiveExpression parses an additive expression
// e.g.: 1 + 2
// e.g.: (1 + 2) - 3
func ParseAdditiveExpression() interface{} {
	fmt.Println("Left - Current Token in ParseAdditiveExpression")
	utils.PrettyPrint(GetCurrentToken())

	left := ParseMultiplicativeExpression()

	for IsNextToken(lexer.PlusToken) || IsNextToken(lexer.MinusToken) {
		MoveToNextToken()

		fmt.Println("Operator - Current Token in ParseAdditiveExpression")
		utils.PrettyPrint(GetCurrentToken())

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

// ParseMultiplicativeExpression parses a multiplicative expression
// e.g. 1 * 2
// e.g. (1 * 2) * 3
func ParseMultiplicativeExpression() interface{} {
	fmt.Println("Left - Current Token in ParseMultiplicativeExpression")
	utils.PrettyPrint(GetCurrentToken())

	left := ParsePrimaryExpression()

	for IsNextToken(lexer.MultiplyToken) || IsNextToken(lexer.DivideToken) || IsNextToken(lexer.ModuloToken) {
		MoveToNextToken()

		fmt.Println("Operator - Current Token in ParseMultiplicativeExpression")
		utils.PrettyPrint(GetCurrentToken())

		token := GetCurrentToken()
		operator := token.Value

		MoveToNextToken()

		fmt.Println("Right - Current Token in ParseMultiplicativeExpression")
		utils.PrettyPrint(GetCurrentToken())

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

func ParsePrimaryExpression() interface{} {

	token := GetCurrentToken()
	fmt.Println("Current Token in ParsePrimaryExpression")
	utils.PrettyPrint(token)

	switch token.Kind {

	case lexer.NumberToken:
		{
			result := ParseNumericLiteral()
			return result
		}

	case lexer.IdentifierToken:
		{
			result := ParseIdentifier()
			return result
		}

	case lexer.OpenParenthesisToken:
		{
			// Skip the opening parenthesis
			MoveToNextToken()

			// Parse the expression inside the parenthesis
			expression := ParseExpression()

			fmt.Println("Expression Inside Parenthesis", utils.JSONStringify(expression))

			// Expect closing parenthesis
			MoveNextWith(lexer.CloseParenthesisToken, "Expected ')' after expression")

			return expression
		}
	default:
		fmt.Println("Unexpected token:", utils.JSONStringify(token))
		return nil
	}

}

// ParseNumericLiteral parses a numeric literal expression
// e.g. 42
func ParseNumericLiteral() NumericLiteralExpression {
	token := GetCurrentToken()

	value, err := strconv.Atoi(token.Value)
	if err != nil {
		PanicWithDetails(token, err.Error())
	}

	expression := NumericLiteralExpression{
		Expression: Expression{
			NodeType: NumericLiteralExpressionNode,
			NodeName: NodeTypeNames[NumericLiteralExpressionNode],
		},
		Value: value,
	}

	return expression
}

// ParseIdentifier parses an identifier expression
// e.g. foo
func ParseIdentifier() IdentifierExpression {
	token := GetCurrentToken()

	expression := IdentifierExpression{
		Expression: Expression{
			NodeType: IdentifierExpressionNode,
			NodeName: NodeTypeNames[IdentifierExpressionNode],
		},
		Symbol: token.Value,
	}

	return expression
}
