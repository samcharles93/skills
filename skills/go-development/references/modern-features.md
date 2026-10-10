# Modern Go Features (1.24-1.27)

Language, library and toolchain changes from Go 1.24 to 1.27, with the usage
guidance the Go team published alongside them on go.dev/blog. Check the module's
`go` directive before using anything here: from 1.27, `go test` runs the
`stdversion` vet check and reports symbols newer than that version.

## Language Features

### Generic Type Aliases (1.24+)

Create parameterized type aliases for cleaner generic code:

```go
type Set[T comparable] = map[T]bool
type Pair[K comparable, V any] = struct{ Key K; Val V }

// Usage
userSet := Set[string]{"alice": true, "bob": true}
```

**When to use:**
- Simplifying common generic patterns
- Creating domain-specific type names
- Reducing boilerplate in generic code

### new() with Expressions (1.26+)

Allocate and initialize pointers in one step:

```go
// Before
val := 42
p := &val

// After
p := new(42)
s := new([]int{1, 2, 3})
```

**Best practice:** Ideal for creating pointers to optional struct fields for JSON/Protobuf marshaling.

### Recursive Type Constraints (1.26+)

Enable self-referential generic patterns:

```go
type Ordered[T Ordered[T]] interface {
    Less(T) bool
}

type Priority[T Ordered[T]] struct {
    items []T
}
```

**Use case:** Building containers that enforce element comparison operations.

### Generic Methods (1.27+)

A method on a concrete type may declare its own type parameters after its name:

```go
func (l List[E]) Map[R any](f func(E) R) List[R] { ... }

NewList(0, 2, 4).Map(add(2)).Map(divideBy(2))
f := List[int].Map[int] // method expression, must be instantiated
```

`math/rand/v2` now has `(*Rand).N[Int intType](Int) Int`.

**Restrictions:**
- Interface methods still cannot declare type parameters.
- A generic method never satisfies an interface method. Implementation is
  decided by the type's method set, and `T.M[P]` is not `M()`.

**When to use:** to organise operations around a type (chainable transforms on a
container) instead of a package-level generic function. Do not reach for it to
satisfy an interface.

### Generic Interfaces (guidance)

From go.dev/blog/generic-interfaces:

- Use a self-referential constraint for methods taking the same type:
  `type Comparer[T any] interface{ Compare(T) int }` used as `E Comparer[E]`.
  `time.Time` satisfies it, the container's zero value stays usable, and calls
  can inline.
- Keep a generic interface's own type parameters `any`; constrain the concrete
  types that use it instead.
- Build one unconstrained core taking a `cmp func(a, b E) int` *parameter* (not a
  struct field, which the compiler analyses less well), and wrap it for
  `cmp.Ordered`, a custom func, and a `Comparer` method.
- Pointer-receiver constraints (`PS interface{ *S; Set[E] }`, construct with
  `PS(new(S))`) work but exclude value-receiver implementations and invite nil
  panics. Prefer accepting the interface value directly:
  `func InsertAll[E any](set Set[E], seq iter.Seq[E])`.

### Struct Literal Keys (1.27+)

A key in a struct literal may be any valid field selector, including promoted
fields of embedded structs:

```go
type Base struct{ ID int }
type User struct{ Base; Name string }

u := User{ID: 1, Name: "a"} // previously needed Base: Base{ID: 1}
```

### Function Type Inference (1.27+)

A generic function now infers its type arguments wherever it is assigned or
converted to a matching function type, not only when passed as an argument.

## Error Handling

### errors.AsType (1.26+)

Type-safe error checking with compile-time guarantees:

```go
// Old way (runtime reflection)
var appErr *AppError
if errors.As(err, &appErr) {
    // Handle error
}

// New way: scoped variable, target type checked at compile time
if target, ok := errors.AsType[*AppError](err); ok {
    log.Printf("App error: %s", target.Code)
}
```

The main gain is correctness: `errors.As` panics at run time on a bad target,
`AsType` fails to compile. It also avoids reflection.

### Wrapping pointers to errors (vet, 1.27+)

The printf check flags `fmt.Errorf("...: %w", p)` when `p` is `*E` and `E` (not
`*E`) implements `error`. The wrapped value is then the wrong type, so
`errors.Is` and `errors.As` miss it. Wrap the value, or give `*E` the method.

### Enhanced fmt.Errorf (1.26+)

`fmt.Errorf` with a constant string and no arguments now costs the same as
`errors.New`, so there is no performance reason to prefer one:

```go
err := fmt.Errorf("connection closed")
```

## Concurrency Patterns

### WaitGroup.Go() (1.25+)

Simplified concurrent execution:

```go
// Before
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    processTask()
}()
wg.Wait()

// After
var wg sync.WaitGroup
wg.Go(processTask)
wg.Wait()
```

**Best practice:** Use for simple parallel task execution. The method handles Add/Done automatically.

### Weak Pointers (1.24+)

Non-owning references for caches:

```go
type Cache struct {
    mu    sync.Mutex
    items map[string]weak.Pointer[CachedData]
}

func (c *Cache) Get(key string) *CachedData {
    c.mu.Lock()
    defer c.mu.Unlock()
    if wp, ok := c.items[key]; ok {
        return wp.Value() // Returns nil if GC'd
    }
    return nil
}
```

**Use case:** Building caches that don't prevent garbage collection of unused entries.

### Goroutine Leak Profile (1.26 experiment, 1.27 GA)

Experimental in 1.26 behind `GOEXPERIMENT=goroutineleakprofile`; on by default
in 1.27. Services importing `net/http/pprof` expose it at
`/debug/pprof/goroutineleak`:

```sh
curl http://localhost:6060/debug/pprof/goroutineleak > leak.prof
go tool pprof leak.prof
```

A goroutine is reported when it is blocked on a channel, `select`, `Mutex`,
`RWMutex`, `WaitGroup` or `Cond` that no live goroutine can reach. Limits:
primitives reachable from a global or a runnable goroutine hide the leak; I/O,
syscalls and spin locks are never reported; the check is worst-case O(n^2), so
sample periodically (hours apart) rather than continuously. In tests, use
`goleak` and `testing/synctest`; this profile is for staging and production.

Common causes it finds, with the fix:

| Pattern | Fix |
|---|---|
| Early return leaves unbuffered senders blocked | buffer the channel to the sender count |
| Receiver gives up on timeout before the send | buffer size 1, or `select` with `ctx.Done()` |
| Missing `return` after a send | add it |
| `range` over a channel nobody closes | sender closes after the last send |
| Blocking send while holding a lock | release first, or close instead of send |
| `wg.Wait()` inside the loop | move it after the loop |
| `break` with a mutex held | unlock before leaving |

From 1.27, tracebacks include `pprof` goroutine labels in the header
(`GODEBUG=tracebacklabels=0` to disable), which makes labelled workers
identifiable in panic output.

In code:

```go
import "runtime/pprof"

// Identify indefinitely blocked goroutines
prof := pprof.Lookup("goroutineleak")
if prof != nil {
    prof.WriteTo(os.Stderr, 2)
}
```

**Runtime metrics:**
```go
import "runtime/metrics"

samples := []metrics.Sample{
    {Name: "/goroutines-created:goroutines"},
    {Name: "/goroutines/runnable:goroutines"},
    {Name: "/goroutines/waiting:goroutines"},
}
metrics.Read(samples)
```

## Testing Improvements

### Fake Time Testing (1.24+)

Instant timeout testing with synthetic clocks:

```go
import "testing/synctest"

func TestTimeout(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        ch := make(chan int)

        // This 60-second timeout completes instantly
        ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
        defer cancel()

        select {
        case v := <-ch:
            t.Logf("Received: %d", v)
        case <-ctx.Done():
            // Executes immediately, not after 60 seconds
        }
    })
}
```

**Best practice:** Use `synctest.Wait()` to ensure goroutines reach blocking points before assertions.

### Benchmark Loop (1.24+)

Cleaner benchmark code:

```go
// Old way
func BenchmarkOld(b *testing.B) {
    data := makeTestData()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        processData(data)
    }
}

// New way
func BenchmarkNew(b *testing.B) {
    data := makeTestData()
    for b.Loop() {
        processData(data)
    }
}
```

### Test Artifacts (1.26+)

Isolated output directories for debugging:

```go
func TestComplexOperation(t *testing.T) {
    dir := t.ArtifactDir()

    // Write debugging outputs
    logFile := filepath.Join(dir, "execution.log")
    os.WriteFile(logFile, logs, 0644)

    // Write memory dumps
    dumpFile := filepath.Join(dir, "heap.pprof")
    pprof.WriteHeapProfile(dumpFile)
}
```

**Use case:** Essential for debugging CI failures where interactive debugging isn't possible.

### Test Attributes (1.25+)

Metadata for test organization:

```go
func TestFeatureX(t *testing.T) {
    t.Attr("issue", "JIRA-1234")
    t.Attr("component", "auth")
    t.Attr("criticality", "high")
    // Test implementation
}
```

### Test Context and Helpers (1.24+)

Built-in context and working directory management:

```go
func TestService(t *testing.T) {
    // Automatic cancellation when test completes
    srv := startServer(t.Context())

    // Isolated test directory
    t.Chdir(t.TempDir())
}
```

## Standard Library Enhancements

### JSON v2 (1.25 experiment, 1.27 GA)

In 1.27 `encoding/json/v2` and `encoding/json/jsontext` are standard packages
(no `GOEXPERIMENT`), and `encoding/json` is reimplemented on top of v2. v1
behaviour is preserved but error *text* can differ, so tests asserting on
`err.Error()` strings may break; `GOEXPERIMENT=nojsonv2` restores the old code
while you fix them. v2 rejects invalid UTF-8 and duplicate object names by
default. Changes from the experiment: `inline` tag option renamed `embed`;
`format` and `unknown` options, `DiscardUnknownMembers` and `SkipFunc` removed.

Custom marshalers without wrapper types:

```go
import "encoding/json/v2"
import "encoding/json/jsontext"

boolMarshaler := json.MarshalToFunc(func(enc *jsontext.Encoder, val bool) error {
    symbol := "✓"
    if !val {
        symbol = "✗"
    }
    return enc.WriteToken(jsontext.String(symbol))
})

data, _ := json.Marshal(true, json.WithMarshalers(boolMarshaler))
```

### Small Library Additions (1.27+)

- `uuid` package: generate and parse UUIDs; drop third-party UUID deps for new code.
- `strings.CutLast`, `bytes.CutLast`: split around the last separator.
- `url.URL.Clone`, `url.Values.Clone`: deep copies, replacing `u2 := *u` (which
  shares `User` and `Values` maps).
- `math/big.Int.Divide` with `Trunc`/`Floor`/`Round`/`Ceil`.
- `hash/maphash.Hasher` and `ComparableHasher` for custom hash containers.
- `net/http`: HTTP/1 `Response.Body.Close` drains unread content up to a limit,
  so explicit `io.Copy(io.Discard, body)` before close is no longer needed for
  reuse. `Server.MaxHeaderValueCount` caps header values.
- `net/http/httptest.NewTestServer`: server on an in-memory network for
  `testing/synctest`. `synctest.Sleep` combines `time.Sleep` and `synctest.Wait`.
- `testing/cryptotest.SetGlobalRandom` replaces `tls.Config.Rand` (deprecated)
  for deterministic crypto tests.
- `net.UnixConn` reads return bare `io.EOF`, not wrapped in `*OpError`.
- `time` channels are always unbuffered; `asynctimerchan` is gone.

### omitzero Tag (1.24+)

Clearer intent than omitempty:

```go
type Event struct {
    Timestamp time.Time `json:"timestamp,omitzero"` // Omits zero time
    Count     int       `json:"count,omitzero"`     // Omits 0
    Message   string    `json:"message,omitzero"`   // Omits ""
}
```

### String Iterators (1.24+)

Memory-efficient string processing:

```go
// Process large files line-by-line without loading into memory
for line := range strings.Lines(largeContent) {
    processLine(line)
}

// Split with iterators
for field := range strings.SplitSeq(csvLine, ",") {
    processField(field)
}
```

### Multiple Log Handlers (1.26+)

Route logs to multiple destinations:

```go
import "log/slog"

stdoutHandler := slog.NewJSONHandler(os.Stdout, nil)
fileHandler := slog.NewTextHandler(logFile, nil)

multi := slog.NewMultiHandler(stdoutHandler, fileHandler)
logger := slog.New(multi)

// Logs to both stdout and file
logger.Info("Application started", "version", "1.0.0")
```

### CSRF Protection (1.25+)

Built-in cross-origin request protection:

```go
import "net/http"

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/data", handleData)

    antiCSRF := http.NewCrossOriginProtection()
    antiCSRF.AddTrustedOrigin("https://app.example.com")
    antiCSRF.AddTrustedOrigin("https://admin.example.com")

    srv := &http.Server{
        Addr:    ":8080",
        Handler: antiCSRF.Handler(mux),
    }
    srv.ListenAndServe()
}
```

**Note:** Wildcard subdomains are not supported; list each domain explicitly.

### Directory-Scoped Filesystem (1.24+)

Prevent path traversal attacks:

```go
import "os"

func HandleUserFile(userID, filename string) error {
    // Restrict all operations to user directory
    userDir, err := os.OpenRoot(filepath.Join("users", userID))
    if err != nil {
        return err
    }

    // This safely fails - cannot escape directory
    _, err = userDir.Open("../../../etc/passwd")

    // This works - within allowed directory
    data, err := userDir.ReadFile(filename)
    return userDir.WriteFile("processed_"+filename, processed, 0644)
}
```

### Context-Aware Dialing (1.26+)

Network-specific dialing with cancellation:

```go
dialer := &net.Dialer{Timeout: 10 * time.Second}

// New: combines efficiency with context support
conn, err := dialer.DialTCP(ctx, "tcp", tcpAddr)
udpConn, err := dialer.DialUDP(ctx, "udp", udpAddr)
```

## Cryptography

### SHA-3 Support (1.24+)

FIPS 202 compliant hashing:

```go
import "crypto/sha3"

hash := sha3.Sum256([]byte("sensitive data"))
```

### Key Derivation Functions (1.24+)

```go
import "crypto/hkdf"
import "crypto/pbkdf2"

// HKDF for key derivation
key := hkdf.Extract(sha256.New, secret, salt)

// PBKDF2 for password hashing
hash := pbkdf2.Key(password, salt, 600000, 32, sha256.New)
```

### Reader-Less Crypto APIs (1.26+)

Simplified cryptographic key generation:

```go
import "crypto/ecdsa"
import "crypto/elliptic"

// Old way
key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

// New way (uses system randomness automatically)
key, _ := ecdsa.GenerateKey(elliptic.P256(), nil)
```

**Testing:** Use `cryptotest.SetGlobalRandom()` for deterministic tests.

### Hybrid Public Key Encryption (1.26+)

RFC 9180 implementation with post-quantum support:

```go
import "crypto/hpke"

// Setup sender and receiver
sender, receiver := setupHPKE()

// Encrypt large message
ciphertext, _ := sender.Seal(plaintext, associatedData)

// Decrypt
plaintext, _ := receiver.Open(ciphertext, associatedData)
```

### Secret Mode (1.26+, Experimental)

Automatic zeroing of sensitive data:

```go
import "runtime/secret"

func processCredentials(encryptedCreds []byte) error {
    return secret.Do(func() error {
        creds := decrypt(encryptedCreds)
        // Credentials automatically zeroed when function exits
        return authenticate(creds)
    })
}
```

## Performance Features

### Container-Aware GOMAXPROCS (1.25+)

Automatic CPU limit detection in containers:

```go
// Respects Docker --cpus flag automatically
// No manual tuning needed in Kubernetes

// Manual override if needed:
runtime.SetDefaultGOMAXPROCS(4)

// Disable via environment:
// GODEBUG=containermaxprocs=0
```

### Swiss Tables (1.24+)

Optimized map implementation:
- Large maps (>1024 entries): ~30% faster
- Pre-sized maps: ~35% improvement
- Iteration: ~10% faster (60% for low-load maps)

```go
// Pre-size maps for best performance
m := make(map[string]int, expectedSize)
```

### Green Tea GC (1.25-1.26)

Experimental garbage collector optimizations:

```bash
# Enable for testing
GOEXPERIMENT=greenteagc go build

# Production use (Go 1.26+, default on)
# 10-40% reduction in GC overhead
# Optimized for high-core systems
```

### Stack-Allocated Slices (1.24-1.26)

From go.dev/blog/allocation-optimizations. Backing stores that do not escape
increasingly avoid the heap:

- 1.24: `make([]T, 0, 10)` with a constant size.
- 1.25: `make([]T, 0, n)` with a variable size uses a 32-byte stack buffer when
  `n` fits.
- 1.26: `append` from a nil slice starts in a speculative stack buffer; an
  escaping result is moved to the heap once, at its exact size.

Consequence: write the plain version. Pass a size hint to `make`, `append` from
nil, return slices normally, and delete hand-rolled "copy to a right-sized
slice before returning" tricks. Confirm with `-benchmem` allocation counts and
`go build -gcflags=-m`. Disable for diagnosis with
`-gcflags=all=-d=variablemakehash=n`.

### Size-Specialized Allocation (1.27+)

Allocations of 80 bytes or less go through a `mallocgc` variant specialised per
size class, 20-30% cheaper, up to about 1% overall in allocation-heavy programs.
Largest gains at 16 and 24 bytes (strings, interfaces, slice headers). Constant
sizes get a direct call; dynamic sizes dispatch at run time. No code change;
costs about 60 KB of binary. Opt out with `GOEXPERIMENT=nosizespecializedmalloc`
(to be removed in 1.28) only to bisect a problem. For real allocation work,
follow the GC guide: go.dev/doc/gc-guide#Optimization_guide.

### Flight Recorder (1.25+)

Keep the last few seconds of execution trace in memory and dump it only when
something goes wrong. Suits sporadic latency in long-running services where a
continuous trace is too large:

```go
fr := trace.NewFlightRecorder(trace.FlightRecorderConfig{
    MinAge:   10 * time.Second, // about 2x the event being chased
    MaxBytes: 32 << 20,         // busy services produce ~10 MB/s
})
if err := fr.Start(); err != nil { ... }

// on a slow request, once (sync.Once):
f, err := os.Create("snapshot.trace")
if err != nil { ... }
defer f.Close()
_, err = fr.WriteTo(f)
```

Inspect with `go tool trace snapshot.trace`. From 1.27, `-http=:6060` binds to
localhost only.

### SIMD (1.26+, Experimental)

`simd` (portable, 1.27) and `simd/archsimd` (amd64 1.26; arm64 and wasm 1.27)
behind `GOEXPERIMENT=simd`. The API changed between releases; see
`references/simd.md`.

## Tooling

### Tool Dependencies (1.24+)

Track executable dependencies:

```bash
# Add tool dependency
go get -tool golang.org/x/tools/cmd/stringer

# Run versioned tool
go tool stringer -type=Status

# List tools
go list -m -json all | jq -r 'select(.Tool) | .Path'
```

### Modernized go fix (1.26+)

`go fix` now runs the modernizers. Run it after every toolchain upgrade, to a
fixed point, once per build configuration. Commands, the analyzer list and the
review workflow are in `references/go-fix.md`.

### `//go:fix inline` for API migration (1.26+)

Mark a forwarding declaration and `go fix` (and gopls) rewrites every caller to
the replacement:

```go
// Deprecated: the parameter order is confusing. Use newmath.Sub.
//go:fix inline
func Sub(y, x int) int { return newmath.Sub(x, y) }

//go:fix inline
type Rational = newmath.Rational

//go:fix inline
const Pi = newmath.Pi
```

`oldmath.Sub(1, 10)` becomes `newmath.Sub(10, 1)`. This is the preferred way to
retire an exported API: implement the old one in terms of the new, deprecate it,
annotate it. Limits: a callee with `defer` is not inlined by `go fix`; the
inliner introduces `var` bindings when substitution would reorder side effects;
inlining can leave an unused local the caller must delete.

### JSON Build Output (1.24+)

Structured build reporting:

```bash
go test -json ./... | jq -r 'select(.Action=="fail") | .Package'
```

## Reflection Enhancements

### TypeAssert (1.25+)

Avoid boxing overhead:

```go
// Old way (boxes value)
person := reflect.ValueOf(alice).Interface().(Person)

// New way (no boxing)
person, ok := reflect.TypeAssert[Person](aliceVal)
```

### Reflective Iterators (1.26+)

Iterator-based type inspection:

```go
import "reflect"

typ := reflect.TypeFor[http.Client]()

// Iterate fields
for f := range typ.Fields() {
    fmt.Printf("Field: %s, Type: %s\n", f.Name, f.Type)
}

// Iterate methods
for m := range typ.Methods() {
    fmt.Printf("Method: %s\n", m.Name)
}
```

## Migration Guide

### Go 1.24 Adoption
1. Update error handling to use `omitzero` instead of `omitempty` for time.Time
2. Replace manual line splitting with `strings.Lines()`
3. Use `b.Loop()` in new benchmarks
4. Add tool dependencies to go.mod

### Go 1.25 Adoption
1. Replace `var wg sync.WaitGroup; wg.Add(1); go func() { defer wg.Done(); ... }` with `wg.Go()`
2. Enable container-aware GOMAXPROCS (automatic)
3. Add CSRF protection to HTTP handlers
4. Use `slog.NewMultiHandler()` for multi-destination logging
5. Use JSON v2 for custom marshaling needs

### Go 1.26 Adoption
1. Replace `errors.As` with `errors.AsType` for compile-time target checking
2. Use `go fix` to modernize existing code
3. Add goroutine leak detection to production monitoring
4. Use test artifacts for CI debugging
5. Adopt context-aware dialing for network code
6. Annotate deprecated exported APIs with `//go:fix inline`

### Go 1.27 Adoption
1. Run `go fix ./...` (note the `waitgroupgo` rename in scripts)
2. Re-run tests that assert on `encoding/json` error strings
3. Fix new vet findings: `stdversion`, `%w` with a pointer to a value-receiver error
4. Replace third-party UUID packages with `uuid` in new code
5. Scrape `/debug/pprof/goroutineleak` periodically in staging
6. `go mod tidy` collapses `require` blocks to one direct, one indirect
7. Re-run `gofumpt`/`gofmt`: alignment fixes may produce whitespace diffs

## Best Practices

### Error Handling
```go
// Prefer errors.AsType for type-safe error inspection
if appErr, ok := errors.AsType[*AppError](err); ok {
    return handleAppError(appErr)
}

// Use fmt.Errorf for error wrapping (zero-allocation in 1.26+)
return fmt.Errorf("failed to process request: %w", err)
```

### Concurrency
```go
// Use WaitGroup.Go for simple parallel execution
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Go(func() { processTask(task) })
}
wg.Wait()

// Monitor goroutines in production: scrape /debug/pprof/goroutineleak
```

### Testing
```go
// Use fake time for timeout tests
func TestOperation(t *testing.T) {
    synctest.Test(t, func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
        defer cancel()
        // Test completes instantly, not in 1 hour
    })
}

// Add attributes for CI integration
func TestCriticalPath(t *testing.T) {
    t.Attr("criticality", "high")
    t.Attr("component", "payment")
}

// Use artifacts for debugging
func TestComplex(t *testing.T) {
    dir := t.ArtifactDir()
    defer writeDebugLog(dir, logs)
}
```

### Performance
```go
// Pre-size maps for known capacity
cache := make(map[string]CachedData, expectedSize)

// Use weak pointers for cache entries
type Cache struct {
    items map[string]weak.Pointer[Data]
}

// Enable container-aware GOMAXPROCS (automatic in 1.25+)
// Respect cgroup CPU limits in Kubernetes
```

### Security
```go
// Use os.Root for directory-scoped operations
userRoot, _ := os.OpenRoot(userDir)
content, _ := userRoot.ReadFile(filename) // Safe from path traversal

// Add CSRF protection to API handlers
csrf := http.NewCrossOriginProtection()
srv.Handler = csrf.Handler(apiMux)

// Use secret.Do for sensitive data
secret.Do(func() {
    processCredentials(decryptedCreds)
    // Credentials zeroed on exit
})
```
