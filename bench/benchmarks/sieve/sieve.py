N = 1000000
is_prime = bytearray(b'\x01') * (N + 1)
is_prime[0] = 0
is_prime[1] = 0

i = 2
while i * i <= N:
    if is_prime[i]:
        j = i * i
        while j <= N:
            is_prime[j] = 0
            j += i
    i += 1

count = 0
for i in range(2, N + 1):
    if is_prime[i]:
        count += 1
print(count)
