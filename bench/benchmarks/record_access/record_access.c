#include <stdio.h>
#include <stdint.h>
typedef struct { double x, y, z; } Vec3;
int main() {
    int64_t N = 1000000;
    Vec3 r = {0, 0, 0};
    double sum = 0;
    for (int64_t i = 0; i < N; i++) {
        r.x = (double)i;
        r.y = r.x * 2.0;
        r.z = r.x + r.y;
        sum += r.x + r.y + r.z;
    }
    printf("%lld\n", (int64_t)sum);
    return 0;
}
