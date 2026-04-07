# `src/runtime/` — C runtime library

Linked into every compiled Monk binary. ~2–5 KB overhead. No dynamic loader,
no GC — pure value semantics via `monk_deep_copy`/`monk_free`, with
copy-on-write refcounts hidden inside array backing stores.

## Files

| File             | Contains                                                       |
| ---------------- | -------------------------------------------------------------- |
| `runtime.h`      | **public API** — `MonkValue` (incl. `MONK_INT/FLOAT/BOOL_ARRAY` kinds + typed structs), every `monk_*` signature |
| `internal.h`     | shared helper declarations (not for generated code), `monk_typed_to_generic`, `monk_free_generic_intermediate` |
| `value.c`        | constructors, `monk_deep_copy_heap`/`monk_free_heap`, array COW share/free, `monk_type_name`, `monk_value_to_cstr`, `monk_show`, typed array converters (`monk_int/float/bool_array_from`) |
| `arith.c`        | `monk_equal`/`less`/`greater` (+ `_equal` variants), `monk_add`/`sub`/`mul`/`div`/`mod`/`neg` |
| `string.c`       | `monk_string_concat`, `length` (handles typed arrays), `substring`, `index_of`, `split`, `trim`, `to_upper_case`, `to_lower_case`, `string_index` |
| `container.c`    | array ops (`_get`/`_set` handle typed arrays; structural mutators convert-to-generic first), COW detach helpers, `monk_typed_to_generic` (shared), `monk_free_generic_intermediate` (single def), `monk_fill`/`monk_fill_bool`/`monk_fill_int`/`monk_fill_float`, `monk_range_int`, record ops |
| `math.c`         | `abs`, `floor`/`ceil`/`round`, `sqrt`/`pow`/`log`/`log10`/`exp`, `min`/`max`, trig |
| `builtins.c`     | `typeof`, `is_array` (true for all array kinds), other `is_*`, file I/O, `env_get`, `exit`, `args` |
| `error.c`        | `monk_guard_begin_ctx`, `monk_guard_end`, `monk_throw`, `monk_current_error` (setjmp/longjmp) |
| `higher_order.c` | `monk_map`, `monk_filter`, `monk_reduce` — uses shared `monk_typed_to_generic` + `monk_free_generic_intermediate` |
| `runtime_test.c` | standalone C test harness                                      |

## Build

`cc` links all `.c` files for each Monk binary. The Go side embeds the
entire `runtime/` directory via `embed.FS` — adding a new `.c` file here
requires only updating this `INDEX.md`. No Go code changes needed.

## Design

- **Value semantics.** Every `monk_*` entry point that stores a value
  calls `monk_deep_copy`. Arrays share backing storage copy-on-write and
  detach before mutation; user-visible behavior remains independent copies.
- **Typed backing-store arrays.** `int[]`/`float[]`/`bool[]` Monk variables use
  `MONK_INT_ARRAY`/`MONK_FLOAT_ARRAY`/`MONK_BOOL_ARRAY` internally — `int64_t*`/
  `double*`/`bool*` data pointers instead of `MonkValue*`. Halves memory per element.
  `monk_int_array_from()` converts from generic `MONK_ARRAY` or COW-shares a typed
  array. `monk_type_name`, `is_array`, `length` all treat typed arrays as "array".
- **Graceful reads, strict writes.** `monk_array_get` out-of-bounds → `none`.
  `monk_array_set` out-of-bounds → panic. Records: missing field read → `none`,
  missing field write → panic. Codegen fast-paths for typed arrays always panic on OOB.
- **UTF-8 aware indexing.** Strings index by codepoint, not byte. See
  `monk_utf8_strlen` + `monk_utf8_offset` in `value.c`.
- **Error handling via setjmp/longjmp.** `guard/against` pushes a context;
  `throw` pops it. See `error.c`.
