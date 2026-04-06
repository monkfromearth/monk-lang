// monk is the Monk Lang compiler CLI.
//
// Usage:
//
//	monk build <file.monk>              Compile to native binary (default: strips .monk)
//	monk build <file.monk> -o <out>     Output binary with custom name
//	monk build <file.monk> -o <out.c>   Output generated C source (no compilation)
//	monk run <file.monk>                Compile and run (temp binary, cleaned up)
//	monk check <file.monk>              Parse and validate without compiling
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/monkfromearth/monk-lang/codegen"
	"github.com/monkfromearth/monk-lang/syntax"
	"github.com/monkfromearth/monk-lang/types"
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
		os.Exit(cmdRun(os.Args[2:]))
	case "check":
		cmdCheck(os.Args[2:])
	case "version":
		fmt.Println("monk 0.0.1 — Buniyaad")
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
  help                            Show this help

The -o flag controls the output format:
  monk build hello.monk              Output: hello (binary)
  monk build hello.monk -o app      Output: app (binary)
  monk build hello.monk -o hello.c  Output: hello.c (C source, no compilation)`)
}

// cmdBuild compiles a .monk file to a native binary or emits C source.
func cmdBuild(args []string) {
	if len(args) < 1 {
		fatal("monk build: missing source file")
	}

	sourceFile := args[0]
	outputFile := ""

	// Parse -o flag
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" {
			if i+1 >= len(args) {
				fatal("monk build: -o requires an output path")
			}
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

	// Read and parse
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fatal("monk build: %s", err)
	}

	prog, err := syntax.Parse(string(source))
	if err != nil {
		fatal("%s: %s", sourceFile, err)
	}

	info, err := types.Check(prog)
	if err != nil {
		fatal("%s: %s", sourceFile, err)
	}

	// Generate C (with type info for scalar unboxing)
	cSource := codegen.GenerateWithTypes(prog, sourceFile, info)

	// Ensure the output directory exists (user may pass a nested path via -o)
	if outDir := filepath.Dir(outputFile); outDir != "" && outDir != "." {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			fatal("monk build: cannot create output directory %s: %s", outDir, err)
		}
	}

	// If output ends in .c, just emit the C source (no compilation)
	if strings.HasSuffix(outputFile, ".c") {
		if err := os.WriteFile(outputFile, []byte(cSource), 0644); err != nil {
			fatal("monk build: cannot write %s: %s", outputFile, err)
		}
		fmt.Fprintf(os.Stderr, "monk: wrote %s\n", outputFile)
		return
	}

	// Otherwise, compile to binary
	cFile := outputFile + ".c"
	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		fatal("monk build: cannot write %s: %s", cFile, err)
	}

	runtimeDir := findRuntime()

	ccArgs := []string{"-std=c11", "-O3", "-flto", "-I" + runtimeDir, cFile}
	for _, src := range runtimeSources {
		ccArgs = append(ccArgs, filepath.Join(runtimeDir, src))
	}
	ccArgs = append(ccArgs, "-lm", "-o", outputFile)
	cmd := exec.Command("cc", ccArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(cFile)
		fatal("monk build: C compilation failed")
	}

	_ = os.Remove(cFile)
	fmt.Fprintf(os.Stderr, "monk: built %s\n", outputFile)
}

// cmdRun compiles and runs a .monk file, then cleans up. Returns the exit
// code to propagate. Returning (instead of calling os.Exit directly) lets the
// deferred temp-dir cleanup run even when the child program exits non-zero.
func cmdRun(args []string) int {
	if len(args) < 1 {
		fatal("monk run: missing source file")
	}

	sourceFile := args[0]

	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fatal("monk run: %s", err)
	}

	prog, err := syntax.Parse(string(source))
	if err != nil {
		fatal("%s: %s", sourceFile, err)
	}

	info, err := types.Check(prog)
	if err != nil {
		fatal("%s: %s", sourceFile, err)
	}

	cSource := codegen.GenerateWithTypes(prog, sourceFile, info)

	dir, err := os.MkdirTemp("", "monk-run-*")
	if err != nil {
		fatal("monk run: %s", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cFile := filepath.Join(dir, "program.c")
	binFile := filepath.Join(dir, "program")

	if err := os.WriteFile(cFile, []byte(cSource), 0644); err != nil {
		fatal("monk run: %s", err)
	}

	runtimeDir := findRuntime()

	ccArgs := []string{"-std=c11", "-O3", "-flto", "-I" + runtimeDir, cFile}
	for _, src := range runtimeSources {
		ccArgs = append(ccArgs, filepath.Join(runtimeDir, src))
	}
	ccArgs = append(ccArgs, "-lm", "-o", binFile)
	compile := exec.Command("cc", ccArgs...)
	compile.Stderr = os.Stderr
	if err := compile.Run(); err != nil {
		fatal("monk run: compilation failed")
	}

	run := exec.Command(binFile, args[1:]...)
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	run.Stdin = os.Stdin
	if err := run.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fatal("monk run: %s", err)
	}
	return 0
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

	prog, err := syntax.Parse(string(source))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", sourceFile, err)
		os.Exit(1)
	}

	if _, err := types.Check(prog); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", sourceFile, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "monk: %s ok\n", sourceFile)
}

// findRuntime locates the runtime/ directory.
// Search order: relative to executable, working directory, MONK_RUNTIME_DIR env,
// and finally extract embedded runtime files to a cache directory.
func findRuntime() string {
	// Try relative to executable. Resolve symlinks first so that Homebrew-style
	// installs (where /opt/homebrew/bin/monk is a symlink into the Cellar) find
	// the runtime next to the real binary, not next to the symlink.
	exe, err := os.Executable()
	if err == nil {
		if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
			exe = resolved
		}
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

	// Try MONK_RUNTIME_DIR env var (validate it contains runtime.h)
	if dir := os.Getenv("MONK_RUNTIME_DIR"); dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "runtime.h")); err == nil {
			return dir
		}
	}

	// Fall back to embedded runtime — extract to ~/.cache/monk/runtime/
	return extractEmbeddedRuntime()
}

// extractEmbeddedRuntime writes the embedded runtime.h and runtime.c to a
// cache directory so cc can find them. Files are only written if missing or
// if the binary is newer than the cached files.
func extractEmbeddedRuntime() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fatal("monk: cannot determine home directory: %s", err)
	}

	cacheDir := filepath.Join(home, ".cache", "monk", "runtime")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fatal("monk: cannot create cache directory: %s", err)
	}

	for _, f := range embeddedRuntimeFiles {
		writeIfChanged(filepath.Join(cacheDir, f.name), f.content)
	}

	return cacheDir
}

// embeddedRuntimeSourceFiles returns the list of .c files that make up the
// runtime (matches the files in runtime/). Keep in sync with embed.go and
// with the runtimeSources slice used at compile time.
var runtimeSources = []string{
	"value.c",
	"arith.c",
	"string.c",
	"container.c",
	"math.c",
	"builtins.c",
	"error.c",
}

// embeddedRuntimeFiles pairs each embedded runtime file with its on-disk
// name under ~/.cache/monk/runtime/.
var embeddedRuntimeFiles = []struct {
	name    string
	content []byte
}{
	{"runtime.h", embeddedRuntimeH},
	{"internal.h", embeddedInternalH},
	{"value.c", embeddedValueC},
	{"arith.c", embeddedArithC},
	{"string.c", embeddedStringC},
	{"container.c", embeddedContainerC},
	{"math.c", embeddedMathC},
	{"builtins.c", embeddedBuiltinsC},
	{"error.c", embeddedErrorC},
}

// writeIfChanged writes content to path only if the file is missing or differs.
// This avoids gratuitous mtime bumps and, more importantly, prevents concurrent
// `monk build` invocations from racing on the same cache file (one reader could
// otherwise observe a half-written header while cc is compiling).
func writeIfChanged(path string, content []byte) {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, content) {
		return
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		fatal("monk: cannot write %s: %s", path, err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
