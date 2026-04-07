#include <stdio.h>
#include <stdint.h>
#include <string.h>

#define MAXN 12

int main() {
    int N = 10;
    int64_t perm[MAXN], perm1[MAXN], count[MAXN];
    int64_t max_flips = 0, perm_count = 0, checksum = 0;

    for (int i = 0; i < N; i++) perm1[i] = i;

    int r = N;
    for (;;) {
        while (r > 1) { count[r - 1] = r; r--; }

        memcpy(perm, perm1, N * sizeof(int64_t));
        int64_t flips = 0;
        int64_t k;
        while ((k = perm[0]) != 0) {
            int64_t k2 = (k + 1) / 2;
            for (int64_t i = 0; i < k2; i++) {
                int64_t tmp = perm[i];
                perm[i] = perm[k - i];
                perm[k - i] = tmp;
            }
            flips++;
        }
        if (flips > max_flips) max_flips = flips;
        checksum += (perm_count % 2 == 0) ? flips : -flips;
        perm_count++;

        /* Next permutation */
        for (;;) {
            if (r == N) goto done;
            int64_t perm0 = perm1[0];
            for (int i = 0; i < r; i++) perm1[i] = perm1[i + 1];
            perm1[r] = perm0;
            count[r]--;
            if (count[r] > 0) break;
            r++;
        }
    }
done:
    printf("%lld\n%lld\n", checksum, max_flips);
    return 0;
}
