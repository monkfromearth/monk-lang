package main

import _ "embed"

// Embedded runtime files. When `monk` is installed somewhere without an
// adjacent runtime/ directory, these bytes are extracted to
// ~/.cache/monk/runtime/ and cc compiles from there.
//
// Keep this list in sync with runtimeFiles in main.go.

//go:embed runtime/runtime.h
var embeddedRuntimeH []byte

//go:embed runtime/internal.h
var embeddedInternalH []byte

//go:embed runtime/value.c
var embeddedValueC []byte

//go:embed runtime/arith.c
var embeddedArithC []byte

//go:embed runtime/string.c
var embeddedStringC []byte

//go:embed runtime/container.c
var embeddedContainerC []byte

//go:embed runtime/math.c
var embeddedMathC []byte

//go:embed runtime/builtins.c
var embeddedBuiltinsC []byte

//go:embed runtime/error.c
var embeddedErrorC []byte
