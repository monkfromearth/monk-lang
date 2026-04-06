const MAX_DEPTH = 14;

function treeSize(depth) {
    let size = 1;
    for (let i = 0; i <= depth; i++) size *= 2;
    return size - 1;
}

function buildAndCheck(depth) {
    const numNodes = treeSize(depth);
    const pool = new Int32Array(numNodes * 3);
    const half = Math.trunc(numNodes / 2);

    for (let i = numNodes - 1; i >= half; i--) {
        const base = i * 3;
        pool[base] = 1;
        pool[base + 1] = -1;
        pool[base + 2] = -1;
    }

    for (let i = half - 1; i >= 0; i--) {
        const base = i * 3;
        pool[base] = 0;
        pool[base + 1] = 2 * i + 1;
        pool[base + 2] = 2 * i + 2;
    }

    const stack = [0];
    let check = 0;

    while (stack.length > 0) {
        const nodeIdx = stack.pop();
        const base = nodeIdx * 3;
        check += pool[base];
        if (pool[base + 1] >= 0) {
            stack.push(pool[base + 1]);
            stack.push(pool[base + 2]);
        }
    }
    return check;
}

let total = buildAndCheck(MAX_DEPTH + 1);
const longLived = buildAndCheck(MAX_DEPTH);

for (let depth = 4; depth <= MAX_DEPTH; depth += 2) {
    let iterations = 1;
    for (let d = 0; d < MAX_DEPTH - depth + 4; d++) iterations *= 2;
    for (let i = 0; i < iterations; i++) {
        total += buildAndCheck(depth);
    }
}
total += longLived;
console.log(total);
