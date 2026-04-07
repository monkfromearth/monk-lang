/*
 * String builtins: concat, length, substring, index_of, split, trim,
 * case conversion, and indexed character access.
 *
 * All string operations are UTF-8 aware for indexing (character count,
 * not byte count). Helpers monk_utf8_* live in value.c.
 */

#include "internal.h"
#include <stdlib.h>
#include <string.h>
#include <ctype.h>

MonkValue monk_string_concat(MonkValue a, MonkValue b) {
    /* Design decision: string + string only. No auto-coercion. */
    if (a.kind != MONK_STRING || b.kind != MONK_STRING)
        monk_panic("string concatenation requires two strings (use to_string())");
    size_t len = strlen(a.str_val) + strlen(b.str_val) + 1;
    char *result = monk_malloc_internal(len);
    strcpy(result, a.str_val);
    strcat(result, b.str_val);
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

void monk_string_append_in_place(MonkValue *target, MonkValue suffix) {
    if (target->kind != MONK_STRING || suffix.kind != MONK_STRING)
        monk_panic("string append requires two strings (use to_string())");
    size_t old_len = strlen(target->str_val);
    size_t suffix_len = strlen(suffix.str_val);
    bool self_append = target->str_val == suffix.str_val;
    char *result = monk_realloc_internal(target->str_val, old_len + suffix_len + 1);
    /* In-place concat assignment for `s = s + rhs` / `s += rhs`.
     * Pass: `s += s` appends from the reallocated buffer itself.
     * Fail: reading suffix.str_val after realloc would use a freed pointer. */
    if (self_append) {
        memmove(result + old_len, result, suffix_len + 1);
    } else {
        memcpy(result + old_len, suffix.str_val, suffix_len + 1);
    }
    target->str_val = result;
}

MonkValue monk_length(MonkValue v) {
    switch (v.kind) {
    case MONK_STRING:      return monk_int(monk_utf8_strlen(v.str_val));
    case MONK_ARRAY:       return monk_int(v.array_val->length);
    case MONK_INT_ARRAY:   return monk_int(v.int_array_val->length);
    case MONK_FLOAT_ARRAY: return monk_int(v.float_array_val->length);
    case MONK_BOOL_ARRAY:  return monk_int(v.bool_array_val->length);
    case MONK_RECORD:      return monk_int(v.record_val->length);
    default: monk_panic("length: expected string, array, or record"); return monk_none();
    }
}

MonkValue monk_substring(MonkValue s, MonkValue start_v, MonkValue end_v) {
    if (s.kind != MONK_STRING) monk_panic("substring: expected string");
    int64_t slen = monk_utf8_strlen(s.str_val);
    int64_t start = start_v.int_val;
    int64_t end = end_v.int_val;
    /* Design decision: indices clamp (graceful on reads) */
    if (start < 0) start = 0;
    if (end > slen) end = slen;
    if (start >= end) return monk_string("");
    int64_t byte_start = monk_utf8_offset(s.str_val, start);
    int64_t byte_end = monk_utf8_offset(s.str_val, end);
    int64_t byte_len = byte_end - byte_start;
    char *result = monk_malloc_internal(byte_len + 1);
    memcpy(result, s.str_val + byte_start, byte_len);
    result[byte_len] = '\0';
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

MonkValue monk_index_of(MonkValue s, MonkValue search) {
    if (s.kind != MONK_STRING || search.kind != MONK_STRING)
        monk_panic("index_of: expected two strings");
    const char *found = strstr(s.str_val, search.str_val);
    if (!found) return monk_int(-1);
    /* Return character index, not byte index */
    int64_t byte_pos = found - s.str_val;
    int64_t char_pos = 0;
    for (int64_t i = 0; i < byte_pos; i++) {
        if ((s.str_val[i] & 0xC0) != 0x80) char_pos++;
    }
    return monk_int(char_pos);
}

MonkValue monk_split(MonkValue s, MonkValue delim) {
    if (s.kind != MONK_STRING || delim.kind != MONK_STRING)
        monk_panic("split: expected two strings");
    /* Count parts first */
    int64_t count = 1;
    const char *p = s.str_val;
    size_t dlen = strlen(delim.str_val);
    if (dlen == 0) {
        /* Split by character */
        int64_t slen = monk_utf8_strlen(s.str_val);
        MonkValue *parts = monk_malloc_internal(sizeof(MonkValue) * slen);
        p = s.str_val;
        for (int64_t i = 0; i < slen; i++) {
            int64_t clen = 1;
            while ((p[clen] & 0xC0) == 0x80) clen++;
            char *ch = monk_malloc_internal(clen + 1);
            memcpy(ch, p, clen);
            ch[clen] = '\0';
            parts[i] = (MonkValue){.kind = MONK_STRING, .str_val = ch};
            p += clen;
        }
        MonkValue result = monk_array(parts, slen);
        for (int64_t i = 0; i < slen; i++) free(parts[i].str_val);
        free(parts);
        return result;
    }
    while ((p = strstr(p, delim.str_val)) != NULL) { count++; p += dlen; }
    MonkValue *parts = monk_malloc_internal(sizeof(MonkValue) * count);
    p = s.str_val;
    for (int64_t i = 0; i < count; i++) {
        const char *next = (i < count - 1) ? strstr(p, delim.str_val) : p + strlen(p);
        size_t part_len = next - p;
        char *part = monk_malloc_internal(part_len + 1);
        memcpy(part, p, part_len);
        part[part_len] = '\0';
        parts[i] = (MonkValue){.kind = MONK_STRING, .str_val = part};
        p = next + dlen;
    }
    MonkValue result = monk_array(parts, count);
    for (int64_t i = 0; i < count; i++) free(parts[i].str_val);
    free(parts);
    return result;
}

MonkValue monk_trim(MonkValue s) {
    if (s.kind != MONK_STRING) monk_panic("trim: expected string");
    const char *start = s.str_val;
    while (*start && isspace((unsigned char)*start)) start++;
    const char *end = s.str_val + strlen(s.str_val);
    while (end > start && isspace((unsigned char)*(end - 1))) end--;
    size_t len = end - start;
    char *result = monk_malloc_internal(len + 1);
    memcpy(result, start, len);
    result[len] = '\0';
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

/* Case conversion is ASCII-only. Monk's string indexing is UTF-8 aware,
 * but to_upper_case/to_lower_case operate byte-wise via toupper/tolower.
 * For ASCII the result is correct; for non-ASCII Unicode letters (é, ñ,
 * Cyrillic, CJK…) the bytes pass through unchanged. Full Unicode case
 * mapping would require pulling in an ICU-style table, which Monk's
 * tiny-runtime goal explicitly avoids. */
MonkValue monk_to_upper_case(MonkValue s) {
    if (s.kind != MONK_STRING) monk_panic("to_upper_case: expected string");
    char *result = monk_strdup_internal(s.str_val);
    for (char *p = result; *p; p++) *p = toupper((unsigned char)*p);
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

MonkValue monk_to_lower_case(MonkValue s) {
    if (s.kind != MONK_STRING) monk_panic("to_lower_case: expected string");
    char *result = monk_strdup_internal(s.str_val);
    for (char *p = result; *p; p++) *p = tolower((unsigned char)*p);
    return (MonkValue){.kind = MONK_STRING, .str_val = result};
}

MonkValue monk_string_index(MonkValue s, MonkValue index) {
    if (s.kind != MONK_STRING) monk_panic("cannot index non-string");
    int64_t idx = index.int_val;
    int64_t slen = monk_utf8_strlen(s.str_val);
    if (idx < 0 || idx >= slen) return monk_none();
    int64_t byte_start = monk_utf8_offset(s.str_val, idx);
    int64_t byte_end = monk_utf8_offset(s.str_val, idx + 1);
    int64_t clen = byte_end - byte_start;
    char *ch = monk_malloc_internal(clen + 1);
    memcpy(ch, s.str_val + byte_start, clen);
    ch[clen] = '\0';
    return (MonkValue){.kind = MONK_STRING, .str_val = ch};
}
