export interface BuiltinInfo {
  signature: string;
  description: string;
  returns: string;
}

export const BUILTINS: Record<string, BuiltinInfo> = {
  // Output
  show: {
    signature: "show(value) -> none",
    description: "Print a value to stdout followed by a newline.",
    returns: "none",
  },

  // Conversion
  to_string: {
    signature: "to_string(value) -> string",
    description:
      "Convert any value to its string representation. Ints, floats, booleans, arrays, and records all have defined formats.",
    returns: "string",
  },
  to_int: {
    signature: "to_int(value string) -> int",
    description:
      "Strict conversion: string must be a valid integer. Throws on invalid input. Does NOT truncate floats — use floor() for that.",
    returns: "int",
  },
  to_float: {
    signature: "to_float(value string) -> float",
    description: "Convert a string to a float. Throws on invalid input.",
    returns: "float",
  },

  // Type checking
  typeof: {
    signature: "typeof(value) -> string",
    description:
      'Returns the type name as a string: "int", "float", "string", "boolean", "none", "array", "record", "function".',
    returns: "string",
  },
  is_number: {
    signature: "is_number(value) -> boolean",
    description: "True if the value is an int or float.",
    returns: "boolean",
  },
  is_string: {
    signature: "is_string(value) -> boolean",
    description: "True if the value is a string.",
    returns: "boolean",
  },
  is_boolean: {
    signature: "is_boolean(value) -> boolean",
    description: "True if the value is a boolean.",
    returns: "boolean",
  },
  is_array: {
    signature: "is_array(value) -> boolean",
    description: "True if the value is an array.",
    returns: "boolean",
  },
  is_record: {
    signature: "is_record(value) -> boolean",
    description: "True if the value is a record.",
    returns: "boolean",
  },
  is_function: {
    signature: "is_function(value) -> boolean",
    description: "True if the value is a function.",
    returns: "boolean",
  },
  is_none: {
    signature: "is_none(value) -> boolean",
    description: "True if the value is none.",
    returns: "boolean",
  },

  // String operations
  length: {
    signature: "length(value) -> int",
    description:
      "Returns the length of a string (Unicode codepoints) or array (element count).",
    returns: "int",
  },
  substring: {
    signature: "substring(s string, start int, end int) -> string",
    description:
      "Extract a substring. Indices are clamped to bounds. Returns empty string if start >= end.",
    returns: "string",
  },
  index_of: {
    signature: "index_of(s string, search string) -> int",
    description:
      "Find the first occurrence of search in s. Returns -1 if not found.",
    returns: "int",
  },
  split: {
    signature: "split(s string, delimiter string) -> string[]",
    description: "Split a string by delimiter. Returns an array of strings.",
    returns: "string[]",
  },
  trim: {
    signature: "trim(s string) -> string",
    description: "Remove leading and trailing whitespace.",
    returns: "string",
  },
  to_upper_case: {
    signature: "to_upper_case(s string) -> string",
    description: "Convert string to uppercase.",
    returns: "string",
  },
  to_lower_case: {
    signature: "to_lower_case(s string) -> string",
    description: "Convert string to lowercase.",
    returns: "string",
  },

  // Array operations
  append: {
    signature: "append(arr, element) -> array",
    description:
      "Return a new array with element added at the end. Original is unchanged (value semantics).",
    returns: "array",
  },
  prepend: {
    signature: "prepend(arr, element) -> array",
    description: "Return a new array with element added at the beginning.",
    returns: "array",
  },
  pop: {
    signature: "pop(arr) -> array",
    description:
      "Return a new array with the last element removed. Returns empty array if already empty.",
    returns: "array",
  },
  drop: {
    signature: "drop(arr, n int) -> array",
    description: "Return a new array with the first n elements removed.",
    returns: "array",
  },
  take: {
    signature: "take(arr, n int) -> array",
    description: "Return a new array with only the first n elements.",
    returns: "array",
  },
  slice: {
    signature: "slice(arr, start int, end int) -> array",
    description:
      "Return a new array from start (inclusive) to end (exclusive). Indices clamped.",
    returns: "array",
  },
  range: {
    signature: "range(n int) -> int[]",
    description:
      "Return an array [0, 1, 2, ..., n-1]. Returns empty array if n <= 0.",
    returns: "int[]",
  },

  // Math
  abs: {
    signature: "abs(n) -> int | float",
    description: "Absolute value. Works on int and float.",
    returns: "int | float",
  },
  floor: {
    signature: "floor(n float) -> int",
    description: "Round down to nearest integer.",
    returns: "int",
  },
  ceil: {
    signature: "ceil(n float) -> int",
    description: "Round up to nearest integer.",
    returns: "int",
  },
  round: {
    signature: "round(n float) -> int",
    description: "Round to nearest integer (0.5 rounds up).",
    returns: "int",
  },
  sqrt: {
    signature: "sqrt(n) -> float",
    description: "Square root. Throws on negative input.",
    returns: "float",
  },
  pow: {
    signature: "pow(base, exp) -> float",
    description: "Raise base to the power of exp.",
    returns: "float",
  },
  log: {
    signature: "log(n) -> float",
    description: "Natural logarithm (base e).",
    returns: "float",
  },
  log10: {
    signature: "log10(n) -> float",
    description: "Base-10 logarithm.",
    returns: "float",
  },
  exp: {
    signature: "exp(n) -> float",
    description: "e raised to the power of n.",
    returns: "float",
  },
  min: {
    signature: "min(a, b) -> int | float",
    description: "Return the smaller of two numbers.",
    returns: "int | float",
  },
  max: {
    signature: "max(a, b) -> int | float",
    description: "Return the larger of two numbers.",
    returns: "int | float",
  },
  sin: {
    signature: "sin(n float) -> float",
    description: "Sine (radians).",
    returns: "float",
  },
  cos: {
    signature: "cos(n float) -> float",
    description: "Cosine (radians).",
    returns: "float",
  },
  tan: {
    signature: "tan(n float) -> float",
    description: "Tangent (radians).",
    returns: "float",
  },
  asin: {
    signature: "asin(n float) -> float",
    description: "Inverse sine. Returns radians.",
    returns: "float",
  },
  acos: {
    signature: "acos(n float) -> float",
    description: "Inverse cosine. Returns radians.",
    returns: "float",
  },
  atan: {
    signature: "atan(n float) -> float",
    description: "Inverse tangent. Returns radians.",
    returns: "float",
  },

  // File I/O
  file_read: {
    signature: "file_read(path string) -> string",
    description:
      "Read entire file contents as a string. Throws if file does not exist.",
    returns: "string",
  },
  file_write: {
    signature: "file_write(path string, content string) -> none",
    description:
      "Write content to a file. Creates the file if it does not exist, overwrites if it does.",
    returns: "none",
  },
  file_exists: {
    signature: "file_exists(path string) -> boolean",
    description: "Returns true if the file exists at the given path.",
    returns: "boolean",
  },

  // Environment
  env_get: {
    signature: "env_get(name string) -> string",
    description:
      'Read an environment variable. Returns empty string if not set.',
    returns: "string",
  },
  exit: {
    signature: "exit(code int) -> none",
    description: "Exit the program with the given exit code.",
    returns: "none",
  },
  args: {
    signature: "args() -> string[]",
    description:
      "Returns command-line arguments as an array of strings (excludes the binary name).",
    returns: "string[]",
  },
};

export const KEYWORDS = [
  "let",
  "const",
  "if",
  "else",
  "for",
  "in",
  "while",
  "break",
  "continue",
  "return",
  "guard",
  "against",
  "throw",
  "type",
  "use",
  "export",
  "from",
  "as",
  "and",
  "or",
  "not",
  "is",
  "true",
  "false",
  "none",
];

export const TYPES = ["int", "float", "string", "boolean", "none"];
