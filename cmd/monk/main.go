package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: monk <command> [arguments]")
		fmt.Fprintln(os.Stderr, "commands: build, run, check, lint, format")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		fmt.Println("monk build: not yet implemented")
	case "run":
		fmt.Println("monk run: not yet implemented")
	case "check":
		fmt.Println("monk check: not yet implemented")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
