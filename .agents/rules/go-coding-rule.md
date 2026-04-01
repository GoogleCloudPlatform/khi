---
trigger: glob
globs: **/*.go
---

# KHI Go Standards

When developing or modifying Go code in the KHI project, you **must** adhere to the following rules and best practices.

## 1. General Coding Rules

1. **Build Verification**: Before running tests or submitting changes, you must always verify that your code compiles successfully.
   - Run `make build-go` to ensure there are no compilation errors across the backend.
2. **Review Verification**: Before asking the user to verify your changes, you MUST invoke a subagent to review your code. See Section 3 for the procedure. **If you make any modifications based on the review, you MUST run the review again to verify the changes.**
3. **Test Verification**: Run `go test` with appropriate filter to run tests only for changed parts first. But make sure you run `make test` before asking user to verify.
4. **Formatting and Linting**:
   - Run `make lint-go` if applicable, and ensure no new linting errors are introduced.
5. **Comments**:
   - Use `godoc`-style comments for all public types, functions, and methods.
6. **Implementing Interface**
   - Add `var _ Interface = &Implementation{};` after the type definition to show that it's implementing the interface explicitly.

## 2. Testing Practices

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

## 3. Subagent Review Guidelines

> [!IMPORTANT]
> **DO NOT FORGET** to invoke the subagent for code review after making changes. You must complete the subagent review before asking the user to verify your implementation.

Follow these rules to perform code reviews using a temporary subagent.

- **Define**: Use `define_subagent` to create a temporary subagent.
- **Invoke**: Pass the modified file paths to the subagent during `invoke_subagent`.
- **Capabilities**: The subagent can read files and perform web searches (e.g., to reference external style guides like [Go Style Decisions](https://google.github.io/styleguide/go/decisions)).

### Review Checklist Focus

The subagent must verify:

- Compliance with Go coding standards.
- Opportunities to simplify or shorten code using newer Go features.
- Any duplicated implementations.
- Sufficiency of test cases covering realistic and practical scenarios.

### Example Format

**`define_subagent`**

```json
{
  "name": "temp_go_reviewer",
  "description": "Reviews Go code against project standards.",
  "prompt_sections": [
    {
      "title": "Checklist",
      "content": "- Compliance with Go coding standards\n- Use of newer Go features\n- Duplicated implementations\n- Sufficient realistic test coverage"
    }
  ],
  "tool_names": ["view_file", "search_web"]
}
```

**`invoke_subagent`**

```json
{
  "TypeName": "temp_go_reviewer",
  "Role": "Go Code Reviewer",
  "Prompt": "Review the changes in: [file path]"
}
```
