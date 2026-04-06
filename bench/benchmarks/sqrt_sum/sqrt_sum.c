#include <stdio.h>
#include <stdint.h>
#include <math.h>
int main() {
    int64_t N = 10000000;
    double sum = 0.0;
    for (int64_t i = 1; i <= N; i++) {
        sum += sqrt((double)i);
    }
    printf("%lld\n", (int64_t)sum);
    return 0;
}
