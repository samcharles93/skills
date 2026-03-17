# Modern Go Features (1.24-1.26)

This document covers the latest Go language features, performance improvements, and best practices introduced in Go 1.24, 1.25, and 1.26.

## Language Features

### Generic Type Aliases (1.24+)

Create parameterized type aliases for cleaner generic code:

```go
type Set[T comparable] = map[T]bool
type Result[T any] = (T, error)

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

## Error Handling

### errors.AsType (1.26+)

Type-safe error checking with compile-time guarantees:

```go
// Old way (runtime reflection)
var appErr *AppError
if errors.As(err, &appErr) {
    // Handle error
}

// New way (compile-time safety, ~3x faster)
if target, ok := errors.AsType[*AppError](err); ok {
    log.Printf("App error: %s", target.Code)
}
```

**Benefits:**
- No reflection overhead
- Scoped variables (no pre-declaration)
- Type safety at compile time
- 3x performance improvement

### Enhanced fmt.Errorf (1.26+)

Zero-allocation error creation for simple cases:

```go
// Now matches errors.New performance when no formatting needed
err := fmt.Errorf("database connection failed: %w", dbErr)
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
    items map[string]weak.Pointer[*CachedData]
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

### Goroutine Monitoring (1.26+)

Production goroutine leak detection:

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

### JSON v2 (1.25+)

Custom marshalers without wrapper types:

```go
import "encoding/json/v2"
import "encoding/json/v2/jsontext"

boolMarshaler := json.MarshalToFunc(func(enc *jsontext.Encoder, val bool) error {
    symbol := "✓"
    if !val {
        symbol = "✗"
    }
    return enc.WriteToken(jsontext.String(symbol))
})

data, _ := json.Marshal(true, json.WithMarshalers(boolMarshaler))
```

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

### SIMD Operations (1.26+, Experimental)

Vectorized operations on amd64:

```go
import "simd/archsimd"

func vectorAdd(a, b []float32) []float32 {
    result := make([]float32, len(a))
    for i := 0; i+16 <= len(a); i += 16 {
        va := archsimd.LoadFloat32x16Slice(a[i:i+16])
        vb := archsimd.LoadFloat32x16Slice(b[i:i+16])
        vSum := va.Add(vb) // Single instruction
        vSum.StoreFloat32x16Slice(result[i:i+16])
    }
    return result
}
```

**Performance:** 10-40x improvement for vector operations.

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

Safe code modernization:

```bash
# Preview changes
go fix -diff .

# Apply all analyzers
go fix .

# Specific analyzer
go fix -forvar .

# All except specific analyzer
go fix -omitzero=false .
```

**Example transformations:**
- Loop to `slices.Contains`: Replaces manual iteration with standard library
- Deprecated API updates: Migrates to current patterns
- Error wrapping: Adds `%w` to fmt.Errorf calls

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
1. Replace `errors.As` with `errors.AsType` for better performance
2. Use `go fix` to modernize existing code
3. Add goroutine leak detection to production monitoring
4. Use test artifacts for CI debugging
5. Adopt context-aware dialing for network code

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

// Monitor goroutines in production
if leaked := checkGoroutineLeaks(); leaked > 0 {
    logger.Warn("goroutine leak detected", "count", leaked)
}
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
    items map[string]weak.Pointer[*Data]
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
