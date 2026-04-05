W = 800
H = 800
MAX_ITER = 50
LIMIT = 4.0

count = 0
for y in range(H):
    cy = y / H * 2.0 - 1.0
    for x in range(W):
        cx = x / W * 3.0 - 2.0
        zx = 0.0
        zy = 0.0
        iter_ = 0
        escaped = False
        while iter_ < MAX_ITER:
            zx2 = zx * zx
            zy2 = zy * zy
            if zx2 + zy2 > LIMIT:
                escaped = True
                iter_ = MAX_ITER
            else:
                new_zx = zx2 - zy2 + cx
                zy = 2.0 * zx * zy + cy
                zx = new_zx
                iter_ += 1
        if not escaped:
            count += 1

print(count)
