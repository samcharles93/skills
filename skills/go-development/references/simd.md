# SIMD in Go (1.26-1.27, experimental)

Both packages need `GOEXPERIMENT=simd` at build and test time. The API is not
covered by the Go 1 compatibility promise and changed between 1.26 and 1.27;
pin the toolchain and expect churn. Feedback goes to go.dev/issue/73787.

| Package | Since | Vector size | Use for |
|---|---|---|---|
| `simd` | 1.27 | unspecified, chosen at run time | portable code; always runs, emulates gaps |
| `simd/archsimd` | 1.26 amd64, 1.27 arm64 NEON + wasm | fixed per type (`Float32x8`) | operations `simd` lacks, hand-tuned kernels |

Default to `simd`. Drop to `archsimd` only for an operation `simd` does not
offer, and only in build-tagged files with an emulation fallback.

## Portable `simd`

Ten types named after the plural element: `Int8s` through `Float64s`. `Len()`
is not a compile-time constant, so loops must not hard-code a width.

```go
//go:build goexperiment.simd

func addTo(dst, a, b []float32) {
	var i int
	n := simd.LoadFloat32s(a).Len()
	for ; i < len(a)-n+1; i += n {
		simd.LoadFloat32s(a[i:]).Add(simd.LoadFloat32s(b[i:])).Store(dst[i:])
	}
	if i < len(a) {
		va, _ := simd.LoadFloat32sPart(a[i:])
		vb, _ := simd.LoadFloat32sPart(b[i:])
		va.Add(vb).StorePart(dst[i:])
	}
}
```

- Load: `LoadT`, `LoadTPart` (zero-fills, returns count), `BroadcastT`.
  Store: `Store`, `StorePart`.
- Comparisons return width-matched masks (`Int8s` gives `Mask8s`) used by
  `IfElse` and `Masked`.
- Reinterpretation is free: `ToBits`, `BitsToIntW`, `BitsToFloatW`,
  `ReshapeToUintW`.
- Gaps in 1.27: no `ReduceSum` (planned), no `OnesCount`, no horizontal or
  pairwise ops, `Sqrt` and `MulAdd` floats only, some ops missing at some widths.

### Dispatch

The compiler clones every function whose signature mentions a `simd` type into
`@simd0` (emulation), `@simd128`, `@simd256` and `@simd512` variants. A function
that uses `simd` only internally becomes a wrapper that dispatches on each call.
Keep dispatch out of inner loops by mentioning a `simd` type in the enclosing
function so the loop calls the specialised copy directly:

```go
func process(rows [][]float32) {
	var _ simd.Float32s
	for _, r := range rows {
		kernel(r)
	}
}
```

These suffixes show up in profiles and stack traces.

### GODEBUG

| Value | Effect |
|---|---|
| `simd=0` | force emulation |
| `simd=128` / `256` / `512` | use that width; `128` panics if features are missing |
| `simd=+128` / `+256` / `+512` | use that width even with a feature missing; may `SIGILL` |

Run tests under `GODEBUG=simd=0` as well as native. The emulated path is what
ships to any machine lacking the hardware, and it is the reference the vector
path must agree with.

## `simd/archsimd`

Types encode width and element: `Float32x4`, `Int32x8`, `Uint8x64`, with masks
`Mask32x4` and so on. Same-meaning operations share a name across widths and
architectures; operations with different edge cases get different names
(`PermuteOrZero` on amd64 wraps modulo 16, `LookupOrZero` on arm64/wasm zeroes
anything outside 0-15). amd64 ops that work within 128-bit lanes of wider
registers carry a `Grouped` suffix.

- Load/store: `LoadFloat32x4(s)`, `LoadFloat32x4Part(s)`,
  `LoadFloat32x4Array(*[4]float32)`, `x.Store(s)`, `x.StorePart(s)`,
  `x.StoreArray(p)`, `BroadcastFloat32x4(v)`.
- Masks: `x.Greater(y)` returns a mask; `x.Masked(m)` zeroes false lanes;
  `x.IfElse(m, y)` picks `x` where true, `y` otherwise.

### 1.26 to 1.27 renames

| 1.26 | 1.27 |
|---|---|
| `Merge` | `IfElse` |
| `As<Type>` | `ToBits`, `BitsToInt32`, `BitsToFloat32`, `ReshapeToUint<W>s` |
| `Load...Slice` / `Store...Slice` | `Load...` / `Store` (slice is the default) |

Fix these before anything else when a 1.26 kernel stops compiling.

### Feature checks

Guard every optional instruction set. Unguarded use on hardware without it dies
with `SIGILL`, which no test on a capable dev machine will catch.

```go
if archsimd.X86.AVX512() { ... } else if archsimd.X86.AVX2() { ... } else { scalar(...) }
```

amd64: `X86.AVX()`, `AVX2()`, `AVX512()`, plus extensions such as
`AVX512GFNI()`, `AVX512VNNI()`. arm64: NEON is baseline; check
`ARM64.PMULL()`, `SVE()`, `SVE2()`.

The check is also a compiler hint: inside an `AVX512()` block,
`x.Add(y).IfElse(m, z)` fuses into one merge-masked `VPADD` instead of
`VPADD` + `VPBLEND`. Hoist the check outside the loop, not inside it.

### Mixing with `simd`

`v.ToArch()` returns `any`; type-switch to the arch type and convert back with
`simd.<T>FromArch`. The compiler specialises the switch away. Taking this route
obliges you to cover every architecture plus emulation, typically as:

```
op_amd64.go     //go:build goexperiment.simd && amd64
op_neon.go      //go:build goexperiment.simd && (arm64 || wasm)
op_other.go     //go:build goexperiment.simd && !(amd64 || arm64 || wasm)
op_emul.go      //go:build goexperiment.simd
```

arm64 code assuming 128-bit vectors will break when SVE lands (planned 1.28).
Switch on the concrete type rather than assuming it.

## Performance rules

- **Loop bound:** write `i < len(s)-n+1`, not `i+n <= len(s)`. The second can
  overflow, so the `prove` pass keeps the bounds checks.
- **No large composites in hot code.** Structs or arrays of vectors stay in
  memory (go.dev/issue/24416), not registers. Pass vectors as separate
  parameters. Not fixed as of 1.27.
- Keep a scalar reference implementation and fuzz the vector path against it,
  covering lengths around every multiple of the vector width for the tail.
- Benchmark with `b.Loop()` and compare with `benchstat` across `GODEBUG=simd`
  levels before claiming a speedup.

## Verifying the gate

The default gate never sees these files. Run explicitly:

```bash
GOEXPERIMENT=simd go vet ./...
GOEXPERIMENT=simd go test ./...
GOEXPERIMENT=simd GODEBUG=simd=0 go test ./...
GOARCH=arm64 GOEXPERIMENT=simd go vet ./...
GOOS=wasip1 GOARCH=wasm GOEXPERIMENT=simd go vet ./...
```

Not supported: AMX, SME, scalable vectors (SVE/RVV) in 1.27.

Sources: go.dev/blog/simd-experiment, go.dev/blog/archsimd.
