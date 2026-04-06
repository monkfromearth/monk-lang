N = 1000000
best_start = 1
best_len = 1

for n in range(2, N + 1):
    current = n
    chain_len = 1
    while current != 1:
        if current % 2 == 0:
            current //= 2
        else:
            current = current * 3 + 1
        chain_len += 1
    if chain_len > best_len:
        best_len = chain_len
        best_start = n
print(best_start)
