/*
 * Error handling: guard/against/throw via setjmp/longjmp.
 *
 * guard_stack is a linked list of contexts, top-of-stack first. Each
 * guard block pushes a context before setjmp; monk_throw pops the top
 * context, stores the error value, and longjmps back.
 *
 * Unhandled throws terminate the program with the error's string form.
 */

#include "internal.h"
#include <stdio.h>
#include <stdlib.h>
#include <setjmp.h>

static MonkGuardContext *guard_stack = NULL;
static MonkValue current_error;
static bool has_error = false;

void monk_guard_begin_ctx(MonkGuardContext *ctx) {
    ctx->prev = guard_stack;
    guard_stack = ctx;
    has_error = false;
}

void monk_guard_end(MonkGuardContext *ctx) {
    guard_stack = ctx->prev;
}

void monk_throw(MonkValue error) {
    if (!guard_stack) {
        /* Unhandled throw at top level — terminate program */
        char *s = monk_value_to_cstr(error);
        fprintf(stderr, "monk: unhandled error: %s\n", s);
        free(s);
        exit(1);
    }
    current_error = monk_deep_copy(error);
    has_error = true;
    MonkGuardContext *ctx = guard_stack;
    guard_stack = ctx->prev;
    longjmp(ctx->buf, 1);
}

MonkValue monk_current_error(void) {
    return current_error;
}
