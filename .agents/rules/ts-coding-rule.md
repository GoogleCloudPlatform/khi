---
trigger: glob
globs: **/*.ts
---

# TypeScript Coding Rules

When developing or modifying TypeScript code in KHI, adhere to these rules alongside `.agents/rules/comment-guideline.md` and `.agents/skills/backward_compatibility_policy/SKILL.md`.

## General TypeScript Rules

- **Comments & TSDoc**: Follow `comment-guideline.md` for all public types, functions, and methods. Keep every TSDoc (`/** ... */`) and inline comment strictly aligned with current reactivity (`signal()` / `input()` vs RxJS), timestamp units (`bigint` nanoseconds vs `*Ms`), nullability, and error/fallback behavior.
- **Identifier Naming**: Choose domain-descriptive names matching exact unit, boolean polarity, mutability, and side effects (e.g., never name a `bigint` nanosecond timestamp `timestampMs` or a mutating method `isValid()` / `get*()`). Outside narrow idioms (`i`/`j` in tiny loops, `el`/`ev` in one-line DOM callbacks, `x`/`y`/`w`/`h` in 2D canvas geometry), avoid cryptic placeholders (`d`, `m`, `tmp`, `val`, `data`, `res`, `proc`, `handle`, `cb`).
- **Types & Style**:
  - Test files must be named `A.spec.ts` when `A.ts` exists (do not organize specs by scenario).
  - Do NOT use `any`.
  - Make fields `readonly` wherever possible.
  - Define an `enum` rather than a union of string literal types.
  - Use absolute imports from the project base path (e.g., `src/app/...`).
  - Default timestamps to `bigint` in nanoseconds (ns); append the unit name for any other unit (e.g., `timestampMs`).
  - Prefix private fields with `_` **only** when a getter/setter with the same name exists.
  - Do NOT prefix interface names with `I`.
  - Do NOT use `as unknown as X` type assertions or object literal types (define an `interface` instead) in non-test code.
  - Do NOT reference types via property lookup (`Type['field']`); import and use the official type name directly.
- **No Backward Compatibility**: Internal TypeScript code requires no deprecated wrappers, unused optional parameters, or fallback shims; update all callers directly (`backward-compatibility-policy`).

## Verification & Quality Checks

- Run `make format-web`, `make lint-web`, `make test-web`, and `make build-storybook` before asking the user to verify (launch long-running checks in parallel when useful).
- Run `make qa-fast QA_LANG=ts` and `make qa-slow QA_LANG=ts` (add `USE_CONTAINER=true` when Chrome is inside the builder container, `CODEQUALITY_DECISIONS=true` to evaluate `scripts/codequality/rules/ts_*.yaml`, and `CODEQUALITY_SIMILARITY=true` for Vertex AI AST similarity advisories) to inspect differential line coverage, mutation score, and redundant specs on changed files.
- **Handling `qa-slow-advisory` Findings (`func_uncovered_lines`, `func_surviving_mutants`, `test_redundancy`)**:
  - **Never** delete legitimate defensive guards or add `CODEQUALITY_ALLOW` to silence advisories; delete or simplify only genuinely dead/unreachable branches.
  - Because Karma runs in Headless Chromium, **never** ignore DOM layout or scroll math as untestable. Extract non-trivial calculations into pure `*calculator.ts` / `*.viewmodel.ts` helpers or strengthen assertions in `*.spec.ts`.
  - For `test_redundancy` (`SAME PATH` / `NO OUTPUT CHECK` such as boilerplate `it('should create', () => expect(component).toBeTruthy())`), consolidate pure equivalence-class duplicates you introduced or assert distinct outputs rather than deleting pre-existing tests.

## TypeScript Unit Testing Practices (`*.spec.ts`)

1. **Mandatory Spec Explanation Comments**: Place a concise `// ... .` comment immediately above **every** `it(...)` block explaining the scenario/preconditions and expected behavior, and ensure setup, trigger, and `expect(...)` genuinely verify it.
2. **Linear Arrange, Act, Assert (AAA)**: Separate Arrange, Act, and Assert with blank lines (add `// Arrange.`, `// Act.`, `// Assert.` when multi-block). Test **one behavior per spec**; never chain multiple `Act -> Assert` sequences or compute expected values with `if`/`else` or `for` inside `it(...)`.
3. **Testing Callbacks and Subscriptions**: Never place `expect(...)` or `fail(...)` inside callbacks, `.subscribe(...)` handlers, or event listeners—if the callback never fires, the spec passes silently. Record emissions in Arrange (or use `jasmine.createSpy(...)`), trigger in Act, and assert **outside** after Act.

```ts
// GOOD: preceding comment, linear AAA, and subscription recorded outside Assert.
// Emits the dragged time range once when the pointer drag completes on the ruler.
it('emits selected time range on pointer up', () => {
  const emittedRanges: TimeRangeNs[] = [];
  component.timeRangeSelected.subscribe((range) => emittedRanges.push(range));

  triggerRangeDrag(fixture, 100, 200);

  expect(emittedRanges).toEqual([{ startNs: 100n, endNs: 200n }]);
});
```

## Angular Coding Rules

- Use Angular signals rather than RxJS for component state.
- Define templates and styles in separate files (never inline in `@Component`).
- Import `KHIIconRegistrationModule` when a component uses `MatIconModule`.
- Use `input()` / `input.required()`, `output()`, `model()`, `signal()`, and `computed()` instead of `@Input` / `@Output`.
- To expose an enum type to the template, declare `protected readonly` before any other fields, methods, or constructor.

## Smart-Dumb Component Architecture

Organize each feature or complex component as follows:

```text
foo/
  components/           # Dumb components (UI rendering and event emission only)
    foo-layout.component(.ts|.scss|.html|.spec.ts|.stories.ts)
    bar.component(.ts|.scss|.html|.spec.ts|.stories.ts)
  types/                # Feature-local types and ViewModels
  foo-smart.component(.ts|.scss|.html)
```

1. **Smart Components**:
   - Manage state and data fetching; allowed to depend on Angular Services.
   - **MUST NOT** contain elements other than layout components in their templates.
   - **MUST NOT** contain styles other than `:host { display: contents }`.
2. **Dumb Components**:
   - Render UI and propagate events via `input()`, `model()`, and `output()` only (**MUST NOT** depend on Angular Services).
   - **MUST** include a `*.stories.ts` file to verify visual states independently.
   - **DO NOT** use `-dumb` or `-ui` suffixes in component names; name components semantically (e.g., `user-profile`, `data-table`).
   - **Layout Components** are dumb components responsible for structural layout of other dumb components and standard HTML elements.
