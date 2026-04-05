const N = 200000;
let count = 0;
for (let n = 2; n <= N; n++) {
    let isPrime = true;
    for (let d = 2; d * d <= n; d++) {
        if (n % d === 0) {
            isPrime = false;
            break;
        }
    }
    if (isPrime) count++;
}
console.log(count);
