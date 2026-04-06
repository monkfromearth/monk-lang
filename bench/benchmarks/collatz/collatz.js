const N = 1000000;
let bestStart = 1;
let bestLen = 1;

for (let n = 2; n <= N; n++) {
    let current = n;
    let chainLen = 1;
    while (current !== 1) {
        if (current % 2 === 0) {
            current = Math.trunc(current / 2);
        } else {
            current = current * 3 + 1;
        }
        chainLen++;
    }
    if (chainLen > bestLen) {
        bestLen = chainLen;
        bestStart = n;
    }
}
console.log(bestStart);
