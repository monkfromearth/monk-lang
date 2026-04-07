#include <stdio.h>
#include <stdint.h>
int main() {
    int64_t N = 10000000;
    int64_t total = 0;
    for (int64_t i = 0; i < N; i++) total += i;
    printf("%lld\n", total);
    return 0;
}
