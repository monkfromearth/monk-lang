N = 400

A = [0] * (N * N)
B = [0] * (N * N)
C = [0] * (N * N)

for i in range(N):
    for j in range(N):
        A[i * N + j] = i + j
        B[i * N + j] = i - j
        C[i * N + j] = 0

for i in range(N):
    for k in range(N):
        aik = A[i * N + k]
        for j in range(N):
            C[i * N + j] += aik * B[k * N + j]

s = C[0] + C[N - 1] + C[N * (N - 1)] + C[N * N - 1]
print(s)
