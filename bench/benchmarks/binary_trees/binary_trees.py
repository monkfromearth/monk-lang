MAX_DEPTH = 14

def tree_size(depth):
    size = 1
    for _ in range(depth + 1):
        size *= 2
    return size - 1

def build_and_check(depth):
    num_nodes = tree_size(depth)
    pool = [0] * (num_nodes * 3)
    half = num_nodes // 2

    for i in range(num_nodes - 1, half - 1, -1):
        base = i * 3
        pool[base] = 1
        pool[base + 1] = -1
        pool[base + 2] = -1

    for i in range(half - 1, -1, -1):
        base = i * 3
        pool[base] = 0
        pool[base + 1] = 2 * i + 1
        pool[base + 2] = 2 * i + 2

    stack = [0]
    check = 0
    while stack:
        node_idx = stack.pop()
        base = node_idx * 3
        check += pool[base]
        if pool[base + 1] >= 0:
            stack.append(pool[base + 1])
            stack.append(pool[base + 2])
    return check

total = build_and_check(MAX_DEPTH + 1)
long_lived = build_and_check(MAX_DEPTH)

depth = 4
while depth <= MAX_DEPTH:
    iterations = 1
    for _ in range(MAX_DEPTH - depth + 4):
        iterations *= 2
    for _ in range(iterations):
        total += build_and_check(depth)
    depth += 2

total += long_lived
print(total)
