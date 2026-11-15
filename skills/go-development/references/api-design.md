# Go API Design Patterns

## Choosing a Configuration Pattern

| Pattern | Best for | Example |
|---------|----------|---------|
| **Functional options** | Most APIs with optional parameters (the default) | `NewParser(WithTimeout(d), WithLocation(loc))` |
| **Bitmask flags** | Many boolean flags on an allocation-critical path | `Minute | Hour | Dom` |
| **Builder** | Step-by-step construction needing cross-field validation | `NewBuilder().WithX().Build()` |

### Functional Options

```go
type Option func(*config)

func NewParser(opts ...Option) *Parser {
    cfg := defaultConfig() // sane defaults first
    for _, opt := range opts {
        opt(&cfg)
    }
    return &Parser{cfg: cfg}
}

func WithTimeout(d time.Duration) Option {
    return func(c *config) { c.timeout = d }
}
```

Each `WithX` validates its own argument and is a pure setter, so options
compose in any order.

### Bitmask Options

Only when profiling shows option-call overhead matters. Define flags with
`1 << iota`, combine with `|`, test with `opts&Flag != 0`, and ship named
presets (`Standard`, `Extended`) so callers rarely combine raw flags.

### Builder

```go
type Builder struct {
    cfg config
    err error
}

func (b *Builder) WithHashKey(key string) *Builder {
    if key == "" {
        b.err = errors.New("hash key cannot be empty")
        return b
    }
    b.cfg.hashKey = key
    return b
}

func (b *Builder) Build() (*Parser, error) {
    if b.err != nil {
        return nil, b.err
    }
    if b.cfg.needsHashKey() && b.cfg.hashKey == "" {
        return nil, errors.New("hash option requires hash key")
    }
    return &Parser{cfg: b.cfg}, nil
}
```

Chaining methods on the value type (each returning a new value) is a third
variant; use it only when configuration is immutable by design.

## Error Design

The rules (lowercase, no punctuation, no "failed to" prefix, wrap with `%w`,
inspect with `errors.Is`/`errors.As`) are core principles in SKILL.md. On top
of them, prefer a sentinel plus a typed error carrying details:

```go
var ErrEmptySpec = errors.New("empty spec")

type ValidationError struct {
    Field string
    Value string
    Err   error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("invalid %s %q: %v", e.Field, e.Value, e.Err)
}
func (e *ValidationError) Unwrap() error { return e.Err }
```

Callers match with `errors.Is(err, ErrEmptySpec)` and
`var ve *ValidationError; errors.As(err, &ve)`.

## Interface Design

The rules (small, defined where consumed, `-er` names, compile-time checks)
are core principles in SKILL.md. The pattern worth keeping here: split a wide
interface into focused ones and compose at the use site, and accept interfaces
while returning concrete types:

```go
type JobAdder interface{ AddJob(spec string, cmd func()) (EntryID, error) }
type Lifecycle interface{ Start(); Stop(); Running() bool }

type Scheduler interface {
    JobAdder
    Lifecycle
}
```

## Validation API

Offer a quick check and a detailed analysis when callers need feedback, both
thin wrappers over one parser so they cannot disagree:

```go
func ValidateSpec(spec string) error       // nil or error
func AnalyzeSpec(spec string) SpecAnalysis // valid, fields, next run, err
```
