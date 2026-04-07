#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <ctype.h>

int main() {
    const char *base = "the Quick Brown fox Jumps Over The Lazy Dog";
    int len = (int)strlen(base);
    int64_t total = 0;
    char upper[64], lower[64];
    for (int i = 0; i < 500000; i++) {
        for (int j = 0; j < len; j++) {
            upper[j] = (char)toupper((unsigned char)base[j]);
            lower[j] = (char)tolower((unsigned char)base[j]);
        }
        upper[len] = '\0';
        lower[len] = '\0';
        total += len + len;
    }
    printf("%lld\n", total);
    return 0;
}
