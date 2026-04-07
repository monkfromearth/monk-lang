#include <stdio.h>
#include <string.h>

#define N 1000000

static char is_prime[N + 1];

int main(void) {
    memset(is_prime, 1, sizeof(is_prime));
    is_prime[0] = 0;
    is_prime[1] = 0;

    for (int i = 2; (long long)i * i <= N; i++) {
        if (is_prime[i]) {
            for (int j = i * i; j <= N; j += i) {
                is_prime[j] = 0;
            }
        }
    }

    int count = 0;
    for (int i = 2; i <= N; i++) {
        if (is_prime[i]) count++;
    }
    printf("%d\n", count);
    return 0;
}
