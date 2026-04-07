package main

import "embed"

// embeddedRuntime contains the entire runtime/ directory embedded at compile
// time. When monk is installed without an adjacent runtime/ directory, these
// files are extracted to ~/.cache/monk/runtime/ by extractEmbeddedRuntime.
//
//go:embed runtime
var embeddedRuntime embed.FS
