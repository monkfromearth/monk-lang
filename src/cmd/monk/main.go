// monk is the Monk Lang compiler CLI.
//
// Usage:
//
//	monk build <file.monk>           Compile to native binary
//	monk build <file.monk> -o <out>  Compile with custom output name
//	monk run <file.monk>             Compile and run (temp binary, cleaned up)
//	monk check <file.monk>           Parse and validate without compiling
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/monkfromearth/monk-lang/src/codegen"
	"github.com/monkfromearth/monk-lang/src/syntax"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		cmdBuild(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	case "check":
		cmdCheck(os.Args[2:])
	case "version":
		fmt.Println("monk 0.0.1-dev")
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "monk: unknown command '%s'\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: monk <command> [arguments]

Commands:
  build <file.monk> [-o output]   Compile to native binary
  run <file.monk>                 Compile and run
  check <file.monk>               Parse and validate
  version                         Print version
  help                            Show this help`)
}

// cmdBuild compiles a .monk file to a native binary.
func cmdBuild(args []string) {
	if len(args) < 1 {
		fatal("monk build: missing source file")
	}

	sourceFile := args[0]
	outputFile := ""

	// Parse -o flag
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
			i++
		}
	}

	// Default output: strip .monk extension
	if outputFile == "" {
		outputFile = strings.TrimSuffix(sourceFile, ".monk")
		if outputFile == sourceFile {
			outputFile = sourceFile + ".out"
		}
	}

	// Read source
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fatal("monk build: %s", err)
	}

	// Parse
	prog, err := syntax.Parse(string(source))
	if err != nil {
		fatal("%s: %s", sourceFile, err)
	}

	// Generate C
	cSource := codegen.Generate(prog, sourceFile)

	// Write .c file
	cFile := outputFile + ".c"
	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		fatal("monk build: cannot write %s: %s", cFile, err)
	}

	// Find runtime
	runtimeDir := findRuntime()

	// Compile with cc
	cmd := exec.Command("cc", "-std=c11", "-O2",
		"-I"+runtimeDir,
		cFile,
		filepath.Join(runtimeDir, "runtime.c"),
		"-lm",
		"-o", outputFile,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(cFile) // clean up on failure
		fatal("monk build: C compilation failed")
	}

	// Clean up .c file
	os.Remove(cFile)

	fmt.Fprintf(os.Stderr, "monk: built %s\n", outputFile)
}

// cmdRun compiles and runs a .monk file, then cleans up the binary.
func cmdRun(args []string) {
	if len(args) < 1 {
		fatal("monk run: missing source file")
	}

	sourceFile := args[0]

	// Read source
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fatal("monk run: %s", err)
	}

	// Parse
	prog, err := syntax.Parse(string(source))
	if err != nil {
		fatal("%s: %s", sourceFile, err)
	}

	// Generate C
	cSource := codegen.Generate(prog, sourceFile)

	// Write temp files
	dir, err := os.MkdirTemp("", "monk-run-*")
	if err != nil {
		fatal("monk run: %s", err)
	}
	defer os.RemoveAll(dir)

	cFile := filepath.Join(dir, "program.c")
	binFile := filepath.Join(dir, "program")

	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		fatal("monk run: %s", err)
	}

	// Find runtime
	runtimeDir := findRuntime()

	// Compile
	compile := exec.Command("cc", "-std=c11", "-O2",
		"-I"+runtimeDir,
		cFile,
		filepath.Join(runtimeDir, "runtime.c"),
		"-lm",
		"-o", binFile,
	)
	compile.Stderr = os.Stderr
	if err := compile.Run(); err != nil {
		fatal("monk run: compilation failed")
	}

	// Run the binary, passing through remaining args
	run := exec.Command(binFile, args[1:]...)
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	run.Stdin = os.Stdin
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fatal("monk run: %s", err)
	}
}

// cmdCheck parses and validates a .monk file without compiling.
func cmdCheck(args []string) {
	if len(args) < 1 {
		fatal("monk check: missing source file")
	}

	sourceFile := args[0]

	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fatal("monk check: %s", err)
	}

	_, err = syntax.Parse(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", sourceFile, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "monk: %s ok\n", sourceFile)
}

// findRuntime locates the runtime/ directory.
// Looks relative to the executable, then falls back to common paths.
func findRuntime() string {
	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(dir, "runtime"),
			filepath.Join(dir, "..", "runtime"),
			filepath.Join(dir, "..", "..", "runtime"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(c, "runtime.h")); err == nil {
				return c
			}
		}
	}

	// Try relative to working directory
	if _, err := os.Stat("runtime/runtime.h"); err == nil {
		return "runtime"
	}

	// Try MONK_RUNTIME_DIR env var
	if dir := os.Getenv("MONK_RUNTIME_DIR"); dir != "" {
		return dir
	}

	fatal("monk: cannot find runtime/ directory. Set MONK_RUNTIME_DIR or run from project root.")
	return ""
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
