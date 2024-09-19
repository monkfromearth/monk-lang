package main

import (
	"fmt"
	"os"

	"github.com/monkfromearth/monk-lang/src/ast"
	"github.com/monkfromearth/monk-lang/src/runtime"
	"github.com/monkfromearth/monk-lang/src/utils"
)

func main() {
	// Check if a file name was provided as a command-line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: monk run <filename>")
		return
	}

	// Get the file name from command-line arguments
	filename := os.Args[1]

	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	tree := ast.GenerateAst(string(content))

	utils.PrettyPrint(tree)

	globalScope := runtime.RuntimeScope{
		Symbols: map[string]runtime.RuntimeValue{
			"true": {
				Type:  runtime.BooleanValue,
				Name:  runtime.ValueNames[runtime.BooleanValue],
				Value: true,
			},
			"false": {
				Type:  runtime.BooleanValue,
				Name:  runtime.ValueNames[runtime.BooleanValue],
				Value: false,
			},
			"none": {
				Type:  runtime.NoneValue,
				Name:  runtime.ValueNames[runtime.NoneValue],
				Value: nil,
			},
		},
		Parent: nil,
		Constants: map[string]bool{
			"true":  true,
			"false": true,
			"none":  true,
		},
	}

	for _, statement := range tree.Statements {

		if statement == nil {
			continue
		}

		result := runtime.EvaluateAst(statement, globalScope)

		utils.PrettyPrint(result)
	}

	utils.PrettyPrint(globalScope)
}
