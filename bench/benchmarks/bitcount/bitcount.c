#include <stdio.h>
#include <stdint.h>
int main() {
    int64_t N = 10000000;
    int64_t total = 0;
    for (int64_t n = 1; n <= N; n++) {
        int64_t x = n;
        while (x > 0) {
            total += x & 1;
            x >>= 1;
        }
    }
    printf("%lld\n", total);
    return 0;
}
