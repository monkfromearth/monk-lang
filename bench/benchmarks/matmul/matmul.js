const N = 400;

const A = new Array(N * N).fill(0);
const B = new Array(N * N).fill(0);
const C = new Array(N * N).fill(0);

for (let i = 0; i < N; i++) {
    for (let j = 0; j < N; j++) {
        A[i * N + j] = i + j;
        B[i * N + j] = i - j;
        C[i * N + j] = 0;
    }
}

for (let i = 0; i < N; i++) {
    for (let k = 0; k < N; k++) {
        const aik = A[i * N + k];
        for (let j = 0; j < N; j++) {
            C[i * N + j] += aik * B[k * N + j];
        }
    }
}

const sum = C[0] + C[N - 1] + C[N * (N - 1)] + C[N * N - 1];
console.log(sum);
