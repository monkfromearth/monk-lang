/* C reference: naive O(n^2) string concatenation to match Monk semantics. */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

int main() {
    int N = 10000;
    char *s = malloc(1);
    s[0] = '\0';
    int64_t len = 0;
    for (int i = 0; i < N; i++) {
        int64_t new_len = len + 5;
        char *ns = malloc(new_len + 1);
        memcpy(ns, s, len);
        memcpy(ns + len, "hello", 5);
        ns[new_len] = '\0';
        free(s);
        s = ns;
        len = new_len;
    }
    printf("%lld\n", len);
    free(s);
    return 0;
}
