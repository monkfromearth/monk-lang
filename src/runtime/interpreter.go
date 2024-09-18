package runtime

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/src/ast"
)

func evaluateBinaryExpression(node ast.BinaryExpression, scope RuntimeScope) RuntimeValue {
	left := EvaluateAst(node.Left, scope)
	right := EvaluateAst(node.Right, scope)

	if left.Type != NumberValue || right.Type != NumberValue {
		panic("Binary expression requires both operands to be numbers")
	}

	leftValue := left.Value.(int)
	rightValue := right.Value.(int)

	switch node.Operator {
	case "+":
		{
			return RuntimeValue{
				Type:  NumberValue,
				Name:  "Number",
				Value: leftValue + rightValue,
			}
		}
	case "-":
		{
			return RuntimeValue{
				Type:  NumberValue,
				Name:  "Number",
				Value: leftValue - rightValue,
			}
		}
	case "*":
		{
			return RuntimeValue{
				Type:  NumberValue,
				Name:  "Number",
				Value: leftValue * rightValue,
			}
		}
	case "/":
		{
			return RuntimeValue{
				Type:  NumberValue,
				Name:  "Number",
				Value: leftValue / rightValue,
			}
		}
	case "%":
		{
			return RuntimeValue{
				Type:  NumberValue,
				Name:  "Number",
				Value: leftValue % rightValue,
			}
		}
	}
	panic(fmt.Sprintf("Unknown operator: %s", node.Operator))
}

func evaluateIdentifierExpress(node ast.IdentifierExpression, scope RuntimeScope) RuntimeValue {

	value, exists := scope.GetSymbol(node.Name)

	if !exists {
		panic(fmt.Sprintf("Symbol `%s` does not exist", node.Name))
	}

	return value
}

func EvaluateAst(tree interface{}, scope RuntimeScope) RuntimeValue {

	switch node := tree.(type) {

	case ast.NumericLiteralExpression:
		{
			return RuntimeValue{
				Type:  NumberValue,
				Name:  "Number",
				Value: node.Value,
			}
		}

	case ast.NoneLiteralExpression:
		{
			return RuntimeValue{
				Type:  NoneValue,
				Name:  "None",
				Value: nil,
			}
		}

	case ast.BinaryExpression:
		return evaluateBinaryExpression(node, scope)

	case ast.IdentifierExpression:
		return evaluateIdentifierExpress(node, scope)

	default:
		panic(fmt.Sprintf("Unknown node type: %T", node))
	}

}
