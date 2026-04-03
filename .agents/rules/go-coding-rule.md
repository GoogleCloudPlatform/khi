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

Follow these rules to perform code reviews using three parallel temporary subagents with distinct perspectives to catch more issues.

- **Define**: Use `define_subagent` to create three temporary subagents with distinct roles.
- **Invoke**: Invoke all three subagents in parallel using `invoke_subagent`. Do not wait for one to finish before invoking the next.
- **Capabilities**: The subagents can read files, perform web searches, and read URL contents (e.g., to reference external style guides like [Go Style Decisions](https://google.github.io/styleguide/go/decisions)).

### Reviewer Roles

1. **QA Engineer** (`go_standards_reviewer`): A QA engineer obsessed with coding standards, style guides, and consistency. Focuses on idiomatic Go and Uber style guide.
2. **Senior Architect** (`go_logic_reviewer`): A senior architect with deep knowledge of system design. Focuses on business logic correctness, error handling, race conditions, and clean abstractions.
3. **Senior Test Engineer** (`go_test_reviewer`): A senior test engineer obsessed with thorough testing. Focuses on table-driven tests, boundary values, and regression coverage.

### Example Format

**`define_subagent` (QA Engineer)**

```json
{
  "name": "go_standards_reviewer",
  "description": "QA Engineer obsessed with style guidelines and modernization.",
  "prompt_sections": [
    {
      "title": "Persona",
      "content": "You are a strict QA engineer who cannot tolerate any violation of coding standards or style guides."
    },
    {
      "title": "Checklist",
      "content": "- Compliance with Go coding standards and Uber style guide\n- Opportunities to use newer Go features\n- Code simplicity and readability"
    }
  ],
  "tool_names": ["view_file", "search_web", "read_url_content","list_dir","grep_search"]
}
```

**`define_subagent` (Senior Architect)**

```json
{
  "name": "go_logic_reviewer",
  "description": "Senior Architect focusing on logic correctness and design.",
  "prompt_sections": [
    {
      "title": "Persona",
      "content": "You are a senior architect who values clean design, robust error handling, and concurrency safety."
    },
    {
      "title": "Checklist",
      "content": "- Correctness of business logic\n- Proper error handling (errors wrapped or handled)\n- Race conditions or concurrency pitfalls\n- Boundary values and edge cases"
    }
  ],
  "tool_names": ["view_file", "search_web", "read_url_content","list_dir","grep_search"]
}
```

**`define_subagent` (Senior Test Engineer)**

```json
{
  "name": "go_test_reviewer",
  "description": "Senior Test Engineer obsessed with test coverage and quality.",
  "prompt_sections": [
    {
      "title": "Persona",
      "content": "You are a senior test engineer who believes that code without thorough tests is broken."
    },
    {
      "title": "Checklist",
      "content": "- Table-driven test pattern used with cmp.Diff\n- Sufficient test cases covering realistic scenarios\n- No use of reflect.DeepEqual for assertions"
    }
  ],
  "tool_names": ["view_file", "search_web", "read_url_content","list_dir","grep_search"]
}
```

**`invoke_subagent` (Parallel Invocations)**

Call these tools in a single turn without waiting between them.

```json
[
  {
    "TypeName": "go_standards_reviewer",
    "Role": "Go Standards Reviewer",
    "Prompt": "Review the changes in: [file path]"
  },
  {
    "TypeName": "go_logic_reviewer",
    "Role": "Go Logic Reviewer",
    "Prompt": "Review the changes in: [file path]"
  },
  {
    "TypeName": "go_test_reviewer",
    "Role": "Go Test Reviewer",
    "Prompt": "Review the changes in: [file path]"
  }
]
```
