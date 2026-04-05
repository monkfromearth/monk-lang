#!/usr/bin/env bash
# Monk Lang benchmark harness.
#
# Usage:
#   ./bench/run.sh            # run all benchmarks, all available languages
#   ./bench/run.sh fibonacci  # run one benchmark only
#
# Requires: hyperfine, jq. Optional (auto-skipped if missing): go, python3, node, bun.
# Monk + C are required.

set -euo pipefail

cd "$(dirname "$0")/.."  # project root

BENCH_DIR="bench"
BUILD_DIR="$BENCH_DIR/build"
RESULTS_DIR="$BENCH_DIR/results"
BENCHMARKS=("fibonacci" "mandelbrot" "matmul")
FILTER="${1:-}"

# ─── Preflight ─────────────────────────────────────────────────────────────────
command -v hyperfine >/dev/null || { echo "missing: hyperfine (brew install hyperfine)"; exit 1; }
command -v jq >/dev/null || { echo "missing: jq (brew install jq)"; exit 1; }
command -v cc >/dev/null || { echo "missing: cc"; exit 1; }
[[ -x ./monk ]] || { echo "missing: ./monk (run 'make build' first)"; exit 1; }

# Detect optional languages.
HAVE_GO=0; HAVE_PY=0; HAVE_NODE=0; HAVE_BUN=0
command -v go >/dev/null && HAVE_GO=1
command -v python3 >/dev/null && HAVE_PY=1
command -v node >/dev/null && HAVE_NODE=1
command -v bun >/dev/null && HAVE_BUN=1

mkdir -p "$BUILD_DIR" "$RESULTS_DIR"

# ─── Environment info ──────────────────────────────────────────────────────────
TS="$(date +%Y-%m-%d-%H%M)"
RESULTS_MD="$RESULTS_DIR/${TS}-results.md"
RESULTS_JSON_DIR="$RESULTS_DIR/${TS}-raw"
mkdir -p "$RESULTS_JSON_DIR"

monk_ver="$(./monk version 2>&1 | head -1)"
cc_ver="$(cc --version 2>&1 | head -1)"
os_info="$(uname -sr)"
cpu_info="$(sysctl -n machdep.cpu.brand_string 2>/dev/null || uname -m)"

echo "=== Monk Benchmarks — $TS ==="
echo "host: $os_info / $cpu_info"
echo "monk: $monk_ver"
echo "cc:   $cc_ver"
[[ $HAVE_GO -eq 1 ]]   && echo "go:   $(go version | awk '{print $3}')"
[[ $HAVE_PY -eq 1 ]]   && echo "py:   $(python3 --version)"
[[ $HAVE_NODE -eq 1 ]] && echo "node: $(node --version)"
[[ $HAVE_BUN -eq 1 ]]  && echo "bun:  $(bun --version)"
echo

# ─── Results header ────────────────────────────────────────────────────────────
{
    echo "# Monk Lang Benchmarks — $TS"
    echo
    echo "## Environment"
    echo
    echo "- Host: \`$os_info\` / $cpu_info"
    echo "- monk: \`$monk_ver\`"
    echo "- cc: \`$cc_ver\`"
    [[ $HAVE_GO -eq 1 ]]   && echo "- go: \`$(go version | awk '{print $3}')\`"
    [[ $HAVE_PY -eq 1 ]]   && echo "- python3: \`$(python3 --version)\`"
    [[ $HAVE_NODE -eq 1 ]] && echo "- node: \`$(node --version)\`"
    [[ $HAVE_BUN -eq 1 ]]  && echo "- bun: \`$(bun --version)\`"
    echo "- hyperfine: \`$(hyperfine --version)\`"
    echo "- Flags: \`cc -O3 -flto\`, \`go build\` (default), Monk \`monk build\` (→ cc -O3 -flto)"
    echo "- Hyperfine runs: \`--warmup 3 --runs 10\`"
    echo
    echo "## Summary"
    echo
    echo "Mean wall-time in milliseconds. Lower is better. **× C** column = ratio to C baseline."
    echo
} > "$RESULTS_MD"

# ─── Per-benchmark ─────────────────────────────────────────────────────────────
declare -a SUMMARY_ROWS

run_benchmark() {
    local name="$1"
    local bdir="$BENCH_DIR/benchmarks/$name"
    local out="$BUILD_DIR/$name"
    mkdir -p "$out"

    echo "── $name ──────────────────────"

    # All language implementations share the benchmark name: <name>.{monk,c,go,py,js}
    local src_monk="$bdir/$name.monk"
    local src_c="$bdir/$name.c"

    ./monk build "$src_monk" -o "$out/monk" 2>/dev/null
    cc -O3 -flto -o "$out/c" "$src_c"
    [[ $HAVE_GO -eq 1 && -f "$bdir/$name.go" ]] && go build -o "$out/go" "$bdir/$name.go"

    # Correctness gate
    local expected="$(cat "$bdir/expected.txt")"
    local commands=()
    local names=()

    check_output() {
        local label="$1"; shift
        local actual
        actual="$("$@")"
        if [[ "$actual" != "$expected" ]]; then
            echo "  [FAIL] $label: got '$actual', expected '$expected'"
            return 1
        fi
        echo "  [ok]   $label → $actual"
    }

    check_output "monk" "$out/monk" || return 1
    check_output "c"    "$out/c"    || return 1
    commands+=("$out/monk" "$out/c")
    names+=("monk" "c")

    if [[ $HAVE_GO -eq 1 && -x "$out/go" ]]; then
        check_output "go" "$out/go" || return 1
        commands+=("$out/go"); names+=("go")
    fi
    if [[ $HAVE_PY -eq 1 && -f "$bdir/$name.py" ]]; then
        check_output "python" python3 "$bdir/$name.py" || return 1
        commands+=("python3 $bdir/$name.py"); names+=("python")
    fi
    if [[ $HAVE_NODE -eq 1 && -f "$bdir/$name.js" ]]; then
        check_output "node" node "$bdir/$name.js" || return 1
        commands+=("node $bdir/$name.js"); names+=("node")
    fi
    if [[ $HAVE_BUN -eq 1 && -f "$bdir/$name.js" ]]; then
        check_output "bun" bun "$bdir/$name.js" || return 1
        commands+=("bun $bdir/$name.js"); names+=("bun")
    fi

    # Hyperfine
    local json="$RESULTS_JSON_DIR/$name.json"
    local hf_args=("--warmup" "3" "--runs" "10" "--shell=none" "--export-json" "$json")
    for i in "${!commands[@]}"; do
        hf_args+=("--command-name" "${names[$i]}" "${commands[$i]}")
    done
    hyperfine "${hf_args[@]}"

    # Extract C baseline in seconds
    local c_mean
    c_mean=$(jq -r '.results[] | select(.command=="c") | .mean' "$json")

    # Append per-benchmark section to results.md
    {
        echo "### $name"
        echo
        echo "| Language | Mean (ms) | StdDev (ms) | × C |"
        echo "|----------|----------:|------------:|----:|"
        jq -r --arg cmean "$c_mean" '
          .results[] |
          [
            .command,
            (.mean * 1000 | . * 100 | round / 100),
            (.stddev * 1000 | . * 100 | round / 100),
            ((.mean / ($cmean | tonumber)) * 100 | round / 100)
          ] | "| \(.[0]) | \(.[1]) | \(.[2]) | \(.[3])× |"
        ' "$json"
        echo
    } >> "$RESULTS_MD"

    echo
}

for b in "${BENCHMARKS[@]}"; do
    if [[ -n "$FILTER" && "$FILTER" != "$b" ]]; then continue; fi
    run_benchmark "$b"
done

echo "=== Done. Results → $RESULTS_MD ==="
