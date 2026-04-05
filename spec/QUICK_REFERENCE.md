# Monk Lang — Quick Reference

> Cheat sheet for the language. For the full spec, see [REFERENCE.md](REFERENCE.md).

---

## Variables

```monk
let name = "Monk"          // mutable
const PI = 3.14159         // immutable (deep freeze)
let x int = 42             // with type annotation
```

## Types

| Type | Example | Falsy value |
|------|---------|-------------|
| `int` | `42`, `0xFF`, `0b1010`, `0o77`, `1_000` | `0` |
| `float` | `3.14`, `1e-3` | never |
| `string` | `"hello"`, `` `template ${x}` `` | never (even `""` is truthy) |
| `boolean` | `true`, `false` | `false` |
| `none` | `none` | always |
| `array` | `[1, 2, 3]` | never (even `[]` is truthy) |
| `record` | `{name: "Alice", age: 30}` | never |
| `function` | `(x int) int { return x * x }` | never |

**Truthiness:** Only `false`, `none`, and `0` are falsy. Everything else is truthy.

## Operators

| Precedence | Operators | Associativity |
|:---:|---|---|
| 1 (lowest) | `or` | left |
| 2 | `and` | left |
| 3 | `\|` | left |
| 4 | `^` | left |
| 5 | `&` | left |
| 6 | `==` `!=` | left |
| 7 | `<` `>` `<=` `>=` | left |
| 8 | `is` | left |
| 9 | `<<` `>>` | left |
| 10 | `+` `-` | left |
| 11 | `*` `/` `%` | left |
| 12 | `not` `!` `-` `~` (unary) | right |
| 13 (highest) | `()` `[]` `.` | left |

**Assignment operators:** `=` `+=` `-=` `*=` `/=` `%=`
**Assignment is a statement**, not an expression. `if x = 5` is a compile error.

**Division:** `int / int = int` (truncates). `float / anything = float`.

## Functions

```monk
let add = (a int, b int) int {
    return a + b
}

let greet = (name string) none {
    show("Hello, " + name)
}

// Functions are expressions — can be passed around
let apply = (f, x int) int { return f(x) }
```

**Arguments are copies** (value semantics). A function cannot modify the caller's data.

## Closures

```monk
let make_counter = () {
    let count = 0
    return () int {
        count += 1
        return count
    }
}

let counter = make_counter()
show(to_string(counter()))  // 1
show(to_string(counter()))  // 2
```

Closures **capture by copy**. The closure owns its snapshot of `count`.

## Control Flow

```monk
// If / else
if x > 0 {
    show("positive")
} else if x == 0 {
    show("zero")
} else {
    show("negative")
}

// While
while condition {
    // ...
}

// For-in (arrays and strings)
for item in [1, 2, 3] {
    show(to_string(item))
}

for char in "hello" {
    show(char)  // one Unicode character per iteration
}

// Range
for i in range(10) {
    show(to_string(i))  // 0 through 9
}

// Break and continue work as expected
```

## Arrays

```monk
let nums = [1, 2, 3, 4, 5]

nums[0]                     // 1 (read — returns none if out of bounds)
nums[10]                    // none (graceful on reads)
length(nums)                // 5

// All array functions return NEW arrays (value semantics)
let more = append(nums, 6)  // [1, 2, 3, 4, 5, 6]
// nums is still [1, 2, 3, 4, 5]

let first3 = take(nums, 3)  // [1, 2, 3]
let rest = drop(nums, 2)    // [3, 4, 5]
let mid = slice(nums, 1, 4) // [2, 3, 4]
```

**Homogeneous:** all elements must be the same type.

## Records

```monk
let person = {name: "Alice", age: 30}

person.name                  // "Alice"
person.age                   // 30
person.missing               // none (graceful on reads)

person.age = 31              // mutation allowed (if let, not const)
person.new_field = "x"       // ERROR — records have fixed shape
```

## Error Handling

```monk
// Throw
let divide = (a int, b int) int {
    if b == 0 { throw "division by zero" }
    return a / b
}

// Guard / Against (replaces try/catch)
guard result = divide(10, 0) against error {
    show("Caught: " + to_string(error))
    result = 0
}

show(to_string(result))      // 0
```

## Value Semantics

**Assignment copies. Function args copy. Closures capture by copy.**

```monk
let original = [3, 1, 2]
let copy = original          // independent copy
copy = append(copy, 4)       // copy is [3, 1, 2, 4]
// original is still [3, 1, 2]
```

No reference types. No shared state. No garbage collector. No refcounting.

## Type Annotations

```monk
let x int = 42               // explicit type
let y = 42                   // inferred as int (first assignment)
let names string[] = ["a"]   // array of strings
let maybe int? = none        // optional type (int or none)
```

## Type Declarations

```monk
type Point = {
    x int
    y int
}

let p Point = {x: 10, y: 20}
```

Structural typing — if the shape matches, the type matches.

## Template Literals

```monk
let name = "World"
let msg = `Hello, ${name}!`  // "Hello, World!"
```

Backtick strings. `${expr}` for interpolation. Supports multiline.

## Module System (planned)

```monk
use math from "./math"
use {sin, cos} from "./trig"
use utils as U from "./utils"
export my_function
```

## Comments

```monk
// Single-line comments only
// No block comments
```

---

## Built-in Functions

### Output & Conversion

| Function | Signature | Description |
|----------|-----------|-------------|
| `show` | `(value) -> none` | Print to stdout with newline |
| `to_string` | `(value) -> string` | Any value to string |
| `to_int` | `(s string) -> int` | Parse integer (strict) |
| `to_float` | `(s string) -> float` | Parse float |

### Math

| Function | Signature | Description |
|----------|-----------|-------------|
| `abs` | `(n) -> number` | Absolute value |
| `floor` | `(n) -> int` | Round down |
| `ceil` | `(n) -> int` | Round up |
| `round` | `(n) -> int` | Round to nearest |
| `sqrt` | `(n) -> float` | Square root (throws if n < 0) |
| `pow` | `(base, exp) -> float` | Power |
| `log` | `(n) -> float` | Natural log |
| `log10` | `(n) -> float` | Base-10 log |
| `exp` | `(n) -> float` | e^n |
| `min` | `(a, b) -> number` | Minimum |
| `max` | `(a, b) -> number` | Maximum |
| `sin` `cos` `tan` | `(n) -> float` | Trig (radians) |
| `asin` `acos` `atan` | `(n) -> float` | Inverse trig |

### String

| Function | Signature | Description |
|----------|-----------|-------------|
| `length` | `(s) -> int` | Unicode codepoint count (also works on arrays) |
| `substring` | `(s, start, end) -> string` | Extract substring (indices clamp) |
| `index_of` | `(s, search) -> int` | Find position (-1 if not found) |
| `split` | `(s, delim) -> string[]` | Split into array |
| `trim` | `(s) -> string` | Strip whitespace |
| `to_upper_case` | `(s) -> string` | Uppercase |
| `to_lower_case` | `(s) -> string` | Lowercase |

### Array

| Function | Signature | Description |
|----------|-----------|-------------|
| `append` | `(arr, elem) -> array` | Add to end (new array) |
| `prepend` | `(arr, elem) -> array` | Add to start (new array) |
| `pop` | `(arr) -> array` | Remove last (new array) |
| `drop` | `(arr, n) -> array` | Remove first n (new array) |
| `take` | `(arr, n) -> array` | First n elements (new array) |
| `slice` | `(arr, start, end) -> array` | Subarray (indices clamp) |
| `range` | `(n) -> int[]` | [0, 1, ..., n-1] |

### Type Checking

| Function | Returns `true` when |
|----------|---------------------|
| `typeof(v)` | Returns type name as string: `"int"`, `"float"`, `"string"`, `"boolean"`, `"none"`, `"array"`, `"record"`, `"function"` |
| `is_number(v)` | int or float |
| `is_string(v)` | string |
| `is_boolean(v)` | boolean |
| `is_array(v)` | array |
| `is_record(v)` | record |
| `is_function(v)` | function |
| `is_none(v)` | none |

### File System & Environment

| Function | Signature | Description |
|----------|-----------|-------------|
| `file_read` | `(path) -> string` | Read entire file (throws on failure) |
| `file_write` | `(path, content) -> none` | Write file (creates/overwrites) |
| `file_exists` | `(path) -> boolean` | Check if file exists |
| `env_get` | `(name) -> string` | Environment variable (empty if unset) |
| `exit` | `(code) -> none` | Exit with status code |
| `args` | `() -> string[]` | Command-line arguments |

---

## Keywords

```
let  const  if  else  for  in  while  break  continue  return
guard  against  throw  type  use  export  from  as
and  or  not  is  true  false  none
```

**Reserved for future:** `ref`, `match`, `enum`, `trait`, `impl`, `self`, `pub`, `mut`, `async`, `await`, `yield`, `defer`, `struct`, `interface`
