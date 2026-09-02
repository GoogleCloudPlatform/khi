---
name: backward-compatibility-policy
description: Guidelines on backward compatibility in KHI, explaining why internal code requires no backward compatibility, identifying anti-patterns with concrete examples, and detailing the single exception for .khi files.
---

# Backward Compatibility Policy in KHI

This guide outlines the backward compatibility policy for Kubernetes History Inspector (KHI), explains why internal code requires no backward compatibility, illustrates common anti-patterns with concrete examples, and describes how to handle `.khi` file changes.

---

## 1. Core Principles

### KHI is an Integrated Monorepo Application

KHI is an end-to-end application comprising a Go backend and an Angular frontend. It is not a public library or a distributed microservice system with independent release cycles. All components are versioned, built, and shipped together.

### Zero Backward Compatibility for Internal Code

Internal Go/TypeScript functions, types, interfaces, methods, Connect/gRPC endpoints, and task Directed Acyclic Graph (DAG) nodes have **no external consumers**.

When modifying or refactoring internal code:

- **Never** retain backward-compatibility wrappers, aliases, or shims.
- **Never** introduce defensive fallbacks for hypothetical legacy callers.
- **Always** unify directly onto the new implementation. Update all call sites, usages, and tests across the repository in the same change.

### The Single Exception: `.khi` Files

The only data format that persists across KHI versions and user environments is the exported inspection dump file (`.khi` file format, serialized with Protocol Buffers). Backward compatibility is relevant **only** when modifying `.khi` serialization, deserialization, or schema structures.

### Explicit Discussion Required for `.khi` Changes

Even for `.khi` files, never make silent assumptions about backward compatibility. If a change affects `.khi` file persistence, you must explicitly raise and discuss the compatibility strategy with the user during the design and planning phase (`/plan`).

---

## 2. Anti-Patterns and Concrete Examples

### Anti-Pattern 1: Unnecessary Variadic Arguments to Avoid Updating 0-Arg Callers

When a function requires a new argument (e.g., going from 0 arguments to 1 argument), do not make the argument variadic (`...Option` or `...*Config`) merely to keep existing 0-argument callers compiling without modification.

**Bad:**

```go
// BAD: Variadic parameter added solely to avoid updating existing call sites.
func ProcessLogEntry(entry *log.Log, options ...ProcessOption) error {
 var opt ProcessOption
 if len(options) > 0 {
  opt = options[0]
 }
 // ...
}
```

**Good:**

```go
// GOOD: Required parameter is explicit. Update all callers across the repository.
func ProcessLogEntry(entry *log.Log, opt ProcessOption) error {
 // ...
}
```

---

### Anti-Pattern 2: Defensive Fallback / Default Injection for Non-Nil Fields

Do not insert nil checks and fallback default assignments for struct fields or parameters that are never nil in the application. Such checks obscure bugs, add dead execution paths, and mislead readers.

**Bad:**

```go
// BAD: Injects a fallback default even though WorkerPool is always provided by the caller.
func NewTaskRunner(cfg *TaskRunnerConfig) *TaskRunner {
 pool := cfg.WorkerPool
 if pool == nil {
  pool = workerpool.NewDefaultPool() // Unnecessary fallback for a non-nil internal contract
 }
 return &TaskRunner{pool: pool}
}
```

**Good:**

```go
// GOOD: Trust the application contract. If the field is required, expect it directly.
func NewTaskRunner(cfg *TaskRunnerConfig) *TaskRunner {
 if cfg.WorkerPool == nil {
  panic("worker pool must not be nil") // Or fail fast during construction
 }
 return &TaskRunner{pool: cfg.WorkerPool}
}
```

---

### Anti-Pattern 3: Retaining Deprecated Functions or Wrapper Shims

When renaming or refactoring a function, do not leave behind the old function with a `// Deprecated:` comment delegating to the new function. Delete the old function and update all call sites across the codebase.

**Bad:**

```go
// BAD: Keeps the old signature as a forwarding shim.
// Deprecated: Use ExtractResourceNamespace instead.
func GetResourceNamespace(res *Resource) string {
 return ExtractResourceNamespace(res)
}

func ExtractResourceNamespace(res *Resource) string {
 return res.Metadata.Namespace
}
```

**Good:**

```go
// GOOD: Rename the function cleanly and update all call sites across the monorepo.
func ExtractResourceNamespace(res *Resource) string {
 return res.Metadata.Namespace
}
```

---

### Anti-Pattern 4: Dual-Field Synchronization & Model Pollution

When replacing a field on a struct or model, do not retain both the old and new fields with runtime fallback or synchronization logic.

**Bad:**

```go
// BAD: Retains OldName alongside NewName and falls back at runtime.
type ClusterMetadata struct {
 ClusterName string // Deprecated
 Name        string
}

func (m *ClusterMetadata) GetName() string {
 if m.Name != "" {
  return m.Name
 }
 return m.ClusterName
}
```

**Good:**

```go
// GOOD: Rename the field directly and update all references across frontend and backend.
type ClusterMetadata struct {
 Name string
}

func (m *ClusterMetadata) GetName() string {
 return m.Name
}
```

---

### Anti-Pattern 5: Defensive Shims for Non-Existent Legacy Data

In internal pipelines, do not add `omitempty`, multi-format parsing, or nil fallbacks for imagined legacy data formats. All internal payloads are generated and consumed by the same version of KHI.

**Bad:**

```go
// BAD: Accepts both string and array formats for internal intermediate representation.
type InternalLogPayload struct {
 Tags any `json:"tags"` // Can be string or []string for "backward compatibility"
}
```

**Good:**

```go
// GOOD: Strictly define the single current format.
type InternalLogPayload struct {
 Tags []string `json:"tags"`
}
```

---

### Anti-Pattern 6: Task DAG Optional Dependencies & Feature Flags

When modifying the KHI Task DAG, do not use `coretask.GetTaskResultOptional` or add flags like `useLegacyMode` to support older, obsolete task graphs. All task definitions and registration points are maintained in the repository.

**Bad:**

```go
// BAD: Falls back to legacy logic if an upstream task is missing from the graph.
func MyTaskFunc(ctx context.Context, mode contract.InspectionTaskModeType) (*MyResult, error) {
 upstream := coretask.GetTaskResultOptional(ctx, UpstreamTaskID.Ref())
 if upstream == nil {
  return runLegacyFallbackLogic(ctx)
 }
 return runCurrentLogic(ctx, upstream)
}
```

**Good:**

```go
// GOOD: Declare the dependency as required and update pipeline definitions accordingly.
var MyTask = coretask.NewTask(
 MyTaskID,
 []taskid.UntypedTaskReference{
  UpstreamTaskID.Ref(),
 },
 func(ctx context.Context, mode contract.InspectionTaskModeType) (*MyResult, error) {
  upstream := coretask.GetTaskResult(ctx, UpstreamTaskID.Ref())
  return runCurrentLogic(ctx, upstream)
 },
)
```

---

### Anti-Pattern 7: Interface Pollution & Type Assertion Proliferation

When an interface needs a new method, do not create an `InterfaceV2` or use runtime type assertions to conditionally call the new method. Update the interface and update all its implementations in the monorepo.

**Bad:**

```go
// BAD: Uses runtime type assertion to avoid updating the existing interface.
type Formatter interface {
 Format(entry *log.Log) string
}

type FormatterV2 interface {
 FormatWithContext(ctx context.Context, entry *log.Log) string
}

func Render(ctx context.Context, f Formatter, entry *log.Log) string {
 if f2, ok := f.(FormatterV2); ok {
  return f2.FormatWithContext(ctx, entry)
 }
 return f.Format(entry)
}
```

**Good:**

```go
// GOOD: Update the interface directly and adjust all implementations in the codebase.
type Formatter interface {
 Format(ctx context.Context, entry *log.Log) string
}

func Render(ctx context.Context, f Formatter, entry *log.Log) string {
 return f.Format(ctx, entry)
}
```

---

### Anti-Pattern 8: Preserving Obsolete Behaviors in Test Suites

When refactoring a signature or behavior, update the test call sites to match the new signature. Do not create complex compatibility adapters or duplicate tests to verify obsolete behaviors.

> [!NOTE]
> Updating test call sites, constructor parameters, and assertions to adapt to refactored code is expected and required.
> The rule **"Do not remove test cases or weaken test assertions without asking the user"** forbids deleting test cases or weakening assertions without approval; it does not forbid updating test call sites to compile and verify refactored code.

**Bad:**

```go
// BAD: Retains tests for deprecated signatures alongside new ones.
func TestHelper_LegacySignature(t *testing.T) {
 // Preserving test for OldFunc which should have been deleted
 got := OldFunc("test")
 // ...
}
```

**Good:**

```go
// GOOD: Update test to invoke the new function directly.
func TestHelper(t *testing.T) {
 got := NewFunc("test", true)
 // ...
}
```

---

### Anti-Pattern 9: Dead Code / Commented-Out Implementations "Just in Case"

Never leave unused helper functions or commented-out code blocks in the codebase with notes like "Kept for backward compatibility" or "Old implementation". Source control history (Git / Jujutsu) preserves all historical implementations.

---

## 3. Workflow for `.khi` Format Changes

When you need to modify `.khi` file serialization, deserialization, or Protocol Buffer definitions:

1. **Identify the Scope**: Determine whether the change affects:
   - Exported file structure (Protocol Buffers under `pkg/generated/khifile/`).
   - Import/Export validators or serializers (`pkg/server/importinspection/`, `pkg/task/inspection/inspectioncore/impl/serializer.go`).
2. **Propose the Plan Explicitly**:
   - In your `/plan` implementation plan, specify what fields or structures are changing.
   - Explicitly raise an **Open Question** or **User Review Required** item asking whether previously exported `.khi` files must remain loadable.
3. **Wait for Approval**: Do not implement migration logic, version branches, or fallback decoders until the user confirms the compatibility requirement.
