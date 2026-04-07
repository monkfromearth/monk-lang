/*
 * Internal runtime helpers — shared across the split runtime .c files.
 *
 * These are NOT part of the public API (runtime.h) and must not be called
 * from generated Monk code. Kept out of runtime.h so user programs don't
 * accidentally link against allocation internals.
 */

#ifndef MONK_INTERNAL_H
#define MONK_INTERNAL_H

#include "runtime.h"
#include <stddef.h>

/* --- Error-reporting allocators (panic on failure) --- */

char *monk_strdup_internal(const char *s);
void *monk_malloc_internal(size_t size);
void *monk_realloc_internal(void *old, size_t size);

/* --- UTF-8 helpers --- */

int64_t monk_utf8_strlen(const char *s);
int64_t monk_utf8_offset(const char *s, int64_t n);

/* --- Numeric coercion used by arithmetic + math builtins --- */

double monk_as_c_double(MonkValue v);

/* --- Shared string representation of any value (malloc'd, caller frees) --- */

char *monk_value_to_cstr(MonkValue v);

/* --- Typed-array to generic conversion (non-consuming) --- */

MonkValue monk_typed_to_generic(MonkValue v);

/* Free a generic MONK_ARRAY created by monk_typed_to_generic.
 * Frees elements, data pointer, and MonkArray struct.
 * Shared by container.c (append/pop/etc.) and higher_order.c (map/filter/reduce). */
void monk_free_generic_intermediate(MonkValue arr);

#endif /* MONK_INTERNAL_H */
