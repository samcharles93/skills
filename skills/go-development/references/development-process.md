# The Go Development Process

A consistent loop for changing Go code. Apply the same sequence whether fixing
a bug, adding a feature, or improving performance. The order matters: each step
produces the evidence the next one depends on.

## 1. Understand before changing

Read the code that owns the behaviour, not the summary of it. Prefer the source
and its tests over prose documentation, which drifts.

- Locate the invariant the code protects, and who depends on it.
- Read the tests around it: they encode contracts a comment may not state.
- When a prior report (an issue, a colleague, an agent) makes a claim about the
  code, verify it against the code before acting on it. Claims decay.

Stop when able to state, in one sentence, what the current behaviour is and why
it was built that way.

## 2. Reproduce before fixing

Write a failing test that demonstrates the defect **before** changing any
implementation. This is not ceremony:

- A test written after the fix often passes for the wrong reason.
- The failure message is the specification of the bug.
- Without it, there is no evidence the fix addressed the reported problem
  rather than a neighbouring one.

For performance defects, the reproduction is a measurement, not a test: capture
a baseline number with the same harness that will measure the fix.

**Verify the reproduction actually fails.** A test that fails to compile has
proved nothing. Run it, read the failure, confirm it describes the defect.

## 3. Change the smallest thing that works

- Fix the cause, not the symptom. A symptom fix moves the bug.
- Preserve documented contracts. If a linter or a test defends an invariant,
  the invariant wins until the owner of that contract says otherwise.
- Leave the surrounding code alone. Unrelated cleanup in a bug fix hides the
  fix in noise and makes the change hard to revert.

## 4. Prove the fix

Re-run the reproduction: it must now pass. Then prove the test is not vacuous:
**mutation-check it**. Temporarily revert the fix (or break the specific line
the fix touched) and confirm the test fails again.

This catches the most expensive class of mistake: a test that passes whether or
not the code is correct. Any absence-assertion (`if _, ok := m[k]; ok`) is
especially prone to going green for the wrong reason after a refactor.

For performance work, measure the same metrics as the baseline, on the same
machine, with the same parameters. Report medians across several runs and the
spread; a single run cannot distinguish a fix from noise.

## 5. Run the full gate

Run `scripts/lint-gate.sh`, or its steps directly:

```bash
gofmt -l .                    # or: golangci-lint fmt --diff
go vet ./...
golangci-lint run ./...
staticcheck ./...
govulncheck ./...
go test ./...
go test -race ./...
```

Everything must pass, not just the package that changed. A change that breaks a
distant package is still broken.

## 6. Report honestly

State what was verified and what was not:

- If a test was skipped, say so.
- If part of the scope was left undone, name it and say why.
- If a fix addresses one cause of a multi-cause problem, say which one and what
  remains. "Fixed" when only half the regression closed is a false report.

## Working on an unfamiliar codebase

Before the first change:

1. Read the build and test entry points (`Makefile`, `Taskfile.yml`, CI config)
   to learn the project's own gate. Use *that* gate, not a generic one.
2. Run the existing test suite to confirm a clean starting point. Debugging a
   pre-existing failure as if it were newly introduced wastes hours.
3. Check the module's Go version in `go.mod`. Write in that version's idiom;
   see `modern-features.md`.
4. Look for repository instructions (`CLAUDE.md`, `AGENTS.md`,
   `CONTRIBUTING.md`) and follow them over general convention.

## Performance work specifically

Performance changes need evidence more than opinion:

1. **Measure first.** Record a baseline with a repeatable harness. Note the
   commit, machine, and parameters. Numbers without those are unfalsifiable.
2. **Attribute before fixing.** Identify which change caused the regression,
   ideally by bisecting with measurements rather than by reading diffs. A
   plausible story is not a cause.
3. **Prove the cause by experiment.** Patch out the suspect and re-measure. If
   the number does not move, the hypothesis was wrong regardless of how good it
   sounded. Record disproved hypotheses: they stop the next person repeating
   the work.
4. **Re-measure after the fix**, and state what did *not* improve.

Common false leads: blaming a comparison function when the cost is in moving
large structs; blaming allocation when the cost is a syscall per iteration;
attributing to the most recent change rather than the one that matches the
shape of the regression.

## Concurrency changes

- Run `go test -race` on any change touching goroutines, channels or shared
  state, and prefer `-count=2` or higher to shake out ordering assumptions.
- Establish lock ordering explicitly. When a function must read state guarded
  by one lock while holding another, snapshot the first before taking the
  second rather than nesting.
- Make goroutine lifetimes explicit: every goroutine needs a defined exit
  condition and an owner that waits for it.

## Test hygiene

- Treat existing tests as close to immutable. Change one only when it is
  clearly wrong, and explain the reasoning explicitly. Never rewrite a test to
  make a failing implementation look correct.
- Tests are documentation of intent: name them for the behaviour they protect,
  not the function they call.
- Bind test HTTP calls and subprocesses to `t.Context()` so a hang fails the
  test rather than the whole run.
- Close what is opened: response bodies, files, stores. A leak in a test is
  still a leak, and it makes the suite flaky under `-race`.
