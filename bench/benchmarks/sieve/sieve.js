const N = 1000000;
const isPrime = new Uint8Array(N + 1).fill(1);
isPrime[0] = 0;
isPrime[1] = 0;

for (let i = 2; i * i <= N; i++) {
    if (isPrime[i]) {
        for (let j = i * i; j <= N; j += i) {
            isPrime[j] = 0;
        }
    }
}

let count = 0;
for (let i = 2; i <= N; i++) {
    if (isPrime[i]) count++;
}
console.log(count);
