package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFiles creates .monk files in a temp directory and returns the dir path.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, source := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// --- ResolvePath tests ---

func TestResolvePathBasic(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"utils.monk": `const X = 1`,
	})
	got, err := ResolvePath("./utils", filepath.Join(dir, "main.monk"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "utils.monk")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolvePathWithExtension(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"utils.monk": `const X = 1`,
	})
	// Explicit .monk extension should also work.
	got, err := ResolvePath("./utils.monk", filepath.Join(dir, "main.monk"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "utils.monk")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolvePathSubdirectory(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"lib/math.monk": `const PI = 3`,
	})
	got, err := ResolvePath("./lib/math", filepath.Join(dir, "main.monk"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "lib/math.monk")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolvePathMissingFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{})
	_, err := ResolvePath("./nope", filepath.Join(dir, "main.monk"))
	if err == nil {
		t.Fatal("expected error for missing module")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err)
	}
}

func TestResolvePathRejectsAbsolute(t *testing.T) {
	_, err := ResolvePath("utils", "/project/main.monk")
	if err == nil {
		t.Fatal("expected error for non-relative path")
	}
	if !strings.Contains(err.Error(), "must be relative") {
		t.Errorf("expected 'must be relative' in error, got: %s", err)
	}
}

// --- Build tests ---

func TestBuildSingleFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.monk": `let x = 42`,
	})
	g, err := Build(filepath.Join(dir, "main.monk"))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Order) != 1 {
		t.Errorf("expected 1 module, got %d", len(g.Order))
	}
}

func TestBuildSimpleImport(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.monk": `use add from "./math"
show(add(1, 2))`,
		"math.monk": `let add = (a int, b int) int { return a + b }
export add`,
	})
	g, err := Build(filepath.Join(dir, "main.monk"))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Order) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(g.Order))
	}
	// math.monk should come before main.monk (dependency first).
	if filepath.Base(g.Order[0]) != "math.monk" {
		t.Errorf("expected math.monk first, got %s", filepath.Base(g.Order[0]))
	}
	if filepath.Base(g.Order[1]) != "main.monk" {
		t.Errorf("expected main.monk second, got %s", filepath.Base(g.Order[1]))
	}
}

func TestBuildDiamondDependency(t *testing.T) {
	// main -> a, main -> b, a -> shared, b -> shared
	// shared should appear once and first.
	dir := writeFiles(t, map[string]string{
		"main.monk": `use fa from "./a"
use fb from "./b"`,
		"a.monk": `use x from "./shared"
let fa = () int { return x }
export fa`,
		"b.monk": `use x from "./shared"
let fb = () int { return x }
export fb`,
		"shared.monk": `let x = 42
export x`,
	})
	g, err := Build(filepath.Join(dir, "main.monk"))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Order) != 4 {
		t.Fatalf("expected 4 modules, got %d", len(g.Order))
	}
	// shared.monk must be first (leaf dependency).
	if filepath.Base(g.Order[0]) != "shared.monk" {
		t.Errorf("expected shared.monk first, got %s", filepath.Base(g.Order[0]))
	}
	// main.monk must be last (entry).
	if filepath.Base(g.Order[3]) != "main.monk" {
		t.Errorf("expected main.monk last, got %s", filepath.Base(g.Order[3]))
	}
}

func TestBuildCircularImport(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a.monk": `use x from "./b"`,
		"b.monk": `use y from "./a"`,
	})
	_, err := Build(filepath.Join(dir, "a.monk"))
	if err == nil {
		t.Fatal("expected circular import error")
	}
	if !strings.Contains(err.Error(), "circular import") {
		t.Errorf("expected 'circular import' in error, got: %s", err)
	}
}

func TestBuildMissingExport(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.monk": `use nope from "./lib"`,
		"lib.monk":  `let x = 1`,
	})
	_, err := Build(filepath.Join(dir, "main.monk"))
	if err == nil {
		t.Fatal("expected missing export error")
	}
	if !strings.Contains(err.Error(), "does not export") {
		t.Errorf("expected 'does not export' in error, got: %s", err)
	}
}

func TestBuildStarImport(t *testing.T) {
	// Star imports don't validate individual names — just resolve the module.
	dir := writeFiles(t, map[string]string{
		"main.monk": `use * from "./lib"`,
		"lib.monk": `let a = 1
export a
let b = 2
export b`,
	})
	g, err := Build(filepath.Join(dir, "main.monk"))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Order) != 2 {
		t.Errorf("expected 2 modules, got %d", len(g.Order))
	}
	// lib should export both a and b.
	lib := g.Modules[g.Order[0]]
	if !lib.Exports["a"] || !lib.Exports["b"] {
		t.Errorf("expected lib to export a and b, got %v", lib.Exports)
	}
}

func TestBuildMissingFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"main.monk": `use x from "./nope"`,
	})
	_, err := Build(filepath.Join(dir, "main.monk"))
	if err == nil {
		t.Fatal("expected error for missing module file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err)
	}
}

func TestCollectExportNames(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"lib.monk": `let helper = (x int) int { return x * 2 }
export helper
const PI = 3
export PI
type Point = { x: int, y: int }
export Point`,
	})
	g, err := Build(filepath.Join(dir, "lib.monk"))
	if err != nil {
		t.Fatal(err)
	}
	mod := g.Modules[g.Order[0]]
	for _, name := range []string{"helper", "PI", "Point"} {
		if !mod.Exports[name] {
			t.Errorf("expected %q to be exported", name)
		}
	}
}
