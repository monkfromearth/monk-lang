/* C reference: Levenshtein edit distance with char-level comparison. */
#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

int main() {
    char s1[101], s2[101];
    for (int i = 0; i < 20; i++) {
        memcpy(s1 + i * 5, "abcde", 5);
        memcpy(s2 + i * 5, "aXcYe", 5);
    }
    s1[100] = s2[100] = '\0';

    int len1 = 100, len2 = 100;
    int W = len2 + 1;
    int64_t total = 0;

    for (int round = 0; round < 50; round++) {
        int *dp = malloc((len1 + 1) * W * sizeof(int));
        for (int i = 0; i <= len1; i++) dp[i * W] = i;
        for (int j = 0; j <= len2; j++) dp[j] = j;

        for (int i = 1; i <= len1; i++) {
            for (int j = 1; j <= len2; j++) {
                int cost = (s1[i - 1] != s2[j - 1]) ? 1 : 0;
                int del = dp[(i - 1) * W + j] + 1;
                int ins = dp[i * W + (j - 1)] + 1;
                int sub = dp[(i - 1) * W + (j - 1)] + cost;
                int min = del;
                if (ins < min) min = ins;
                if (sub < min) min = sub;
                dp[i * W + j] = min;
            }
        }
        total += dp[len1 * W + len2];
        free(dp);
    }
    printf("%lld\n", total);
    return 0;
}
