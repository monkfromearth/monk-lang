const N = 50000000;
let sum = 0.0;
for (let k = 0; k < N; k++) {
    const term = 1.0 / (2 * k + 1);
    if (k % 2 === 0) {
        sum += term;
    } else {
        sum -= term;
    }
}
const pi = 4.0 * sum;
console.log(Math.trunc(pi * 10000000));
