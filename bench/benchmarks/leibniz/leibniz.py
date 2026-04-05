N = 50_000_000
s = 0.0
for k in range(N):
    term = 1.0 / (2 * k + 1)
    if k % 2 == 0:
        s += term
    else:
        s -= term
pi = 4.0 * s
print(int(pi * 10000000))
