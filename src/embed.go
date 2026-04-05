package main

import _ "embed"

//go:embed runtime/runtime.h
var embeddedRuntimeH []byte

//go:embed runtime/runtime.c
var embeddedRuntimeC []byte
