---
trigger: glob
globs: **/*.go
---

# Go Coding Rules

When developing or modifying Go code in KHI, adhere to these rules alongside `.agents/rules/comment-guideline.md` and `.agents/skills/backward_compatibility_policy/SKILL.md`.

## Verification Commands

- **Build & Test**: Run `make build-go` to verify compilation, `make test-go` for full-strength tests, and `make test-go-short` for `-short`.
- **Fast Gate**: Run `make qa-fast-must` (or `make pre-commit`) for formatting, license headers, `lint-go` (`depguard` / `no-x-exp-stdlib`, `errorlint`, `forbidigo`, `nolintlint`, `staticcheck`, `govet`, `no-short-skip`), `go test -short`, and fast AST rules. Naked `//nolint` comments are forbidden.
- **Slow & Advisory QA**: Run `make qa-slow-must QA_LANG=go` (`code_clone`, `deadcode`, `unused_exports`, `go test -race`, cross-OS builds) and `make qa-slow-advisory QA_LANG=go FORMAT=json` (`covcheck`, `mutcheck`, `test_redundancy`, opt-in `CODEQUALITY_SIMILARITY=true`). `covcheck` and `mutcheck` run with `-short`; `make test-go` and PR CI run full strength.
- **Semantic Decisions**: Opt-in `CODEQUALITY_DECISIONS=true` evaluates `scripts/codequality/rules/*.yaml` (`go_code_quality`, `go_test_quality`, `go_refactor_safety`) via the Gemini Decisions API (auto-skips when no API key is configured; in Swarm runs without it, Auditors consult the YAML rules directly).
- **Untracked Files in Git**: In `git` workspaces, stage new files (`git add -A`) before running `make qa-fast` or `make qa-slow-advisory` with `BASE=HEAD` so untracked files are included in the diff.
- Restart verification from the start of a phase after any edit.

## Quality Budgets and Advisory Policy

`make qa-fast-must` and `make qa-slow-must` enforce hard limits on new/compliant units and a **strict no-worsening ratchet (`after <= before`)** on legacy units:

- **Production Functions (`MUSTFIX`)**:
  - `func_cognitive <= 15`, `func_nesting <= 4` (`else if` adds no nesting; prefer early returns), `func_code_lines <= 60` (comments/blanks excluded—never strip GoDoc), `func_statements <= 40`, `func_params <= 5`, `func_results <= 3`.
  - `code_clone <= 15` consecutive normalized AST statements (`qa-slow-must`).
  - `defer_unlock_before_lock == 0`: never place `defer x.Unlock()`/`RUnlock()` before `x.Lock()`/`RLock()`/`TryLock()` in the same block. When fixing mutex/defer ordering in one method, audit **all** sibling methods touching the same mutex.
- **Files, Style & Test Complexity (`MUSTFIX`)**:
  - `file_code_lines <= 450` (prod), `test_file_code_lines <= 800` (test), excluding comments/blanks/imports.
  - `catchall_filename`: never name new files `util(s).go`, `helper(s).go`, `common.go`, `misc.go`, `shared.go`, or `*_util(s).go` / `*_helper(s).go` / `*_misc.go`.
  - `multiline_string_escape`: use raw backquote literals for multiline strings instead of middle `\n` escapes (exceptions: trailing `\n` in format strings, `cmp.Diff` failure headers, or content containing backquotes).
  - `trun_cognitive <= 11`, `test_helper_cognitive <= 15` (prefer splitting unrelated behaviors into separate `Test*` functions over shattering flat tests into single-caller `t.Helper()` wrappers).
- **Advisory Findings (`qa-fast-advisory` & `qa-slow-advisory`, Exit `0`)**:
  - `func_uncovered_lines`, `func_surviving_mutants`, `test_redundancy`, and `advisorycheck` diff signals (`test_removed`, `test_cases_reduced`, `test_assertion_weakened`, `test_skip_added`, `test_stress_reduced`, `test_sleep_added`, `suppression_added`, `orphan_test_file`) are **advisory** for review.
  - **Never** delete legitimate defensive guards (`n == 0`, `res != nil`), slice/map capacity pre-allocations (`make(..., 0, cap)`), or pre-existing tests, and never add `CODEQUALITY_ALLOW` to silence advisories.
  - **100% statement/mutant coverage is NOT a goal**: never poke unexported internals or add timer/spin-poll waits solely to clear advisories (GC zeroing, sealed-interface guards, defensive branches, and interface boilerplate require no test contortions).
  - Strengthen assertions (`cmp.Diff` on whole structs, boundary inputs) first, and prefer a single-argument `*With` injection seam (e.g., `runWith(deps)`) for testing hard-to-reach I/O boundaries.
  - For `test_redundancy` (`SAME PATH` / `NO OUTPUT CHECK`), assert distinct concrete outputs rather than deleting pre-existing tests.

## Go Coding Conventions

- **Comments & GoDoc**: Follow `comment-guideline.md`. Document all exported declarations, struct fields, and interface methods (`// SymbolName <third-person verb>... .`) and non-obvious private symbols. Keep comments strictly aligned with current behavior, boundaries, data structures, locks, and errors.
- **Identifier Naming**: Outside narrow Go idioms (`ctx`, `err`, `t`/`b`, `r`/`w`, concise receivers, short loop `i`), avoid opaque placeholders (`d`, `m`, `v`, `tmp`, `val`, `res`, `data`, `proc`, `doWork`, `handle`). Match names to exact domain role, unit, polarity, mutability, and side effects; omit package/receiver stutter (`yamlconfig.Parse`, `JobName`).

```go
// BAD: cryptic names ("d", "m"), missing unit, and hidden mutation in "check".
func check(d int64, m map[string]string) error { ... }

// GOOD: domain role, exact unit, and mutating verb are explicit.
func pruneExpiredEntries(timeoutMillis int64, entriesByKey map[string]string) time.Duration { ... }
```

- **Imports & Declaration Ordering**:
  - Group imports (`gci`): stdlib, third-party, `github.com/GoogleCloudPlatform/khi/...`.
  - Order top-level declarations (`decorder` + `decl-visibility-order`): `const -> var -> type -> func`, exported before unexported (preserve `iota` blocks), with compile-time interface checks (`var _ Iface = (*Impl)(nil)`) in `var` before `type`.
  - Order functions (`funcorder` + `decl-visibility-order`): `init()`, exported functions and exported-receiver groups (`New*`/`new*` -> exported methods -> unexported methods), then unexported-receiver groups and unexported helpers.
- **Error & Assertion Messages**: Start `fmt.Errorf`, `t.Error*`, and `t.Fatal*` messages lowercase (`"failed to ..."`) unless beginning with a capitalized identifier or acronym (`MyFunc() mismatch (-want +got)`).
- **No Backward Compatibility**: Internal Go code requires no compatibility shims or defensive nil fallbacks; update all callers directly (`backward-compatibility-policy`).
- **Surgical Diffs**: Tie every change to a concrete bug, explicit rule/AST violation, or genuine coverage gap; do not rewrite working algorithms or add unrequested locks, abstractions, or APIs.

## Go Testing Practices

1. **Test File Naming**: Tests for `A.go` must live in `A_test.go` (except `interface_test.go`, `*_integration_test.go`, `*_script_test.go`, `*_contract_test.go`).
2. **Table-Driven Tests & Case Comments**: Use `testCases := []struct{...}` with `t.Run()`. Place a `// ... .` comment immediately above **every** table case `{ ... }` and a `// TestXxx ... .` GoDoc comment directly above every `Test*`. Verify the described behavior genuinely without coincidental passes.
3. **Linear Arrange, Act, Assert (AAA)**: Separate Arrange, Act, and Assert with blank lines (add `// Arrange.`, `// Act.`, `// Assert.` when multi-block). Test one behavior per case; never use `if`/`else` or `for` to compute expected values.
4. **Structuring `want` in Tables**: Declare literal `want` values in the table (never derive `want` from inputs inside `t.Run`). Use a single `want ReturnType` struct + `cmp.Diff` instead of field-by-field `want*` fields (keep separate fields only for independent side effects or staged setup).
5. **Assertions & Diffs**: Compare scalars with `==`/`!=` (`t.Errorf("got %v, want %v", got, tc.want)`). Use `cmp.Diff(tc.want, got)` for structs, slices, maps, and multiline strings. Never use `reflect.DeepEqual`.
6. **Testing Callbacks**: Never place assertions (`t.Error*`, `t.Fatal*`, `cmp.Diff`) inside non-`Run`/`Cleanup`/`Fuzz` callbacks. Record calls/args in Arrange (using mutex/atomic/channel if concurrent), run Act, and assert outside after Act.
7. **Stress Tests & `testing.Short()`**: Declare full-strength defaults first and scale parameters inside `if testing.Short() { ... }` while keeping bounds sufficient to exercise every branch and lock path. Under `no-short-skip`, never call `t.Skip*`, return early, or guard logic with `!testing.Short()`.
8. **Deterministic Concurrency (`testing/synctest`) & Test Hygiene**: Use `testing/synctest` (`synctest.Test` + `synctest.Wait`) with channel synchronization for concurrency, coalescing, timeout, and clock-dependent tests instead of `time.Sleep`, wall-clock assumptions, or probabilistic retry loops (`sync.Mutex`/`sync.RWMutex` are not durably blocking in `synctest`, so prefer channel-based waits when observing blocked goroutines, and pair with a lightweight timing-independent `-race` test asserting only schedule-independent invariants). Reuse consistent shared setup helpers across sibling tests instead of repeating per-type boilerplate, and split unrelated behaviors into separate `Test*` functions rather than shattering flat tests into single-caller `t.Helper()` wrappers.
9. **Surgical Test Edits & Harness Reuse**: Never rewrite already-passing `*_test.go` files wholesale, rename/split passing `Test*` functions, or mass-add boilerplate comments to untouched test cases; keep test edits proportional and directly tied to concrete bugs, coverage gaps, surviving mutants, or `MUSTFIX` limits. Add targeted cases in place, parameterize shared interface/concurrency suites instead of creating overlapping sibling `*_test.go` files, and use ordered slices rather than `range map` when constructing order-sensitive structures.

```go
// TestExportProgress verifies that Export reports start and completion progress in order.
func TestExportProgress(t *testing.T) {
  testCases := []struct {
    name         string
    items        []string
    wantProgress []float64
  }{
    // Reports zero, midpoint, and full completion for a two-item slice.
    {
      name:         "two items",
      items:        []string{"a", "b"},
      wantProgress: []float64{0.0, 0.5, 1.0},
    },
  }
  for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
      var gotProgress []float64
      recordProgress := func(pct float64) {
        gotProgress = append(gotProgress, pct)
      }

      err := Export(tc.items, recordProgress)

      if err != nil {
        t.Fatalf("Export() unexpected error: %v", err)
      }
      if diff := cmp.Diff(tc.wantProgress, gotProgress); diff != "" {
        t.Errorf("progress mismatch (-want +got):\n%s", diff)
      }
    })
  }
}
```
