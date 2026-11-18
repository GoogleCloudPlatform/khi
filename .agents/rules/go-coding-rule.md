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
  - Duplication (`code_clone`): `<= 15` consecutive normalized AST statements duplicated across production functions (`>= 16` triggers a violation). When opted in via `CODEQUALITY_SIMILARITY=true` (with optional billing project `CODEQUALITY_SIMILARITY_PROJECT=<project-id>`; enabled in Cloud Build CI), `make codequality` / `make codequality-fast` also emits non-blocking `Similarity` advisories (`gemini-embedding-2` over AST-normalized Go functions, methods, and task closures) whenever newly added units match existing helpers — inspect and reuse matching `[shared]` or domain helpers instead of duplicating logic.
  - Semantic Go Code & Test Rules (`CODEQUALITY_DECISIONS=true` / `make codequality-decisions`): Evaluates declarative YAML rules in `scripts/codequality/rules/*.yaml` via the Gemini Decisions API (`gemini-decision-1`), including `go_code_quality` (comment-to-logic alignment and identifier naming accuracy) and `go_test_quality` (test case explanation alignment, Arrange-Act-Assert structure, callback testing, and clear test naming).
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

1. **Comments and GoDoc (Strict Logic Alignment)**:
   - All comments must be written in English.
   - Add GoDoc comments on all exported symbols (`const`, `var`, `type`, `func`, and methods) AND on all exported struct fields and exported interface methods of exported types.
   - Start GoDoc comments on exported declarations with the exact declared symbol name followed by a third-person singular verb (e.g., `// ParseFilter parses ...`), and end every GoDoc/declaration comment with a period (`.`).
   - Add GoDoc comments for private types, functions, and methods when their names are not self-explanatory or usage is not intuitive.
   - **Comments Must Match Implementation Logic**: Every GoDoc and inline comment must accurately reflect the code's actual behavior, boundary conditions, data structures, synchronization primitives, and error/fallback contracts. Whenever you refactor implementation details (such as replacing `sync.Map` with a custom map or changing fallback behavior to return an error), update or remove affected comments in the same change.

2. **Identifier Naming (Functions, Parameters, and Variables)**:
   - Choose function, method, parameter, field, and local variable names that clearly communicate their domain role and match how they are actually used:
     - **Avoid Cryptic or Overly Short Names**: Outside narrow, conventional short-scope Go idioms (`ctx` for `context.Context`, `err` for `error`, `t`/`b` in tests/benchmarks, `r`/`w` for `io.Reader`/`io.Writer`, concise receiver names, and `i` in a tiny index loop), do **NOT** use opaque 1–2 letter or generic placeholder names (`d`, `m`, `v`, `tmp`, `val`, `res`, `data`, `proc`, `doWork`, `handle`) that obscure a symbol's role across multi-line logic.
     - **Match Name to Actual Usage**: Never give a variable, parameter, or function a name that contradicts its unit, polarity, mutability, filtering stage, or side effects (for example, naming a millisecond duration `timeoutSec`, naming filtered records `rawLogs`, naming a failure flag `isReady`, or naming a state-mutating function `validate*` or `check*`).
     - **Avoid Package/Receiver Stutter**: Omit redundant package or receiver prefixes on functions and methods (e.g., `yamlconfig.Parse` instead of `yamlconfig.ParseYAMLConfig`, `(c *Config) WriteTo` instead of `WriteConfigTo`), use noun-like names without `Get` for pure accessors (`JobName`), and use verb-like names for operations with side effects or computation (`WriteDetail`, `ParseFilter`).

```go
// BAD: cryptic short names and misleading unit/side-effect names.
func check(d int64, m map[string]string) error {
 timeoutSec := time.Duration(d) * time.Millisecond // Unit contradicts "Sec"!
 delete(m, "expired")                              // Mutates input map despite "check" name!
 return nil
}

// GOOD: names express domain role, exact unit, and mutating behavior.
func pruneExpiredEntries(timeoutMillis int64, entriesByKey map[string]string) time.Duration {
 timeout := time.Duration(timeoutMillis) * time.Millisecond
 delete(entriesByKey, "expired")
 return timeout
}
```

1. **Imports and Declaration Ordering**:
   - **Import sections (`gci`)**: Group imports in three blank-line-separated sections: (1) Go standard library, (2) third-party/external packages, and (3) `github.com/GoogleCloudPlatform/khi/...` internal packages.
   - **Top-level declaration order (`decorder` + `decl-visibility-order`)**: Order top-level declarations strictly as `const -> var -> type -> func`. Within each kind (`const`, `var`, `type`), place exported declarations before unexported declarations (never reorder specs inside a single `const (...)` block where `iota` is used).
   - **Interface implementation assertions**: Place compile-time interface checks (`var _ Interface = (*Implementation)(nil)`) in the `var` section *before* `type` declarations (not after the `type` definition).
   - **Function and method order (`funcorder` + `decl-visibility-order`)**: Order functions as: (1) `init()`, (2) exported standalone functions and exported-receiver groups (`New*`/`new*` constructors -> exported methods -> unexported methods on the exported receiver), and (3) unexported-receiver groups and unexported standalone helpers.

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

1. **Table-Driven Tests & Mandatory Test Case Explanations**:
   - Write unit tests using the table-driven testing pattern: define a slice of anonymous structs representing the test cases and iterate over them using `t.Run()`.
   - **Explain Every Test Case with a Preceding Comment**: Place a concise `// ...` comment (ending with `.`) immediately above **every** table case literal `{ ... }` stating the scenario/preconditions and expected behavior. For standalone `Test*` functions without a table, place a `// TestXxx verifies that ...` comment immediately above `func TestXxx(t *testing.T)`.
   - Ensure the test case's inputs, `Act` call, and `Assert` expectations genuinely test the exact scenario described by its comment and cannot pass coincidentally (for example, when testing `Clear()`, populate an item first so the container is not already empty).
2. **Test Structure: Arrange, Act, Assert (AAA)**:
   - Structure every unit test (and every `t.Run` subtest body) into three distinct, linear phases:
     - **Arrange**: Set up only the minimal inputs, dependencies, and system under test needed to elicit the target behavior. Precondition checks on fixture setup (such as `if err != nil { t.Fatalf(...) }` when creating temp files) belong in Arrange because a failure indicates a broken test setup rather than a bug in the system under test.
     - **Act**: Invoke the single target function or method under test.
     - **Assert**: Verify the returned output or resulting state against expected values.
   - Visually separate Arrange, Act, and Assert with blank lines. When any phase spans multiple blocks or contains internal blank lines, add explicit `// Arrange.`, `// Act.`, and `// Assert.` comments so the `Act` step stands out immediately.
   - **One Behavior per Test Case**: Do **NOT** chain multiple `Act -> Assert -> Act -> Assert` sequences for independent behaviors inside a single test or subtest, and keep test bodies straight-line without `if/else` or `for` logic computing expected values.
3. **Testing Callbacks**:
   - When testing functions, methods, or types that invoke a callback (`func(...)`, listener, hook, or progress reporter), **NEVER** embed behavioral assertions (`t.Error`, `t.Errorf`, `t.Fatal`, `cmp.Diff`) directly inside the callback body. If a defect prevents the callback from ever being invoked, assertions inside the callback are silently skipped and the test passes coincidentally; in-callback assertions also mix Assert into Arrange.
   - Instead:
     - **Arrange**: Define the callback to **record** each invocation and its arguments into local test state (such as a `gotCalls` slice or struct variable, guarded by `sync.Mutex`, `atomic`, or a channel when invoked concurrently).
     - **Act**: Call the function or method under test.
     - **Assert**: **Outside** the callback (after `Act`), assert on the recorded state to verify whether the callback was invoked, how many times / in what order, and with what arguments.
4. **Assertions and Diffs**:
   - **Scalar Values**: For single scalar values (e.g. `int`, `bool`, `float`, single-line `string`, enum), compare directly using equality operators (`==` or `!=`). Report test failures with `t.Errorf` in `got, want` format (e.g., `t.Errorf("got %v, want %v", got, tc.want)` or `t.Errorf("MyFunction(%v) = %v, want %v", tc.input, got, tc.want)`). Do **NOT** use `cmp.Diff` for scalar values.
   - **Complex Structs and Collections**: **MUST USE** `github.com/google/go-cmp/cmp` (`cmp.Diff`) when comparing complex structs, slices, maps, or nested structures. Output `cmp.Diff` when an assertion fails to clearly communicate the mismatch (`-want +got`).
   - **Multiline Strings Exception**: `cmp.Diff` is permitted for a single string variable if and only if the string spans multiple lines (e.g., generated YAML, JSON, or multiline text blocks) where line-by-line diffing is beneficial.
   - **DO NOT USE** the `reflect` package for test assertions (e.g., `reflect.DeepEqual`). Always prefer `cmp.Diff` for complex types.
5. **Running Tests**:
   - Executing `make test-go` runs all backend tests.
   - For iterating on specific tests, `go test ./pkg/path/to/test -run TestName` is acceptable, provided a full `make test-go` ensures no regressions before finalizing work.
6. **Test File Naming**: When adding tests for a file `A.go`, the test file **must** be named `A_test.go`. Do not create independent test files that group tests from multiple files.
7. **Structuring `want` in Table-Driven Tests**:
   - **Never Derive `want` from Inputs in the Test Loop**: Every expected value must be declared explicitly as a literal in the test case table. Never compute expected values from input fields inside the `for _, tc := range testCases` body (e.g., `wantMounted := !tc.jobMode`), as this duplicates production logic in the test.
   - **Whole-Struct `want` for Data Outputs**: When testing a function that returns a struct or multiple data values, do not split fields into separate `wantLen`, `wantStart`, `wantEnd`, or `wantID` fields on the table struct. Define a single `want ReturnType` (or a local outcome struct) and compare it with a single `cmp.Diff(tc.want, got)`.
   - **Multiple Side Effects or Staged Assertions**: When a test verifies multiple independent side effects or uses an early expectation to gate subsequent setup (e.g., checking whether a component was initialized before starting a test server), keep all expected values in the table case definition (either as explicit `want*` fields like `wantMCP` and `wantRenderTool`, or grouped in a nested `want` struct).

> [!IMPORTANT]
> Typical table-driven and callback tests should follow these patterns:
>
> ### 1. Scalar Comparison with Case Explanation & AAA Structure
>
> ```go
> func TestCountItems(t *testing.T) {
>  testCases := []struct {
>   name  string
>   input string
>   want  int
>  }{
>   // Counts a single non-empty token without delimiters as one item.
>   {
>    name:  "single item",
>    input: "foo",
>    want:  1,
>   },
>   // Returns zero when the input string is empty.
>   {
>    name:  "empty input",
>    input: "",
>    want:  0,
>   },
>  }
>  for _, tc := range testCases {
>   t.Run(tc.name, func(t *testing.T) {
>    got := CountItems(tc.input)
>
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
>   // Extracts numeric ID and display name from a valid key-value pair string.
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
>
>    if diff := cmp.Diff(tc.want, got); diff != "" {
>     t.Errorf("ParseUser() mismatch (-want +got):\n%s", diff)
>    }
>   })
>  }
> }
> ```
>
> ### 3. Testing Callbacks (Record in Arrange, Assert Outside)
>
> ```go
> // BAD: assertion inside callback silently passes if Export never invokes onProgress!
> func TestExportProgress_Bad(t *testing.T) {
>  _ = Export(items, func(pct float64) {
>   if pct < 0 || pct > 1 {
>    t.Errorf("pct = %v, want in [0, 1]", pct)
>   }
>  })
> }
>
> // TestExportProgress verifies that Export reports start and completion progress in order.
> func TestExportProgress(t *testing.T) {
>  var gotProgress []float64
>  recordProgress := func(pct float64) {
>   gotProgress = append(gotProgress, pct)
>  }
>
>  err := Export([]string{"a", "b"}, recordProgress)
>
>  if err != nil {
>   t.Fatalf("Export() unexpected error: %v", err)
>  }
>  wantProgress := []float64{0.0, 0.5, 1.0}
>  if diff := cmp.Diff(wantProgress, gotProgress); diff != "" {
>   t.Errorf("progress updates mismatch (-want +got):\n%s", diff)
>  }
> }
> ```
>
> ### 4. Multiline String Comparison (Exception)
>
> ```go
> if diff := cmp.Diff(tc.wantYAML, gotYAML); diff != "" {
>  t.Errorf("RenderYAML() mismatch (-want +got):\n%s", diff)
> }
> ```
>
> ### 5. Anti-Patterns to Avoid in Unit Tests
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
>
> // BAD 3: Chaining multiple Act -> Assert -> Act -> Assert behaviors in one subtest
> t.Run("add and clear", func(t *testing.T) {
>  cache.Add("k", "v")
>  if got := cache.Len(); got != 1 { t.Errorf("got %d, want 1", got) }
>  cache.Clear()
>  if got := cache.Len(); got != 0 { t.Errorf("got %d, want 0", got) }
> })
> ```
