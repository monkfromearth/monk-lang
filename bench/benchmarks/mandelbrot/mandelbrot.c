/* Benchmark: Mandelbrot set, count of bounded points. Same algorithm as .monk. */
#include <stdio.h>

#define W 800
#define H 800
#define MAX_ITER 50
#define LIMIT 4.0

int main(void) {
    long count = 0;
    for (int y = 0; y < H; y++) {
        double cy = ((double)y / (double)H) * 2.0 - 1.0;
        for (int x = 0; x < W; x++) {
            double cx = ((double)x / (double)W) * 3.0 - 2.0;
            double zx = 0.0, zy = 0.0;
            int iter = 0;
            int escaped = 0;
            while (iter < MAX_ITER) {
                double zx2 = zx * zx;
                double zy2 = zy * zy;
                if (zx2 + zy2 > LIMIT) {
                    escaped = 1;
                    iter = MAX_ITER;
                } else {
                    double new_zx = zx2 - zy2 + cx;
                    zy = 2.0 * zx * zy + cy;
                    zx = new_zx;
                    iter++;
                }
            }
            if (!escaped) count++;
        }
    }
    printf("%ld\n", count);
    return 0;
}
