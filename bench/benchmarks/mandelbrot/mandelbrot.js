const W = 800;
const H = 800;
const MAX_ITER = 50;
const LIMIT = 4.0;

let count = 0;
for (let y = 0; y < H; y++) {
    const cy = y / H * 2.0 - 1.0;
    for (let x = 0; x < W; x++) {
        const cx = x / W * 3.0 - 2.0;
        let zx = 0.0;
        let zy = 0.0;
        let iter = 0;
        let escaped = false;
        while (iter < MAX_ITER) {
            const zx2 = zx * zx;
            const zy2 = zy * zy;
            if (zx2 + zy2 > LIMIT) {
                escaped = true;
                iter = MAX_ITER;
            } else {
                const newZx = zx2 - zy2 + cx;
                zy = 2.0 * zx * zy + cy;
                zx = newZx;
                iter++;
            }
        }
        if (!escaped) count++;
    }
}

console.log(count);
