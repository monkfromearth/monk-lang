package codegen

import (
	"fmt"
	"strings"

	"github.com/monkfromearth/monk-lang/syntax"
)

// Small utilities + the Monk-builtin → C-runtime name map.

// emitLine writes a formatted line into g.body. Statement emitters use this
// for everything except manual string-building.
func (g *generator) emitLine(format string, args ...any) {
	fmt.Fprintf(&g.body, format, args...)
}

// mangleName prefixes Monk variable names to avoid C keyword collisions.
func mangleName(name string) string {
	return "mk_" + name
}

// cString returns a C string literal with proper escaping. Used both for
// user-provided string values AND for the filename in #line directives
// (which is why it escapes backslashes + CR — Windows paths and CRLF).
func cString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return "\"" + s + "\""
}

// compoundToArith maps the compound-assign operators to their monk_* runtime
// arithmetic functions, used by the boxed-path emitAssign.
// Fix 3: panic on unknown compound operator instead of silent fallback.
func compoundToArith(op syntax.TokenKind) string {
	switch op {
	case syntax.PlusEqual:
		return "monk_add"
	case syntax.MinusEqual:
		return "monk_sub"
	case syntax.StarEqual:
		return "monk_mul"
	case syntax.SlashEqual:
		return "monk_div"
	case syntax.PercentEqual:
		return "monk_mod"
	default:
		panic(fmt.Sprintf("codegen: unhandled compound operator: %s", op))
	}
}

// builtinMap maps Monk builtin names to C runtime function names.
var builtinMap = map[string]string{
	"show":          "monk_show",
	"to_string":     "monk_to_string",
	"to_int":        "monk_to_int",
	"to_float":      "monk_to_float",
	"length":        "monk_length",
	"substring":     "monk_substring",
	"index_of":      "monk_index_of",
	"split":         "monk_split",
	"trim":          "monk_trim",
	"to_upper_case": "monk_to_upper_case",
	"to_lower_case": "monk_to_lower_case",
	"append":        "monk_append",
	"prepend":       "monk_prepend",
	"pop":           "monk_pop",
	"drop":          "monk_drop",
	"take":          "monk_take",
	"slice":         "monk_slice",
	"range":         "monk_range",
	"abs":           "monk_abs",
	"floor":         "monk_floor",
	"ceil":          "monk_ceil",
	"round":         "monk_round",
	"sqrt":          "monk_sqrt",
	"pow":           "monk_pow",
	"log":           "monk_log",
	"log10":         "monk_log10",
	"exp":           "monk_exp",
	"min":           "monk_min",
	"max":           "monk_max",
	"sin":           "monk_sin",
	"cos":           "monk_cos",
	"tan":           "monk_tan",
	"asin":          "monk_asin",
	"acos":          "monk_acos",
	"atan":          "monk_atan",
	"typeof":        "monk_typeof",
	"is_number":     "monk_is_number",
	"is_string":     "monk_is_string",
	"is_boolean":    "monk_is_boolean",
	"is_array":      "monk_is_array",
	"is_record":     "monk_is_record",
	"is_function":   "monk_is_function",
	"is_none":       "monk_is_none",
	"file_read":     "monk_file_read",
	"file_write":    "monk_file_write",
	"file_exists":   "monk_file_exists",
	"env_get":       "monk_env_get",
	"exit":          "monk_exit",
	"map":           "monk_map",
	"filter":        "monk_filter",
	"reduce":        "monk_reduce",
}
