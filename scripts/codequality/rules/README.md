# Declarative Gemini Decisions Rules

This directory (`scripts/codequality/rules/`) stores declarative YAML rules evaluated by `scripts/codequality` via the Gemini Decisions API (`gemini-decision-1`).

Adding a new semantic check requires **no Go code changes**: create a `*.yaml` file in this directory and run `go test ./scripts/codequality/internal/decisioncheck/...` to validate its schema and templates offline.

## Built-in Rules

- **`go_code_quality.yaml`**: Verifies comment-to-logic alignment and clear, accurate function/parameter/variable naming on modified Go production files.
- **`go_refactor_safety.yaml`**: Verifies via diff hunks that modified Go production functions do not silently remove defensive guards, capacity pre-allocations, or lock/error invariants.
- **`go_test_quality.yaml`**: Verifies preceding test-case explanation comments and logic alignment, linear Arrange-Act-Assert structure, callback testing practices (recording inside callbacks and asserting outside), and clear naming on modified Go `*_test.go` files.
- **`ts_code_quality.yaml`**: Verifies TSDoc and inline comment-to-logic alignment and clear, accurate function/method/parameter/variable naming on modified TypeScript production files (`web/src/**/*.ts`).
- **`ts_test_quality.yaml`**: Verifies preceding spec explanation comments and logic alignment, linear Arrange-Act-Assert structure, callback and subscription testing practices (recording inside callbacks/`.subscribe(...)` and asserting outside), and clear naming on modified TypeScript `*.spec.ts` files.

## Running Rules Locally

Set `CODEQUALITY_DECISIONS_API_KEY` in `~/.config/khi/codequality.mk` (or `GEMINI_API_KEY` in your environment) and run:

```bash
# Run all enabled Gemini Decisions rules on current VCS diff
CODEQUALITY_DECISIONS=true make qa-fast-advisory

# Run a single rule on current VCS diff
go run ./scripts/codequality check --speed=fast --filter=advisory --decisions --decisions-rules=go_test_quality
```

Responses are cached in `~/.cache/khi/decisions/decisions_v1.db` keyed by the SHA-256 hash of the model, shared input, and rendered question JSON. Editing a rule YAML file or any target/reference file automatically invalidates stale cache entries.

## YAML Rule Schema

### 1. Top-Level Metadata

- `id` (required): Unique `snake_case` identifier.
- `name` (required): Short human-readable title.
- `description`: Summary of the invariant checked by the rule.
- `enabled`: `true` (default) or `false`.
- `enforcement`: `advisory` (default; reports warnings) or `blocking` (exits with code `1`).

### 2. `target` (Bundle Grouping)

Each resolved target bundle is sent once as shared `input` in a single `POST /v1beta/decisions` call:

- `kind` (required):
  - `go_logical_package`: Groups a Go package root together with its `impl/` and `internal/` subdirectories.
  - `changed_file`: Evaluates each matching modified file as an individual bundle (works for Go, TypeScript, Protobuf, Markdown, HTML, and config files).
- `include` / `exclude`: Slash-separated path globs supporting `*`, `?`, and `**` (`*.go` matches direct children only; `**/*_test.go` matches any depth).
- `require_files`: Relative files that must exist in a package directory for the rule to apply (for example `["README.md"]`).

### 3. `context` (Shared Input Assembly)

- `include_files` / `exclude_files`: Target-relative globs controlling which files are included in a `go_logical_package` bundle.
- `include_companion_go`: When `true` on `changed_file` targets ending in `_test.go`, automatically prepends the sibling `<base>.go` file if present.
- `include_companion_ts`: When `true` on `changed_file` targets ending in `.spec.ts`, automatically prepends the sibling `<base>.ts` file if present.
- `reference_files`: Repository-relative Markdown/doc files (such as `.agents/rules/description-rule.md`) prepended to the shared input from the trusted repository root.
- `include_diff`: Appends unified VCS diff hunks for the target when `true`.
- `line_numbers`: Prefixes file lines with 1-based line numbers (`true` by default).
- `include_go_ast_summary`: Appends a compact Go AST declaration and struct-field index when `true`.

### 4. `questions` and Outcome-Specific `fix_hint`

Each question supports `for_each` expansion and Go `text/template` variables:

| `for_each` Extractor | Expanded Scope | Template Variables |
| :--- | :--- | :--- |
| `bundle` (or omitted) | 1 question per bundle | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |
| `markdown_sections` | Each substantive Markdown heading section | `.Target`, `.File`, `.Heading`, `.Line`, `.StartLine`, `.EndLine` |
| `go_substantive_units` | `Module.Tasks` entries or Go source-file subsystems | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind` |
| `changed_go_decls` | Top-level production Go declarations overlapping the VCS diff | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |
| `changed_go_funcs` | Production Go functions and methods overlapping the VCS diff | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |
| `changed_go_test_funcs` | Go unit test functions (`Test*`) overlapping the VCS diff | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |
| `changed_go_test_cases` | Table-driven test cases and standalone `Test*` functions overlapping the VCS diff | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |
| `changed_ts_decls` | Production TypeScript declarations (classes, interfaces, enums, types, functions, and methods) overlapping the VCS diff | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |
| `changed_ts_funcs` | Production TypeScript functions and class methods overlapping the VCS diff | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |
| `changed_ts_specs` | Jasmine spec blocks (`it`, `fit`, `xit`) in `*.spec.ts` files overlapping the VCS diff | `.Target`, `.File`, `.Line`, `.StartLine`, `.EndLine`, `.Name`, `.Kind`, `.Doc` |

TypeScript extractors (`changed_ts_decls`, `changed_ts_funcs`, `changed_ts_specs`) parse ASTs via the Node.js TypeScript Compiler API (`web/node_modules/typescript`).

Supported `type` primitives and required `fix_hint` placement:

1. **`noul` (Yes/No)**:
   - Optional `criteria.true_description` and `criteria.false_description`.
   - Requires question-level `fix_hint` and `pass: { expect: true|false, min_probability: 0.50 }`.
2. **`choice` (Multiple Choice, 2..255 options)**:
   - `options` map where each key defines `description` and (for every non-passing option) an outcome-specific `fix_hint` explaining how to resolve that specific failure mode.
   - Requires `pass: { expected_choice: <key>, min_probability: 0.50 }` (or `allowed_choices`).
3. **`score` (Rubric Rating, 2..255 ordered levels)**:
   - `levels` list (`0..N-1`) where each failing level defines `description` and `fix_hint`.
   - Requires `pass: { min_score: <float> }` (and/or `max_score`).
