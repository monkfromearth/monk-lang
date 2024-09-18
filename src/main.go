package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/monkfromearth/monk-lang/src/ast"
	"github.com/monkfromearth/monk-lang/src/runtime"
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

	data, err := json.MarshalIndent(tree, "", "  ")

	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(data))

	globalScope := runtime.RuntimeScope{
		Symbols: map[string]runtime.RuntimeValue{
			"universe": {
				Type:     runtime.NumberValue,
				Name:     runtime.ValueNames[runtime.NumberValue],
				Value:    42,
				Constant: true,
			},
			"true": {
				Type:     runtime.BooleanValue,
				Name:     runtime.ValueNames[runtime.BooleanValue],
				Value:    true,
				Constant: true,
			},
			"false": {
				Type:     runtime.BooleanValue,
				Name:     runtime.ValueNames[runtime.BooleanValue],
				Value:    false,
				Constant: true,
			},
			"none": {
				Type:     runtime.NoneValue,
				Name:     runtime.ValueNames[runtime.NoneValue],
				Value:    nil,
				Constant: true,
			},
		},
		Parent:    nil,
		Constants: make(map[string]runtime.RuntimeValue),
	}

	for _, statement := range tree.Statements {

		result := runtime.EvaluateAst(statement, globalScope)

		data, err = json.MarshalIndent(result, "", "  ")

		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			return
		}

		fmt.Println(string(data))
	}
}
