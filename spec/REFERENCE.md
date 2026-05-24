# Monk Lang Reference

> Complete syntax and semantics specification.
> Single source of truth for both the implementation and LLMs.
> If the code disagrees with this document, one of them has a bug.

---

## Table of Contents

- [Design Philosophy](#design-philosophy)
- [Comments](#comments)
- [Data Types](#data-types)
- [Variables and Constants](#variables-and-constants)
- [Operators](#operators)
- [Functions](#functions)
- [Control Flow](#control-flow)
- [Arrays](#arrays)
- [Records](#records)
- [Error Handling](#error-handling)
- [Type System](#type-system)
- [Module System](#module-system)
- [Template Literals](#template-literals)
- [Built-in Functions](#built-in-functions)
- [Keywords](#keywords)
- [Operator Precedence](#operator-precedence)
- [Edge Cases and Special Behaviors](#edge-cases-and-special-behaviors)

---

## Design Philosophy

Five principles govern every design decision in Monk.

### 1. Explicit Over Implicit

No hidden behavior. If a type changes, you wrote the conversion. If an error occurs, you wrote the handler. If a variable is mutable, you wrote `let`.

- No implicit type coercion. Use `to_string()`, `to_int()`, `floor()`.
- No exceptions. Errors are explicit via `guard`/`against`/`throw`.
- No garbage collection. Memory model is deterministic (TBD — see Memory Model notes).
- `const` means fully frozen — no reassignment, no element mutation.

**One deliberate exception:** truthiness. `false`, `none`, and `0` are falsy in boolean contexts (`if`, `while`, `and`, `or`, `not`). Everything else is truthy. This is the only implicit conversion in the language.

### Consistency Test

Every edge case should be resolvable by asking: "Is this a read or an operation?" and "Is this explicit or implicit?" If a proposed behavior violates these rules, the spec has a bug.

### 2. Graceful on Reads, Strict on Operations

When you **ask for data** that might not be there, Monk degrades gracefully — it returns `none` or `[]` instead of crashing. When you **operate on invalid values**, Monk stops you immediately with an error.

| Situation | Behavior | Why |
|-----------|----------|-----|
| `arr[100]` on 3-element array | `none` | Reading — data might not exist |
| `config.missing` on untyped record | `none` | Reading — field might not exist |
| `range(-5)` | `[]` | Selection — nothing to select |
| `take([1], 5)` | `[1]` | Selection — return what you can |
| `pop([])` | `[]` | Selection — nothing to remove |
| `none + 1` | Error | Operation — invalid operands |
| `10 / 0` | Error | Operation — mathematically undefined |
| `sqrt(-1)` | Error | Operation — mathematically undefined |
| `5 < "hello"` | Error | Operation — incompatible types |

The distinction: **reading is exploration, operating is commitment.**

### 3. Values by Default, References by Intent

Monk should make aliasing explicit. Plain variables and plain function parameters behave like values. Shared mutation and pointer-like behavior exist in the language, but they are spelled with `ref`.

```monk
let increment = (counter ref int) none {
    counter = counter + 1
}

let total = 0
increment(ref total)
show(total)   // 1
```

The rule is: values are the default, references are explicit. This keeps ordinary code easy to reason about, while still allowing pointers where they are genuinely needed.

### 4. Local Reasoning Over Magic

Monk should reward the reader who stays close to the code in front of them. The meaning of a line should come from the line itself, not from hidden state, surprising coercions, or far-away special cases.

- Prefer explicit code paths over implicit behavior.
- Prefer local readability over cleverness.
- Keep special cases visible and rare.

### 5. Small Core, Strong Library

Monk should stay small at the syntax level and explicit at the capability level. If a feature can live cleanly in the runtime or builtin library, prefer that over adding more syntax.

- The core language should be easy to learn and easy to scan.
- Powerful behavior should be exposed through explicit builtins, types, and modules.
- The language should be honest about cost: if something allocates, copies, or checks at runtime, that should not be hidden from the reader.

### Additional Principles

- Functions are expressions assigned to variables. No `func` keyword.
- Mandatory type annotations for function parameters and return types.
- No semicolons. Newlines are insignificant — expressions continue across lines when unambiguous (e.g., line ends with an operator or open bracket).
- Immutable by default (`const`). Explicit mutability (`let`).
- First-assignment type inference for variables.
- Single-line comments only (`//`).
- Script-style execution — top-level statements run directly, no `main` function required.
- Assignment in conditions is forbidden — `if x = 5` is an error, preventing the `=` vs `==` bug.

---

## Comments

```monk
// Single-line comment
let x = 42  // Inline comment
```

Multi-line comments are not supported.

---

## Data Types

### Primitive Types

| Type      | Examples                          | Notes                        |
|-----------|-----------------------------------|------------------------------|
| `int`     | `42`, `-99`, `0`                  | 64-bit signed integer        |
| `float`   | `3.14`, `-0.5`, `1.23e5`, `1e-3` | 64-bit floating-point        |
| `string`  | `"Hello"`, `""`                   | Double-quoted, escape seqs   |
| `boolean` | `true`, `false`                   | Boolean values               |
| `none`    | `none`                            | Null/empty value             |

### Numeric Literals

```monk
// Decimal
let x = 42
let y = 3.14
let z = 1.23e5

// Hex, binary, octal
let hex = 0xFF
let bin = 0b1010
let oct = 0o77

// Underscore separators (readability)
let big = 1_000_000
let hex_big = 0xFF_FF
```

### Escape Sequences (Strings)

| Sequence | Meaning   |
|----------|-----------|
| `\n`     | Newline   |
| `\t`     | Tab       |
| `\"`     | Quote     |
| `\\`     | Backslash |

### Collection Types

| Type     | Example                              |
|----------|--------------------------------------|
| `array`  | `[1, 2, 3]`, `["a", "b"]`, `[]`     |
| `record` | `{name: "Alice", age: 30}`, `{}`    |

Primitives are copied by value. Arrays, records, and closures follow normal Monk value rules unless they are intentionally shared through `ref`.

---

## Variables and Constants

### Declaration

```monk
// Immutable (preferred) — variable AND contents are frozen
const name = "Alice"
const pi = 3.14159
const nums = [1, 2, 3]
// nums[0] = 99    // Error: cannot mutate const

// Mutable — variable AND contents can change
let counter = 0
let active = true
let data = [1, 2, 3]
data[0] = 99       // OK

// With explicit type annotation (space-separated, NOT colon)
let count int = 42
const age int = 25
let score float = 95.5
let message string = "Hello"
let is_valid boolean = true
```

### Mutability Rules

`const` means **deeply frozen** — cannot reassign the variable, and cannot mutate elements or fields:

```monk
const x = 100
// x = 200            // Error: cannot reassign const

const arr = [1, 2, 3]
// arr = [4, 5, 6]    // Error: cannot reassign const
// arr[0] = 99        // Error: cannot mutate const

const person = {name: "Alice", age: 30}
// person.name = "Bob" // Error: cannot mutate const
```

`let` means **fully mutable** — can reassign and mutate:

```monk
let y = 100
y = 200               // OK

let arr = [1, 2, 3]
arr[0] = 99           // OK
arr = [4, 5, 6]       // OK

// Type must be preserved on reassignment
let n = 42
n = 24                // OK: int to int
// n = "text"         // Error: cannot assign string to int variable
```

### First-Assignment Type Inference

Variables without explicit type annotations are locked to the type of their first assigned value.

```monk
let counter = 0        // Inferred: int
let message = "Hello"  // Inferred: string
let active = true      // Inferred: boolean

counter = 5            // OK
// counter = "invalid" // Error: string assigned to int variable
```

---

## Operators

### Arithmetic

| Op  | Meaning        | Notes                                                    |
|-----|----------------|----------------------------------------------------------|
| `+` | Addition       | Numeric only. Use `to_string()` for string concatenation |
| `-` | Subtraction    | Also unary negation                                      |
| `*` | Multiplication |                                                          |
| `/` | Division       | int/int = int (truncating). float involved = float       |
| `%` | Modulo         | Remainder                                                |

Arithmetic operators only accept numeric operands (`int` or `float`). Using them on strings, booleans, none, or collections is a type error.

### Division Semantics

```monk
10 / 3        // 3 (int / int = int, truncates toward zero)
10.0 / 3      // 3.333... (float / int = float)
10 / 3.0      // 3.333... (int / float = float)
10 / 2        // 5 (int, exact result)
7 % 3         // 1 (int)
```

### Comparison

| Op         | Alt     | Meaning                |
|------------|---------|------------------------|
| `==`       | `is`    | Equal to               |
| `!=`       |         | Not equal to           |
| `<`        |         | Less than              |
| `>`        |         | Greater than           |
| `<=`       |         | Less than or equal     |
| `>=`       |         | Greater than or equal  |

`is` is a synonym for `==`. There is no `is not` operator — use `!=`.

**Equality (`==`, `!=`):** works on primitives only. Comparing arrays, records, or functions with `==` is a type error.

**Ordering (`<`, `>`, `<=`, `>=`):** works on numbers and strings. Strings use lexicographic (Unicode code point) ordering. Comparing different types is a type error. `none` cannot be ordered — `none < 5` is a type error.

```monk
5 == 5          // true
5 == "5"        // type error: cannot compare int and string
"abc" < "abd"   // true (lexicographic)
"A" < "a"       // true (uppercase before lowercase in Unicode)
// [1, 2] == [1, 2]  // type error: cannot compare arrays
// none < 5          // type error: cannot order none
```

### Logical

| Op    | Alt  | Meaning     | Notes          |
|-------|------|-------------|----------------|
| `and` | `&&` | Logical AND | Short-circuit  |
| `or`  | `||` | Logical OR  | Short-circuit  |
| `not` | `!`  | Logical NOT | Unary          |

Logical operators accept **any type** and use truthiness rules. They always return `boolean`.

**Truthiness:** `false`, `none`, and `0` are falsy. Everything else is truthy.

```monk
not 0           // true (0 is falsy)
not none        // true (none is falsy)
not ""          // false ("" is truthy)
not []          // false ([] is truthy)
0 or 5          // true (0 is falsy, 5 is truthy)
"a" and "b"     // true (both truthy)
none and 5      // false (none is falsy, short-circuits)
```

### Bitwise

| Op   | Meaning       | Notes                           |
|------|---------------|---------------------------------|
| `&`  | Bitwise AND   | Integer operands only           |
| `\|` | Bitwise OR    | Integer operands only           |
| `^`  | Bitwise XOR   | Integer operands only           |
| `~`  | Bitwise NOT   | Unary, integer operand only     |
| `<<` | Left shift    | Integer operands only           |
| `>>` | Right shift   | Arithmetic right shift          |

Integers are 64-bit signed, two's complement.

### Assignment

| Op   | Meaning             |
|------|---------------------|
| `=`  | Assign              |
| `+=` | Add and assign      |
| `-=` | Subtract and assign |
| `*=` | Multiply and assign |
| `/=` | Divide and assign   |
| `%=` | Modulo and assign   |

Assignment is a **statement**, not an expression. It cannot appear inside conditions, function arguments, or other expressions. `if x = 5 { }` is a syntax error.

### String Concatenation

The `+` operator does NOT auto-coerce types. Use `to_string()` for concatenation:

```monk
"Hello " + "World"            // "Hello World" (string + string is OK)
"Number: " + to_string(42)    // "Number: 42"
"Bool: " + to_string(true)    // "Bool: true"
// "Number: " + 42            // Error: cannot add string and int
```

---

## Functions

Functions are expressions. They are assigned to variables, not declared with a keyword.

### Syntax

```monk
let name = (param1 type1, param2 type2) return_type {
    // body
    return value
}
```

- Type annotations on parameters: **mandatory**.
- Return type: **mandatory**.
- Block body with `{ }`: **mandatory**. No shorthand.
- Every code path must return a value matching the declared return type. Missing a return is a **compile error**.
- Default parameter values are supported for trailing parameters.

### Examples

```monk
// No parameters
let greet = () string {
    return "Hello, World!"
}

// With parameters
let add = (a int, b int) int {
    return a + b
}

// Immutable function (recommended)
const square = (x int) int {
    return x * x
}

// Mutable function variable (can be reassigned)
let op = (x int, y int) int {
    return x + y
}
op = (x int, y int) int {
    return x * y
}

// Optional return type
let safe_divide = (a int, b int) int? {
    if b == 0 {
        return none
    }
    return a / b
}

// Void function (returns none)
let log = (msg string) none {
    show(msg)
}

// Default parameter values (trailing parameters only)
let greet = (name string, greeting string = "Hello") string {
    return greeting + ", " + name
}
greet("Alice")          // "Hello, Alice"
greet("Alice", "Hi")    // "Hi, Alice"
```

### Function Arguments

Plain arguments are values. A function cannot modify the caller's binding unless the parameter is marked `ref`:

```monk
let process = (arr int[]) none {
    arr[0] = 99     // modifies the LOCAL value
}
let data = [1, 2, 3]
process(data)
show(data[0])       // 1 (unchanged)
```

To modify caller state directly, use `ref` on both sides:

```monk
let increment = (counter ref int) none {
    counter = counter + 1
}

let total = 0
increment(ref total)
show(total)   // 1
```

Without `ref`, return the new value:

```monk
let process = (arr int[]) int[] {
    arr[0] = 99
    return arr
}
let data = [1, 2, 3]
data = process(data)
show(data[0])       // 99
```

### Function Type Signatures

Function types are written as `(param_types) -> return_type`. There is no generic `function` type — all function references must have a known signature.

```monk
// The compiler infers add's type as (int, int) -> int from the definition.
let add = (a int, b int) int {
    return a + b
}

// When accepting a function as a parameter, declare the expected signature:
let apply = (fn (int, int) -> int, a int, b int) int {
    return fn(a, b)
}
apply(add, 10, 5)  // 15

// In type definitions:
type BinaryOp = (int, int) -> int
type Predicate = (int) -> boolean

let op BinaryOp = add  // Checked: add matches (int, int) -> int
```

### Higher-Order Functions

```monk
// Returns a function
let create_multiplier = (factor int) (int) -> int {
    return (x int) int {
        return x * factor
    }
}
let triple = create_multiplier(3)
show(triple(4))  // 12

// Accepts a function
let apply = (fn (int, int) -> int, a int, b int) int {
    return fn(a, b)
}
```

### Closures

Closures capture plain values by default. Shared state must be explicit, following the same `ref` rule as function parameters.

```monk
let create_counter = (initial int) () -> int {
    let count = initial       // closure gets its own local count
    return () int {
        count = count + 1
        return count
    }
}

let c = create_counter(10)
show(c())  // 11
show(c())  // 12
```

Closures do not implicitly share outer mutable state:

```monk
let x = 0
let increment = () none {
    x = x + 1       // modifies the closure's local capture
}
increment()
show(x)              // 0 (outer x is unchanged)
```

To share mutable state across scopes, the shared access must be explicit via `ref`.

### Recursion

Functions can reference themselves by name inside their own body:

```monk
let fibonacci = (n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n - 1) + fibonacci(n - 2)
}

let factorial_tail = (n int, acc int) int {
    if n <= 1 {
        return acc
    }
    return factorial_tail(n - 1, acc * n)
}
```

---

## Control Flow

### If / Else

```monk
if condition {
    // then
}

if condition {
    // then
} else {
    // else
}

if condition1 {
    // ...
} else if condition2 {
    // ...
} else {
    // ...
}
```

Braces are always required. No parentheses around the condition. Conditions use truthiness rules (`false`, `none`, `0` are falsy).

### While Loops

```monk
let count = 5
while count > 0 {
    show(count)
    count = count - 1
}
```

### For Loops

`for...in` iterates over arrays and strings. Numbers are not iterable — use `range()`.

The loop variable is `const` — it cannot be reassigned inside the loop body.

```monk
// Array iteration
for x in [10, 20, 30] {
    show(x)  // 10, 20, 30
    // x = 99  // Error: loop variable is const
}

// Range iteration (0 to n-1)
for i in range(5) {
    show(i)  // 0, 1, 2, 3, 4
}

// String iteration (characters)
for char in "hello" {
    show(char)  // "h", "e", "l", "l", "o"
}
```

### Break and Continue

```monk
for i in range(10) {
    if i == 5 {
        break       // Exit loop
    }
    if i % 2 == 0 {
        continue    // Skip to next iteration
    }
    show(i)
}
```

`break` and `continue` can be used inside loops, including inside `against` blocks that are within a loop.

### Block Statements

Blocks create new scopes:

```monk
let x = 10
{
    let y = 20
    show(x + y)  // 30
}
// y is not accessible here
```

---

## Arrays

### Literals

All arrays must be homogeneous — every element must be the same type:

```monk
let numbers = [1, 2, 3, 4, 5]       // Inferred: int[]
let strings = ["apple", "banana"]     // Inferred: string[]
let nested = [[1, 2], [3, 4]]        // Inferred: int[][]
let empty = []                        // Type inferred from context

// [1, "hello", true]                // Error: array elements must be the same type
// Use a record for mixed data: {id: 1, name: "hello", active: true}
```

Trailing commas are allowed:

```monk
let items = [
    "apple",
    "banana",
    "cherry",
]
```

### Empty Array Type Inference

Empty arrays `[]` infer their element type from context:

```monk
let nums int[] = []      // int[] from annotation
let items = []
items = append(items, 1) // inferred as int[] from first element
```

### Typed Arrays

```monk
let nums int[] = [1, 2, 3]
let words string[] = ["hello", "world"]
let flags boolean[] = [true, false]
let sparse int?[] = [1, none, 3]
```

Typed arrays reject elements of the wrong type.

### Indexing

Zero-based. Out-of-bounds returns `none` (graceful on reads):

```monk
let arr = [10, 20, 30, 40, 50]
show(arr[0])     // 10
show(arr[4])     // 50
show(arr[10])    // none (out of bounds)
show(arr[-1])    // none (negative index)

let i = 2
show(arr[i])     // 30 (expression as index)
```

### String Indexing

Strings can be indexed like arrays. Returns a single-character string. Out-of-bounds returns `none`:

```monk
show("hello"[0])    // "h"
show("hello"[4])    // "o"
show("hello"[10])   // none
```

### Index Assignment

Only on `let` arrays. `const` arrays cannot be mutated:

```monk
let arr = [10, 20, 30]
arr[0] = 99
show(arr)  // [99, 20, 30]

// Typed arrays enforce element type on assignment
let nums int[] = [1, 2, 3]
nums[0] = 42       // OK
// nums[0] = "hello"  // Error: cannot assign string to int[] element

// Out-of-bounds write is an error (strict on operations)
// arr[10] = 99     // Error: index 10 out of bounds
```

### Array Functions

Array helper functions like `append`, `prepend`, `take`, `drop`, and `slice` return new arrays. Element assignment still mutates an existing `let` array. Out-of-bounds parameters clamp gracefully:

```monk
let fruits = ["apple", "banana", "cherry"]

// Add elements
append(fruits, "date")       // ["apple", "banana", "cherry", "date"]
prepend(fruits, "orange")    // ["orange", "apple", "banana", "cherry"]

// Remove elements
pop(fruits)                  // ["apple", "banana"] (remove last)
pop([])                      // [] (nothing to remove)
drop(fruits)                 // ["banana", "cherry"] (remove first, default n=1)
drop(fruits, 2)              // ["cherry"] (remove first 2)
drop(fruits, 100)            // [] (clamps to length)

// Select elements
take(fruits)                 // ["apple"] (first element, default n=1)
take(fruits, 2)              // ["apple", "banana"] (first 2 elements)
take(fruits, 100)            // ["apple", "banana", "cherry"] (clamps)
slice(fruits, 1, 3)          // ["banana", "cherry"] (from index 1 to 3)
slice(fruits, 0, 100)        // ["apple", "banana", "cherry"] (clamps)

// Higher-order
map([1, 2, 3], (x int) int { return x * 2 })           // [2, 4, 6]
filter([1, 2, 3, 4], (x int) boolean { return x > 2 }) // [3, 4]
reduce([1, 2, 3], (acc int, x int) int { return acc + x }, 0)  // 6
reduce([], (acc int, x int) int { return acc + x }, 0)          // 0 (returns initial)

// Generate
range(5)   // [0, 1, 2, 3, 4]
range(0)   // [] (empty)
range(-5)  // [] (empty)
```

---

## Records

### Literals

```monk
let person = {name: "Alice", age: 30}

let config = {
    debug: true,
    port: 8080,
    name: "MyApp",
}
```

Trailing commas are allowed. `let empty = {}` creates an empty untyped record.

### Property Access

Dot notation only:

```monk
show(person.name)   // "Alice"
show(person.age)    // 30
```

### Property Assignment

Only on `let` records. `const` records cannot be mutated:

```monk
let person = {name: "Alice", age: 30}
person.age = 31
show(person.age)  // 31

// Typed records enforce field types and reject unknown fields
type Person = { name: string, age: int }
let user Person = { name: "Bob", age: 25 }
user.age = 30          // OK
// user.age = "thirty"  // Error: cannot assign string to int field
// user.email = "x"     // Error: type Person has no field 'email'
```

### Missing Property Behavior

Depends on whether the record is typed (graceful on reads, strict on operations):

```monk
// TYPED record — error on unknown property (caught at compile time)
type Person = { name: string, age: int }
let user Person = { name: "Alice", age: 30 }
// show(user.email)  // Error: type Person has no property 'email'

// UNTYPED record — none on unknown property READ
let config = { debug: true, port: 8080 }
show(config.missing)  // none (graceful on reads)

// UNTYPED record — error on unknown property WRITE
// config.host = "localhost"  // Error: config has no field 'host'

// Records have fixed shape from creation. Want more fields? Create a new record:
let full_config = { debug: config.debug, port: config.port, host: "localhost" }
```

### Nested Records

```monk
let address = {street: "123 Main", city: "SF"}
let company = {
    name: "TechCorp",
    address: address,
}
show(company.address.city)  // "SF"
```

### Records with Functions

```monk
let calc = {
    add: (a int, b int) int { return a + b },
    mul: (a int, b int) int { return a * b },
}
show(calc.add(5, 3))  // 8
```

### Typed Records

```monk
type Point = { x: int, y: int }
type Person = { name: string, age: int, active: boolean }

let origin Point = { x: 0, y: 0 }
let user Person = { name: "Bob", age: 30, active: true }
```

Structural typing: records are validated by shape, not by name.

---

## Error Handling

Monk uses `guard`/`against`/`throw` instead of exceptions. This is a try-catch mechanism.

### Throw

Any value can be thrown. An unhandled `throw` at the top level terminates the program with an error message.

```monk
let divide = (a int, b int) int {
    if b == 0 {
        throw "Division by zero"
    }
    return a / b
}
```

### Guard / Against

```monk
guard result = divide(10, 0) against error {
    show("Error: " + to_string(error))
    result = 0
}
show(result)  // 0
```

Semantics:
- `guard` declares the variable in the **enclosing scope**.
- The expression after `=` is evaluated.
- If it succeeds, the variable gets the result. The `against` block is skipped.
- If it throws, the `against` block runs with the error bound to the named variable.
- If the `against` block does not assign the variable, it defaults to `none`.
- `guard` with a non-throwing expression is valid — the `against` block is simply dead code.
- `break`, `continue`, and `return` work normally inside `against` blocks.

```monk
// Success: result = 5
guard result = divide(10, 2) against error {
    result = 0
}
show(result)  // 5

// Error, with assignment: result = 0
guard result = divide(10, 0) against error {
    result = 0
}
show(result)  // 0

// Error, no assignment: result = none
guard result = divide(10, 0) against error {
    show("Something went wrong")
}
show(result)  // none

// Guard inside a loop
for i in range(10) {
    guard val = might_fail(i) against err {
        continue    // skip this iteration
    }
    show(val)
}
```

### Error Propagation

Unhandled errors propagate up the call stack:

```monk
let outer = () string {
    let r = divide(10, 0)  // Throws, propagates out of outer()
    return "unreachable"
}

guard result = outer() against err {
    show("Caught: " + to_string(err))
    result = "fallback"
}
```

### Multiple Guards

```monk
let process = (x int, y int, z int) int {
    guard div = divide(x, y) against err {
        show("Division failed: " + to_string(err))
        div = 1
    }
    guard age = validate_age(z) against err {
        show("Invalid age: " + to_string(err))
        age = 0
    }
    return div + age
}
```

---

## Type System

### Type Annotations

Space-separated (not colon):

```monk
let count int = 42
const name string = "Alice"
let active boolean = true
let score float = 95.5
```

### Array Type Annotations

```monk
let numbers int[] = [1, 2, 3]
let words string[] = ["hello", "world"]
```

### Function Type Annotations

Parameters and return type are mandatory on function definitions. Function type signatures use `(param_types) -> return_type`:

```monk
let add = (a int, b int) int {
    return a + b
}

// Function type in parameter position
let apply = (fn (int, int) -> int, a int, b int) int {
    return fn(a, b)
}

// Function type in type definitions
type Predicate = (int) -> boolean
type Transform = (string) -> string
```

### Custom Type Definitions

```monk
// Type aliases (structural — UserId and int are interchangeable)
type UserId = int
type Email = string

let id UserId = 12345
let age int = id          // OK: both are int

// Record types
type Point = { x: int, y: int }
type Rectangle = { width: int, height: int, position: Point }

let origin Point = { x: 0, y: 0 }
```

### Reference Types

`ref T` means a reference to a mutable location holding `T`. References are explicit in both the parameter list and the call site.

```monk
let swap = (a ref int, b ref int) none {
    let temp = a
    a = b
    b = temp
}

let x = 10
let y = 20
swap(ref x, ref y)
show(x)   // 20
show(y)   // 10
```

Rules:

- `ref` parameters may only be passed assignable `let` locations
- temporaries, literals, and `const` values cannot be passed as `ref`
- plain assignment and plain parameters remain value-based
- first implementation scope: `ref` is valid in function parameter types and `ref x` call-site arguments
- first-class stored references outside parameter passing are deferred until the core `ref` model is stable

### Optional Types

The `?` suffix allows a type to also hold `none`. Nesting is invalid — `int??` is a compile error.

```monk
let maybe int? = 42
maybe = none          // OK

let name string? = none
name = "Alice"        // OK

// Optional array elements
let sparse int?[] = [1, none, 3, none, 5]

// Functions returning optional
let safe_div = (a int, b int) int? {
    if b == 0 { return none }
    return a / b
}

// int?? is NOT valid — one level of optional only
```

### Numeric Widening

`int` widens to `float` implicitly (no data loss). `float` to `int` requires explicit conversion:

```monk
let f float = 42        // OK: int widens to float
// let n int = 3.14     // Error: cannot assign float to int
let n int = floor(3.14) // OK: explicit conversion via floor/ceil/round

let calc = (a float, b float) float { return a + b }
calc(1, 2)               // OK: int args widen to float
```

### Type Inference

Variables without annotations get their type from the first assignment and are locked to it.

### Element Type Enforcement

Typed arrays and records validate their contents:

```monk
let nums int[] = [1, 2, 3]
// nums = ["a", "b"]  // Error: cannot assign string[] to int[]
```

### Structural Typing

Records are compared by shape, not by name. Type aliases are structural — they are interchangeable with their underlying type:

```monk
let a = { name: "Alice", age: 30 }
a = { name: "Bob", age: 25 }         // OK: same shape
// a = { firstName: "Bob", age: 25 } // Error: different shape

type UserId = int
type Age = int
let id UserId = 5
let a Age = id    // OK: both are int
```

### Definite Assignment Analysis

Variables cannot be read before they are assigned. Every code path in a non-none function must return a value matching the declared return type. Missing a return is a compile error.

### `typeof` Return Values

`typeof` always returns a base type name string:

```monk
typeof(42)              // "int"
typeof(3.14)            // "float"
typeof("hello")         // "string"
typeof(true)            // "boolean"
typeof(none)            // "none"
typeof([1, 2, 3])       // "array"
typeof({a: 1})          // "record"
typeof(some_function)   // "function"
```

---

## Memory Management

Monk uses deterministic memory management. No garbage collection. Plain code stays value-oriented, and explicit `ref` introduces shared access when needed.

- Primitives are copied by value.
- Plain assignment and plain function parameters follow value semantics.
- `ref` parameters and native handles allow explicit pointer/reference behavior.
- Heap values are managed deterministically by the runtime.
- `const` freezes the value and cannot be passed as `ref`.

The implementation may use reference counting, copy-on-write, or specialized backing stores internally. Those are implementation choices. The language-level rule is simpler: aliasing must be explicit in source.

See `spec/MEMORY_MODEL_DISCUSSION.md` for design history and tradeoffs.

---

## Module System

### Imports

```monk
use function_name from "./module"
use { func1, func2 } from "./utils"
use * from "./all"

// With alias
use long_name as short from "./module"
```

### Exports

```monk
let helper = (x int) int { return x * 2 }
export helper

const PI = 3.14159
export PI

type Point = { x: int, y: int }
export Point
```

Module files use the `.monk` extension. Paths are relative to the importing file. Circular imports are forbidden — a compile error.

Module-level code executes once, when the module is first imported.

---

## C Foreign Function Interface (FFI)

> **Status: PLANNED.** Monk will grow a C FFI, but the surface syntax is still undecided.
>
> **What is decided so far:**
> - Monk should be able to declare and call external C functions
> - the compiler should be able to emit C headers and linker inputs for those bindings
> - Monk's explicit `ref` / pointer model should be the foundation for C pointer-style parameters
> - opaque native handles should be representable in Monk
>
> **Still open:**
> - the exact surface syntax for declaring C bindings
> - how headers and libraries are written in source
> - which pointer/reference forms are exposed directly versus wrapped
> - how callbacks and record marshalling should work
> - how much of the raw C shape should be visible in Monk signatures
>
> **Recommended v1 boundary, regardless of syntax:**
> - direct scalars: `int`, `float`, `boolean`
> - strings as UTF-8 C strings
> - opaque native handles
> - pointer-style parameters expressed with Monk's explicit reference model
> - no callbacks or record marshalling in v1
>
> **Validation targets:**
> - `libm`
> - `sqlite3`
> - `zlib`

---

## Template Literals

Backtick strings. No interpolation. Same type as regular strings. Supports multiline:

```monk
let greeting = `Hello, World!`
let with_quotes = `Can contain "double" and 'single' quotes`

let multiline = `first line
second line
third line`

// Concatenation via +
let name = "Alice"
let msg = `Hello ` + name + `!`

typeof(greeting)      // "string"
is_string(greeting)   // true
```

---

## Built-in Functions

### Output and Conversion

| Function    | Signature                   | Description                                                |
|-------------|-----------------------------|------------------------------------------------------------|
| `show`      | `(value: any) -> none`     | Print to stdout (see output format below)                  |
| `to_string` | `(value: any) -> string`   | Convert any value to string representation                 |
| `to_int`    | `(value) -> int`           | Convert a number or parse a string to int. Accepts `int` (pass-through), `float` (truncates toward zero), or `string` (strict — throws on "3.14" or non-numeric). |
| `to_float`  | `(value) -> float`         | Convert a number or parse a string to float. Accepts `int` (widens), `float` (pass-through), or `string` (throws on non-numeric). |

#### Output Format

`show` and `to_string` produce these representations:

```monk
show(42)                    // 42
show(3.14)                  // 3.14
show(true)                  // true
show(false)                 // false
show(none)                  // none
show("hello")               // hello (no quotes)
show([1, 2, 3])             // [1, 2, 3]
show(["a", "b"])            // ["a", "b"] (strings quoted inside collections)
show({name: "Alice"})       // {name: "Alice"}
show(some_function)         // <function>
```

`to_string` returns the same representation as a string value. `show` prints it followed by a newline.

### Math

| Function | Signature                              | Description           |
|----------|----------------------------------------|-----------------------|
| `abs`    | `(x: number) -> number`               | Absolute value        |
| `floor`  | `(x: number) -> int`                  | Round down            |
| `ceil`   | `(x: number) -> int`                  | Round up              |
| `round`  | `(x: number) -> int`                  | Round to nearest      |
| `sqrt`   | `(x: number) -> float`                | Square root. Throws if x < 0. |
| `pow`    | `(base: number, exp: number) -> float` | Power                |
| `log`    | `(x: number) -> float`                | Natural log. Throws if x <= 0. |
| `log10`  | `(x: number) -> float`                | Base-10 log. Throws if x <= 0. |
| `exp`    | `(x: number) -> float`                | e^x                   |
| `min`    | `(a: number, b: number) -> number`    | Minimum of two values |
| `max`    | `(a: number, b: number) -> number`    | Maximum of two values |

### Trigonometry

| Function | Signature               | Description      |
|----------|-------------------------|------------------|
| `sin`    | `(x: number) -> float` | Sine (radians)   |
| `cos`    | `(x: number) -> float` | Cosine (radians) |
| `tan`    | `(x: number) -> float` | Tangent (radians)|
| `asin`   | `(x: number) -> float` | Inverse sine     |
| `acos`   | `(x: number) -> float` | Inverse cosine   |
| `atan`   | `(x: number) -> float` | Inverse tangent  |

### String

| Function        | Signature                                     | Description                        |
|-----------------|-----------------------------------------------|------------------------------------|
| `length`        | `(value: string \| array \| record) -> int`   | Length / size / field count         |
| `substring`     | `(s: string, start: int, end: int) -> string` | Extract substring. Indices clamp.  |
| `index_of`      | `(s: string, search: string) -> int`          | Find position (-1 if not found)    |
| `split`         | `(s: string, delimiter: string) -> string[]`  | Split into array                   |
| `trim`          | `(s: string) -> string`                       | Remove surrounding whitespace      |
| `to_upper_case` | `(s: string) -> string`                       | Uppercase                          |
| `to_lower_case` | `(s: string) -> string`                       | Lowercase                          |

`length` on a string counts Unicode scalar values. `length` on a record returns the number of top-level fields.

### Array

| Function  | Signature                                                     | Description                         |
|-----------|---------------------------------------------------------------|-------------------------------------|
| `append`  | `(arr: array, elem: any) -> array`                            | Add to end (new array)              |
| `prepend` | `(arr: array, elem: any) -> array`                            | Add to start (new array)            |
| `pop`     | `(arr: array) -> array`                                       | Remove last (new array). `pop([])` = `[]`. |
| `drop`    | `(arr: array, n: int = 1) -> array`                           | Remove first n (new array). Clamps. |
| `take`    | `(arr: array, n: int = 1) -> array`                           | First n elements (new array). Clamps. |
| `slice`   | `(arr: array, start: int, end: int) -> array`                 | Extract subarray. Indices clamp.    |
| `map`     | `(arr: array, fn: (any) -> any) -> array`                     | Transform each element              |
| `filter`  | `(arr: array, fn: (any) -> boolean) -> array`                 | Select matching elements            |
| `reduce`  | `(arr: array, fn: (any, any) -> any, initial: any) -> any`    | Fold. `reduce([], fn, x)` = `x`.   |
| `range`   | `(n: int) -> int[]`                                           | Generate 0 to n-1. `range(0)` = `[]`. |
| `fill`    | `(n: int, value: T) -> T[]`                                   | Create array of n copies. `fill(0, x)` = `[]`. |

### Type Checking

| Function      | Signature                  | Returns          |
|---------------|----------------------------|------------------|
| `typeof`      | `(value: any) -> string`   | Type name string |
| `is_number`   | `(value: any) -> boolean`  | Is int or float  |
| `is_string`   | `(value: any) -> boolean`  | Is string        |
| `is_boolean`  | `(value: any) -> boolean`  | Is boolean       |
| `is_array`    | `(value: any) -> boolean`  | Is array         |
| `is_record`   | `(value: any) -> boolean`  | Is record        |
| `is_function` | `(value: any) -> boolean`  | Is function      |
| `is_none`     | `(value: any) -> boolean`  | Is none          |

### File System

| Function     | Signature                                    | Description                              |
|--------------|----------------------------------------------|------------------------------------------|
| `file_read`  | `(path: string) -> string`                   | Read entire file as string. Throws on failure. |
| `file_write` | `(path: string, content: string) -> none`    | Write string to file. Throws on failure. |
| `file_exists`| `(path: string) -> boolean`                  | Check if file exists.                    |

### Environment and Process

| Function  | Signature                        | Description                              |
|-----------|----------------------------------|------------------------------------------|
| `env_get` | `(name: string) -> string?`     | Get environment variable. Returns none if unset. |
| `exit`    | `(code: int) -> none`           | Exit the program with a status code.     |
| `args`    | `() -> string[]`                | Get command-line arguments.              |

---

## Keywords

### Reserved Keywords

| Keyword    | Purpose                              |
|------------|--------------------------------------|
| `let`      | Mutable variable declaration         |
| `const`    | Immutable variable declaration       |
| `if`       | Conditional branch                   |
| `else`     | Alternative branch                   |
| `for`      | Loop iteration                       |
| `in`       | Loop target                          |
| `while`    | Conditional loop                     |
| `break`    | Exit loop                            |
| `continue` | Next loop iteration                  |
| `return`   | Function return                      |
| `guard`    | Error handling (try)                 |
| `against`  | Error handling (catch)               |
| `throw`    | Raise an error                       |
| `type`     | Type definition                      |
| `use`      | Import / native header-library use   |
| `export`   | Export                               |
| `from`     | Import source                        |
| `as`       | Alias                                |
| `ref`      | Explicit reference / pointer passing |
| `is`       | Equality (synonym for `==`)          |
| `not`      | Logical negation                     |
| `and`      | Logical AND                          |
| `or`       | Logical OR                           |
| `true`     | Boolean literal                      |
| `false`    | Boolean literal                      |
| `none`     | Null literal                         |

### Reserved for Future Use

| Keyword | Planned Purpose         |
|---------|-------------------------|
| `async` | Asynchronous functions  |
| `await` | Await async result      |

---

## Operator Precedence

From highest to lowest:

| Prec | Operators                         | Assoc | Category        |
|------|-----------------------------------|-------|-----------------|
| 1    | `()` `[]` `.`                     | Left  | Call, index, access |
| 2    | `!` `not` `-` `~`                | Right | Unary           |
| 3    | `*` `/` `%`                      | Left  | Multiplicative  |
| 4    | `+` `-`                          | Left  | Additive        |
| 5    | `<<` `>>`                        | Left  | Shift           |
| 6    | `<` `>` `<=` `>=`                | Left  | Relational      |
| 7    | `==` `!=` `is`                   | Left  | Equality        |
| 8    | `&`                               | Left  | Bitwise AND     |
| 9    | `^`                               | Left  | Bitwise XOR     |
| 10   | `\|`                              | Left  | Bitwise OR      |
| 11   | `and` `&&`                        | Left  | Logical AND     |
| 12   | `or` `\|\|`                       | Left  | Logical OR      |
| 13   | `=` `+=` `-=` `*=` `/=` `%=`     | Right | Assignment      |

---

## Edge Cases and Special Behaviors

### Division
- `int / int` = `int` (truncates toward zero): `10 / 3` produces `3`
- If either operand is `float`, result is `float`: `10.0 / 3` produces `3.333...`
- Integer division with exact result: `10 / 2` produces `5` (int)

### Division and Modulo by Zero
- `x / 0` and `x % 0` are runtime errors

### Integer Overflow
- Integers are 64-bit signed. Overflow behavior is implementation-defined (wrap or error, depending on target).

### Numeric Widening
- `int` widens to `float` implicitly: `let f float = 42` is valid
- `float` to `int` requires explicit conversion: `floor()`, `ceil()`, `round()`

### Truthiness
- `false`, `none`, and `0` are **falsy**
- Everything else is **truthy** (including `""`, `[]`, `{}`)
- This is the only implicit conversion in the language

### none Behavior
- `none == none` is `true`
- `none` compared to any non-none value via `==` is `false`
- `none` in arithmetic or string operations is a **runtime error** (`none + 1` = error)
- `none` is falsy in boolean contexts
- `typeof(none)` returns `"none"`
- Ordering `none` (`none < 5`) is a type error

### Array Out of Bounds
- **Read** returns `none` (graceful)
- **Write** is a runtime error (strict on operations)

### String Indexing
- `"hello"[0]` returns `"h"` (single-character string)
- Out-of-bounds returns `none`
- Strings are indexed by Unicode scalar values

### String Comparison
- Lexicographic ordering by Unicode code points
- `"abc" < "abd"` is `true`

### Cross-Type Operations
- Comparing different types with `==` is a type error (except `none` checks via `is_none`)
- Ordering different types with `<`/`>` is a type error
- Arithmetic on non-numeric types is a type error

### Type Comparisons
- `5 == "5"` is a type error (not `false` — you can't compare different types at all)
- No implicit coercion

### Array/Record/Function Comparison
- `[1, 2] == [1, 2]` is a type error — collections cannot be compared with `==`
- Functions cannot be compared with `==`

### Loop Variables
- `for x in [1, 2, 3]`: x takes values 1, 2, 3
- `for c in "abc"`: c takes values "a", "b", "c"
- Loop variable is `const` — scoped to the loop body, cannot be reassigned

### Range
- `range(5)` returns `[0, 1, 2, 3, 4]`
- `range(0)` returns `[]` (empty array)
- `range(-5)` returns `[]` (empty array)

### Fill
- `fill(3, true)` returns `[true, true, true]`
- `fill(0, x)` returns `[]` (empty array)
- `fill(-1, x)` returns `[]` (empty array)
- Return type matches the value: `fill(n, true)` → `bool[]`, `fill(n, 0)` → `int[]`

### Bounds Clamping
- `slice`, `take`, `drop`, `substring` clamp indices to valid ranges (graceful on reads)
- `pop([])` returns `[]`
- `reduce([], fn, initial)` returns `initial`

### Scope Shadowing
- Inner variables can shadow outer variables
- Outer variable is restored when inner scope exits

### Return from Functions
- `return` without a value returns `none`
- `return` exits the nearest enclosing function
- Every code path in a non-none function must return a value (compile error otherwise)

### Record Shape
- Records have a **fixed shape** from creation — you cannot add new fields after creation
- **Typed records:** accessing or assigning a non-existent field is a compile error
- **Untyped records:** reading a non-existent field returns `none` (graceful on reads); writing a non-existent field is a runtime error (strict on operations)

### Newlines
- Newlines are generally insignificant whitespace
- Expressions continue across lines when the line ends with an operator, comma, or open bracket/brace/paren
- The parser reads ahead when a line is syntactically incomplete

### Trailing Commas
- Allowed in array literals, record literals, function arguments, and function parameters

### Assignment
- Assignment is a statement, not an expression
- `if x = 5 { }` is a syntax error (prevents `=` vs `==` bugs)

### Top-Level Code
- Monk is script-style — top-level statements execute directly
- No `main` function required

---

**This document defines what Monk Lang is. The implementation must conform to this spec. When in doubt, this document wins.**
