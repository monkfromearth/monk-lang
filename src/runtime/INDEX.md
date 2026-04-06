# `src/runtime/` — C runtime library

Linked into every compiled Monk binary. ~2–5 KB overhead. No dynamic loader,
no GC, no refcount — pure value semantics via `monk_deep_copy`/`monk_free`.

## Files

| File             | Contains                                                       |
| ---------------- | -------------------------------------------------------------- |
| `runtime.h`      | **public API** — `MonkValue`, every `monk_*` function signature |
| `internal.h`     | shared helper declarations (not for generated code)            |
| `value.c`        | `monk_panic`, allocators, UTF-8 helpers, constructors, `monk_deep_copy_heap`/`monk_free_heap`, `monk_is_truthy`, `monk_type_name`, `monk_value_to_cstr`, `monk_show`, `monk_to_string`, `monk_to_int`, `monk_to_float` |
| `arith.c`        | `monk_equal`/`less`/`greater` (+ `_equal` variants), `monk_add`/`sub`/`mul`/`div`/`mod`/`neg` |
| `string.c`       | `monk_string_concat`, `length`, `substring`, `index_of`, `split`, `trim`, `to_upper_case`, `to_lower_case`, `string_index` |
| `container.c`    | array ops (`_get`/`_set`, `append`/`prepend`, `pop`/`drop`/`take`/`slice`, `range`) + record ops (`_get`/`_set`) |
| `math.c`         | `abs`, `floor`/`ceil`/`round`, `sqrt`/`pow`/`log`/`log10`/`exp`, `min`/`max`, trig |
| `builtins.c`     | `typeof`, `is_*`, file I/O, `env_get`, `exit`, `args`          |
| `error.c`        | `monk_guard_begin_ctx`, `monk_guard_end`, `monk_throw`, `monk_current_error` (setjmp/longjmp) |
| `runtime_test.c` | standalone C test harness                                      |

## Build

`cc` links all `.c` files for each Monk binary. The Go side
(`src/main.go`, `src/embed.go`) tracks the list in `runtimeSources` and
`embeddedRuntimeFiles` — **keep those in sync with the files in this dir**.

## Design

- **Value semantics.** Every `monk_*` entry point that stores a value
  calls `monk_deep_copy`. Every `*_set` frees the old value first. This is
  slow but gives predictable ownership — no refcount races, no GC pauses.
- **Graceful reads, strict writes.** `monk_array_get` out-of-bounds → `none`.
  `monk_array_set` out-of-bounds → panic. Records: missing field read → `none`,
  missing field write → panic.
- **UTF-8 aware indexing.** Strings index by codepoint, not byte. See
  `monk_utf8_strlen` + `monk_utf8_offset` in `value.c`.
- **Error handling via setjmp/longjmp.** `guard/against` pushes a context;
  `throw` pops it. See `error.c`.
