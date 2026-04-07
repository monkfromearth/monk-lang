/*
 * Miscellaneous builtins: type introspection, file I/O, environment,
 * process exit, command-line args.
 *
 * File I/O funcs read/write whole files in one shot. There is no
 * streaming API — spec decision to keep the surface tiny.
 */

#include "internal.h"
#include <stdio.h>
#include <stdlib.h>

/* --- Type checking --- */

MonkValue monk_typeof(MonkValue v)      { return monk_string(monk_type_name(v)); }
MonkValue monk_is_number(MonkValue v)   { return monk_bool(v.kind == MONK_INT || v.kind == MONK_FLOAT); }
MonkValue monk_is_string(MonkValue v)   { return monk_bool(v.kind == MONK_STRING); }
MonkValue monk_is_boolean(MonkValue v)  { return monk_bool(v.kind == MONK_BOOL); }
MonkValue monk_is_array(MonkValue v)    { return monk_bool(v.kind == MONK_ARRAY || v.kind == MONK_INT_ARRAY || v.kind == MONK_FLOAT_ARRAY || v.kind == MONK_BOOL_ARRAY); }
MonkValue monk_is_record(MonkValue v)   { return monk_bool(v.kind == MONK_RECORD); }
MonkValue monk_is_function(MonkValue v) { return monk_bool(v.kind == MONK_FUNCTION); }
MonkValue monk_is_none(MonkValue v)     { return monk_bool(v.kind == MONK_NONE); }

/* --- File system & environment --- */

MonkValue monk_file_read(MonkValue path) {
    if (path.kind != MONK_STRING) monk_panic("file_read: expected string path");
    FILE *f = fopen(path.str_val, "rb");
    if (!f) monk_panic("file_read: cannot open file");
    fseek(f, 0, SEEK_END);
    long size = ftell(f);
    if (size < 0) { fclose(f); monk_panic("file_read: cannot determine file size"); }
    fseek(f, 0, SEEK_SET);
    char *buf = monk_malloc_internal(size + 1);
    size_t read = fread(buf, 1, size, f);
    buf[read] = '\0';
    fclose(f);
    return (MonkValue){.kind = MONK_STRING, .str_val = buf};
}

MonkValue monk_file_write(MonkValue path, MonkValue content) {
    if (path.kind != MONK_STRING || content.kind != MONK_STRING)
        monk_panic("file_write: expected two strings");
    FILE *f = fopen(path.str_val, "wb");
    if (!f) monk_panic("file_write: cannot open file");
    /* fputs returns EOF on write failure (disk full, permissions, etc.).
     * Close first so the FD isn't leaked, THEN panic. */
    int rc = fputs(content.str_val, f);
    if (fclose(f) != 0 || rc == EOF) monk_panic("file_write: write failed");
    return monk_none();
}

MonkValue monk_file_exists(MonkValue path) {
    if (path.kind != MONK_STRING) monk_panic("file_exists: expected string path");
    FILE *f = fopen(path.str_val, "r");
    if (f) { fclose(f); return monk_bool(true); }
    return monk_bool(false);
}

MonkValue monk_env_get(MonkValue name) {
    if (name.kind != MONK_STRING) monk_panic("env_get: expected string");
    char *val = getenv(name.str_val);
    if (!val) return monk_none();
    return monk_string(val);
}

void monk_exit(MonkValue code) {
    if (code.kind != MONK_INT) monk_panic("exit: expected int");
    exit((int)code.int_val);
}

MonkValue monk_args(void) {
    /* Will be populated by generated main() with argc/argv */
    return monk_array(NULL, 0);
}
