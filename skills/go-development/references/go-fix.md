# go fix (1.26+)

From Go 1.26, `go fix` runs the modernizer analyzers from the shared analysis
framework (the same one behind `go vet` and gopls). Every fix it applies must be
safe to apply without review for correctness, performance or style; that is the
bar separating a fixer from a vet check.

## Commands

```bash
go fix -diff ./...          # preview
go fix ./...                # apply, silently rewriting files
go tool fix help            # list analyzers
go tool fix help forvar     # one analyzer's documentation
go fix -rangeint ./...      # only this analyzer
go fix -any=false ./...     # all except this one
```

Fixes to generated files are discarded; fix the generator.

## Workflow

1. Start from a clean git tree so the commit holds only `go fix` edits.
2. Raise the `go` directive first. Modernizers only fire in files whose
   effective version (from `go.mod` or a `//go:build go1.N` line) already allows
   the feature.
3. Run `go fix ./...` **until it reaches a fixed point**, usually twice. One fix
   can enable another, and edits overlapping an earlier fix are skipped with a
   warning until the next run.
4. Run once per build configuration. Each run sees one GOOS/GOARCH, so tagged
   files are otherwise skipped:

   ```bash
   GOOS=linux   GOARCH=amd64 go fix ./...
   GOOS=darwin  GOARCH=arm64 go fix ./...
   GOOS=windows GOARCH=amd64 go fix ./...
   ```

5. Build. Two individually safe fixes can combine into a compile error, usually
   a now-unused variable. Fix those by hand.
6. Consider one commit per analyzer (`go fix -<name> ./...`) when the diff spans
   hundreds of files; reviewers can then check each rewrite pattern once.
7. Run the full quality gate.

Repeat on every toolchain upgrade. New modernizers ship with most releases, and
proposal review now asks whether a language or library change needs one.

## Analyzers (1.26)

| Analyzer | Rewrites |
|---|---|
| `any` | `interface{}` to `any` |
| `buildtag` | checks `//go:build` and `// +build` lines |
| `fmtappendf` | `[]byte(fmt.Sprintf(...))` to `fmt.Appendf` (removed in 1.27) |
| `forvar` | drops redundant `x := x` in range loops (1.22 semantics) |
| `hostport` | checks address strings passed to `net.Dial` |
| `inline` | applies `//go:fix inline` directives |
| `mapsloop` | explicit map loops to `maps` calls |
| `minmax` | if/else clamping to `min`/`max` |
| `rangeint` | three-clause loops to `for i := range n` |
| `stringscut` | `strings.Index` plus slicing to `strings.Cut` |
| `stringsbuilder` | string `+=` in loops to `strings.Builder` |
| `newexpr` | `newInt`-style helpers to `new(expr)` |

1.27 adds `atomictypes`, `embedlit`, `slicesbackward` and `unsafefuncs`, and
renames `waitgroup` to `waitgroupgo`. Check `go tool fix help` for the current
list rather than trusting this table.

Deliberately excluded: `appendclipped` (`append([]T{}, s...)` to
`slices.Clone`), because `Clone` of an empty slice returns nil, a behaviour
change.

### `newexpr`

```go
data, err := json.Marshal(&RequestJSON{
    URL:      url,
    Attempts: new(10),
})
```

Afterwards the helpers may be unused. Delete them unless they are part of a
published API (then annotate them `//go:fix inline` instead), and use
`deadcode` to find them. Calls where `new` is shadowed stay unfixed.

## `//go:fix inline`

Mark a forwarding function, alias or constant, and the `inline` analyzer
rewrites callers to the target. This is how to retire an exported API: implement
the old one via the new, add `// Deprecated:`, then annotate. Details and limits
in `modern-features.md`.

## In the editor and in agents

The same modernizers run in gopls as diagnostics while typing, and gopls' MCP
server exposes those diagnostics to coding agents. Prefer acting on them over
hand-written modernisation: the fixer's output is known safe.

Source: go.dev/blog/gofix, go.dev/blog/inliner.
