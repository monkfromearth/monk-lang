package runtime

import (
	"fmt"

	"github.com/monkfromearth/monk-lang/src/ast"
	"github.com/monkfromearth/monk-lang/src/utils"
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

	value, exists := scope.GetSymbol(node.Symbol)

	if !exists {
		panic(fmt.Sprintf("Symbol `%s` does not exist", node.Symbol))
	}

	return value
}

func evaluateVariableDeclarationStatement(node ast.VariableDeclarationStatement, scope RuntimeScope) RuntimeValue {
	symbol := node.Symbol

	value := EvaluateAst(node.Value, scope)

	_, success := scope.DeclareSymbol(symbol, value, node.IsConstant)

	if !success {
		panic("Cannot declare a variable with the same name twice for variable `" + symbol + "`")
	}

	return value
}

func evaluateAssignmentExpression(node ast.AssignmentExpression, scope RuntimeScope) RuntimeValue {
	symbol := node.Symbol

	value := EvaluateAst(node.Value, scope)

	_, success := scope.AssignSymbol(symbol, value)

	if !success {
		panic("Cannot assign to a constant")
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

	case ast.BinaryExpression:
		return evaluateBinaryExpression(node, scope)

	case ast.IdentifierExpression:
		return evaluateIdentifierExpress(node, scope)

	case ast.VariableDeclarationStatement:
		return evaluateVariableDeclarationStatement(node, scope)

	case ast.AssignmentExpression:
		return evaluateAssignmentExpression(node, scope)

	default:
		panic("Unknown node type: " + utils.JSONStringify(node))
	}

}
