---
trigger: glob
globs: **/*.go
---

# Go Coding rules

When developing or modifying Go code in the KHI project, you **must** adhere to the following rules and best practices.

## Verifications

1. **Build Verification**: Before running tests or submitting changes, you must always verify that your code compiles successfully.
   - Run `make build-go` to ensure there are no compilation errors across the backend.
2. **Test Verification**: Run `go test` with an appropriate filter to run tests only for changed parts first. But make sure you run `make test-go` before asking the user to verify.
3. **Formatting and Linting**:
   - Run `make format-go` to ensure standard Go formatting.
   - Run `make lint-go` if applicable, and ensure no new linting errors are introduced.
4. If you make any corrections during a verification phase, you MUST restart the verification from the beginning for that phase.

## General Coding Rules

1. **Comments**:
   - All comments must be written in English.
   - Add GoDoc comments for all public types, functions, and methods.
   - Add GoDoc comments for private types, functions, and methods when their names are not self-explanatory or usage is not intuitive.
   - Add `var _ Interface = (*Implementation)(nil);` after the type definition to show that it's implementing the interface explicitly.

## Testing Practices

1. **Table-Driven Tests**: Tests must be written using the table-driven testing pattern. Define a slice of anonymous structs representing the test cases, and iterate over them using `t.Run()`.
2. **Assertions and Diffs**:
   - **MUST USE** `github.com/google/go-cmp/cmp` for complex comparisons and generating diffs. Show `cmp.Diff` when an assertion fails to clearly communicate the mismatch.
   - **DO NOT USE** the `reflect` package for test assertions (e.g., `reflect.DeepEqual`). Always prefer `cmp.Diff`.
3. **Running Tests**:
   - Executing `make test-go` runs all backend tests.
   - For iterating on specific tests, `go test ./pkg/path/to/test -run TestName` is acceptable, provided a full `make test-go` ensures no regressions before finalizing work.
4. **Test File Naming**: When adding tests for a file `A.go`, the test file **must** be named `A_test.go`. Do not create independent test files that group tests from multiple files.

> [!IMPORTANT]
> A typical table-driven test should look something like this:
>
> ```go
> import (
>  "testing"
>  "github.com/google/go-cmp/cmp"
> )
>
> func TestMyFunction(t *testing.T) {
>  testCases := []struct {
>   name     string
>   input    string
>   want string
>  }{
>   {
>    name:     "valid input",
>    input:    "foo",
>    want: "bar",
>   },
>  }
>  for _, tc := range testCases {
>   t.Run(tc.name, func(t *testing.T) {
>    got := MyFunction(tc.input)
>    if diff := cmp.Diff(tc.want, got); diff != "" {
>     t.Errorf("MyFunction() mismatch (-want +got):\n%s", diff)
>    }
>   })
>  }
> }
> ```
