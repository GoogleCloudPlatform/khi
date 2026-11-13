---
trigger: glob
globs: **/*.go
---

# Go Coding rules

When developing or modifying Go code in the KHI project, you **must** adhere to the following rules and best practices.

## Verifications

1. **Build Verification**: Before running tests or submitting changes, you must always verify that your code compiles successfully.
   - Run `make build-go` to ensure there are no compilation errors across the backend.
2. **Test Verification**: Run `go test` with an appropriate filter to run tests only for changed parts first. Run `make codequality-fast` during inner-loop refactoring for fast (<1s) AST complexity/size/naming ratchet feedback, and run `make codequality-go` (or `make codequality`) to verify all Go quality rules and ratchets (AST structural budgets, uncovered lines, surviving mutants, and redundant tests) on changed code. When a mutant survives or a quality rule fails:
   - First check if the branch is redundant or guarded by an existing invariant — **never** suppress invariant, defensive, or unreachable conditions with `CODEQUALITY_ALLOW`; simplify or delete the redundant branch instead.
   - Strengthen existing assertions (`cmp.Diff` on whole structs, boundary inputs) rather than adding mock-heavy boilerplate tests.
   - Only for genuine non-deletable `HARD_LIMIT` cases, suppress with a required rule ID, category, and explanation (`// CODEQUALITY_ALLOW(<rule_id>, <category>): <reason>` or `// CODEQUALITY_ALLOW(func_surviving_mutants, <category>, <operator>): <reason>`, where `<category>` is one of `dispatch`, `protocol`, `equivalent`, `perf`, `observability`, `external` as allowed by the rule).
   - Make sure you run `make test-go` before asking the user to verify.
3. **Formatting and Linting**:
   - Run `make format-go` to ensure standard Go formatting.
   - Run `make lint-go` and ensure zero linting errors (`depguard`, `errorlint`, `forbidigo`, `nolintlint`, `staticcheck`, `govet`, etc.). Naked `//nolint` comments without a specific linter name and explanation are forbidden.
4. If you make any corrections during a verification phase, you MUST restart the verification from the beginning for that phase.

## Go Quality Rules & Ratchet Budgets

`make codequality-go` (and `make codequality-fast` for Stage-1 AST rules) enforces quantitative budgets on newly added/compliant Go units and a **strict no-worsening ratchet (`after <= before`)** on legacy units that already exceed a budget:

- **Production Functions**:
  - Cognitive complexity (`func_cognitive`): `<= 15` (flat `switch` counts as `+1`; deep `if/for` nesting and mixed `&&`/`||` chains incur compounding penalties).
  - Max control-flow nesting (`func_nesting`): `<= 4` (`else if` does not increase nesting; prefer early-return guard clauses).
  - Function size: `<= 60` code lines (`func_code_lines`, comments and blank lines excluded — **never** delete GoDoc comments to shrink line counts) and `<= 40` AST statements (`func_statements`).
  - Signature bounds: `<= 5` parameters (`func_params`), `<= 3` return values (`func_results`), and **no control-flow `bool` flag parameters** on `>= 3`-parameter functions (`control_flag_param`). Do **not** shred complex functions into 6+ parameter pass-through helpers; group cohesive state into domain structs/methods instead.
  - Duplication (`code_clone`): `<= 15` consecutive normalized AST statements duplicated across production functions (`>= 16` triggers a violation). Extract shared abstractions instead of copy-pasting.
  - Uncovered substantive lines (`func_uncovered_lines`): `0` uncovered substantive executable lines in touched functions (idiomatic `if err != nil { return ... }` propagation, `defer`, `panic`, diagnostic log statements, and `func main()` entry wrappers are excluded automatically; legacy functions enforce `after <= before` via lazy Go `-overlay` comparison).
  - Surviving mutants (`func_surviving_mutants`): `0` surviving mutants in touched functions (legacy functions enforce `after <= before` via lazy Go `-overlay` comparison).
- **Files & Naming**:
  - File size: `<= 450` code lines for production `.go` files (`file_code_lines`) and `<= 800` code lines for `*_test.go` files (`test_file_code_lines`), excluding comments, blanks, and `import` blocks.
  - Catch-all filenames (`catchall_filename`): Newly created files must **never** be named `util(s).go`, `helper(s).go`, `common.go`, `misc.go`, `shared.go`, or `*_util(s).go` / `*_helper(s).go` / `*_misc.go`. Name files after their concrete domain concept or pipeline stage.
- **Test Code (`*_test.go`)**:
  - `t.Run` loop body cognitive complexity (`trun_cognitive`): `<= 8` (declarative `testCases` table literals do not count toward this limit; keep `t.Run` bodies linear and assert with `cmp.Diff(tc.want, got)`).
  - Test helper cognitive complexity (`test_helper_cognitive`): `<= 15`.
  - Redundant subtests (`test_redundancy`): `0` subsumed duplicate (`SAME PATH`) or assertion-free (`NO OUTPUT CHECK`) subtests in touched `Test*` functions.

## General Coding Rules

1. **Comments**:
   - All comments must be written in English.
   - Add GoDoc comments for all public types, functions, and methods.
   - Add GoDoc comments for private types, functions, and methods when their names are not self-explanatory or usage is not intuitive.
   - Add `var _ Interface = (*Implementation)(nil);` after the type definition to show that it's implementing the interface explicitly.

2. **Error and Test Assertion Messages**:
   - Do not capitalize the first character of messages in `fmt.Errorf`, `t.Error`, `t.Errorf`, `t.Fatal`, `t.Fatalf`, etc. (e.g., use lowercase "failed to ..." instead of "Failed to ...").
   - This rule does not apply if the message starts with a capitalized name, such as a method name or proper acronym (e.g. `MyFunction() mismatch (-want +got)`).

3. **Backward Compatibility and Refactoring**:
   - Internal Go code in KHI requires **no backward compatibility**. When modifying functions, methods, interfaces, or structs, do **NOT** retain deprecated shims, wrappers, or aliases, and do **NOT** introduce unnecessary variadic parameters or defensive nil fallbacks solely to preserve old call sites.
   - Always update all call sites across the monorepo directly.
   - Updating test call sites, constructors, and assertions to adapt to refactored signatures is expected and required.
   - Refer to the `backward-compatibility-policy` skill for concrete anti-patterns and details.

4. **Multiline String Literals**:
   - When a string literal contains line breaks, such as YAML, JSON, or Markdown fixtures and expected multi-line output in tests, write it as a raw string literal enclosed in backquotes that keeps the real line breaks.
   - Do **NOT** embed `\n` escape sequences in the middle of a string literal, and do not join fixed lines with `"\n"` to build multi-line content.
   - Exceptions:
     - A single trailing `\n` at the end of a format string (e.g., `fmt.Fprintf(w, "%s\n", line)`).
     - The `\n` that puts a `cmp.Diff` result on its own line in a test failure message (e.g., `t.Errorf("MyFunction() mismatch (-want +got):\n%s", diff)`).
     - Content that itself contains backquotes, which a raw string literal cannot hold.

```go
// BAD: escaped line breaks hide the structure of the content.
const nodeYAML = "apiVersion: v1\nkind: Node\nmetadata:\n  name: node-1\n"

// GOOD: a raw string literal keeps the real line breaks.
const nodeYAML = `apiVersion: v1
kind: Node
metadata:
  name: node-1
`
```

## Testing Practices

1. **Table-Driven Tests**: Tests must be written using the table-driven testing pattern. Define a slice of anonymous structs representing the test cases, and iterate over them using `t.Run()`.
2. **Assertions and Diffs**:
   - **Scalar Values**: For single scalar values (e.g. `int`, `bool`, `float`, single-line `string`, enum), compare directly using equality operators (`==` or `!=`). Report test failures with `t.Errorf` in `got, want` format (e.g., `t.Errorf("got %v, want %v", got, tc.want)` or `t.Errorf("MyFunction(%v) = %v, want %v", tc.input, got, tc.want)`). Do **NOT** use `cmp.Diff` for scalar values.
   - **Complex Structs and Collections**: **MUST USE** `github.com/google/go-cmp/cmp` (`cmp.Diff`) when comparing complex structs, slices, maps, or nested structures. Output `cmp.Diff` when an assertion fails to clearly communicate the mismatch (`-want +got`).
   - **Multiline Strings Exception**: `cmp.Diff` is permitted for a single string variable if and only if the string spans multiple lines (e.g., generated YAML, JSON, or multiline text blocks) where line-by-line diffing is beneficial.
   - **DO NOT USE** the `reflect` package for test assertions (e.g., `reflect.DeepEqual`). Always prefer `cmp.Diff` for complex types.
3. **Running Tests**:
   - Executing `make test-go` runs all backend tests.
   - For iterating on specific tests, `go test ./pkg/path/to/test -run TestName` is acceptable, provided a full `make test-go` ensures no regressions before finalizing work.
4. **Test File Naming**: When adding tests for a file `A.go`, the test file **must** be named `A_test.go`. Do not create independent test files that group tests from multiple files.
5. **Structuring `want` in Table-Driven Tests**:
   - **Never Derive `want` from Inputs in the Test Loop**: Every expected value must be declared explicitly as a literal in the test case table. Never compute expected values from input fields inside the `for _, tc := range testCases` body (e.g., `wantMounted := !tc.jobMode`), as this duplicates production logic in the test.
   - **Whole-Struct `want` for Data Outputs**: When testing a function that returns a struct or multiple data values, do not split fields into separate `wantLen`, `wantStart`, `wantEnd`, or `wantID` fields on the table struct. Define a single `want ReturnType` (or a local outcome struct) and compare it with a single `cmp.Diff(tc.want, got)`.
   - **Multiple Side Effects or Staged Assertions**: When a test verifies multiple independent side effects or uses an early expectation to gate subsequent setup (e.g., checking whether a component was initialized before starting a test server), keep all expected values in the table case definition (either as explicit `want*` fields like `wantMCP` and `wantRenderTool`, or grouped in a nested `want` struct).

> [!IMPORTANT]
> Typical table-driven tests should follow these patterns:
>
> ### 1. Scalar Comparison (e.g. `int`, `bool`, single-line `string`)
>
> ```go
> func TestCountItems(t *testing.T) {
>  testCases := []struct {
>   name  string
>   input string
>   want  int
>  }{
>   {
>    name:  "single item",
>    input: "foo",
>    want:  1,
>   },
>  }
>  for _, tc := range testCases {
>   t.Run(tc.name, func(t *testing.T) {
>    got := CountItems(tc.input)
>    if got != tc.want {
>     t.Errorf("CountItems(%q) = %d, want %d", tc.input, got, tc.want)
>    }
>   })
>  }
> }
> ```
>
> ### 2. Complex Struct Comparison
>
> ```go
> import (
>  "testing"
>
>  "github.com/google/go-cmp/cmp"
> )
>
> func TestParseUser(t *testing.T) {
>  testCases := []struct {
>   name  string
>   input string
>   want  User
>  }{
>   {
>    name:  "valid user",
>    input: "id: 1, name: Alice",
>    want: User{
>     ID:   1,
>     Name: "Alice",
>    },
>   },
>  }
>  for _, tc := range testCases {
>   t.Run(tc.name, func(t *testing.T) {
>    got := ParseUser(tc.input)
>    if diff := cmp.Diff(tc.want, got); diff != "" {
>     t.Errorf("ParseUser() mismatch (-want +got):\n%s", diff)
>    }
>   })
>  }
> }
> ```
>
> ### 3. Multiline String Comparison (Exception)
>
> ```go
> if diff := cmp.Diff(tc.wantYAML, gotYAML); diff != "" {
>  t.Errorf("RenderYAML() mismatch (-want +got):\n%s", diff)
> }
> ```
>
> ### 4. Anti-Patterns to Avoid in Table-Driven Tests
>
> ```go
> // BAD 1: Splitting a returned struct into field-by-field want* scalars
> testCases := []struct {
>  name      string
>  items     []int
>  wantLen   int
>  wantStart int
>  wantEnd   int
> }{...}
>
> // BAD 2: Computing an expected value from an input field inside the test loop
> for _, tc := range testCases {
>  t.Run(tc.name, func(t *testing.T) {
>   _ = Init(ctx, tc.jobMode)
>   wantMCP := !tc.jobMode // Re-implements production branching logic!
>   if got := HasMCP(ctx); got != wantMCP { ... }
>  })
> }
> ```
