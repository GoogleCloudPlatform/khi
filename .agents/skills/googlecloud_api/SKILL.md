---
name: googlecloud-api
description: Guidelines and mandatory rules for calling Google Cloud APIs in KHI, including CallOptionInjector, ClientFactory, and context propagation.
---

# Calling Google Cloud APIs in KHI

This guide outlines the mandatory patterns and best practices for invoking Google Cloud APIs (such as Cloud Logging, Cloud Monitoring, GKE, and Compute Engine) across KHI tasks and SDK components.

---

## 1. Core Principles

All calls to Google Cloud APIs in KHI must carry proper call options (such as gRPC metadata, routing headers, quota parameters, and credentials) associated with the target Google Cloud resource container (`googlecloud.Project`, `googlecloud.Folder`, or `googlecloud.Organization`).

> [!IMPORTANT]
> **Mandatory Context Injection:**
> You **MUST** always apply `CallOptionInjector` to the call's `context.Context` (or HTTP headers) prior to invoking any Google Cloud API.
> Failing to call `InjectToCallContext` means required metadata and options will not propagate to the RPC, which can cause authentication, quota, or routing failures.

---

## 2. Using CallOptionInjector in Inspection Tasks

Inspection tasks under `pkg/task/` that interact with Google Cloud APIs must retrieve `CallOptionInjector` through the KHI task DAG dependency system.

### Step-by-Step Task Pattern

1. **Bind Task Inputs:**
   Declare `gcpcommon.APIClientCallOptionsInjectorTaskID.Ref()` on `*coretask.Binder` using `coretask.Use` (or `coretask.UseOptional` when the injector is optional).

2. **Retrieve Injector:**
   In the task execution function, read the injector from the bound input handle with `.Get(ctx)`.

3. **Inject Options Before Calling APIs:**
   Call `InjectToCallContext` on `ctx` for the target container, and pass the resulting context to client initialization and API calls.

### Code Sample: Task Implementation

```go
package mylog_impl

import (
 "context"
 "fmt"

 "cloud.google.com/go/logging/apiv2/loggingpb"
 "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
 inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
 coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
 "github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
 "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
 "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

var MyDiscoveryTask = inspectiontaskbase.DefineInspectionTask(
 taskid.NewDefaultImplementationID[[]string]("mylog.khi.google.com/discovery"),
 func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[[]string] {
  clientFactoryInput := coretask.Use(b, gcpcommon.APIClientFactoryTaskID.Ref())
  callOptionInjectorInput := coretask.Use(b, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

  return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]string, error) {
   if taskMode == inspectioncore.TaskModeDryRun {
    return []string{}, nil
   }

   clientFactory := clientFactoryInput.Get(ctx)
   callOptionInjector := callOptionInjectorInput.Get(ctx)

   container := googlecloud.Project("my-project-id")

   // 1. Inject call options into context for the container.
   callCtx := callOptionInjector.InjectToCallContext(ctx, container)

   // 2. Create client and make calls using callCtx.
   loggingClient, err := clientFactory.NewClient(callCtx, container)
   if err != nil {
    return nil, fmt.Errorf("failed to create logging client: %w", err)
   }
   defer loggingClient.Close()

   it := loggingClient.ListLogEntries(callCtx, &loggingpb.ListLogEntriesRequest{
    ResourceNames: []string{"projects/my-project-id"},
    PageSize:      10,
   })
   _ = it

   return []string{}, nil
  }
 },
)
```

---

## 3. Using CallOptionInjector in Reusable Components and Fetchers

When writing reusable helper structs that hold Google Cloud API clients (such as `LogFetcher`, `LocationFetcher`, or `StructuredLogEstimator`):

1. **Accept as Constructor Argument:** Pass `callOptionInjector *googlecloud.CallOptionInjector` into the constructor.
2. **Store as Struct Field:** Keep `CallOptionInjector *googlecloud.CallOptionInjector` on the struct.
3. **Inject in Methods:** Right before making the RPC calls (or before passing context to goroutines/`errgroup`), execute:

   ```go
   if e.CallOptionInjector != nil {
       ctx = e.CallOptionInjector.InjectToCallContext(ctx, container)
   }
   ```

### Code Sample: Component Implementation

```go
package myfetcher

import (
 "context"

 "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
 "golang.org/x/sync/errgroup"
)

type ResourceFetcher struct {
 CallOptionInjector *googlecloud.CallOptionInjector
 // Other client fields...
}

func NewResourceFetcher(injector *googlecloud.CallOptionInjector) *ResourceFetcher {
 return &ResourceFetcher{
  CallOptionInjector: injector,
 }
}

func (f *ResourceFetcher) FetchResources(ctx context.Context, container googlecloud.ResourceContainer) error {
 // Inject options before creating child context or errgroup.
 if f.CallOptionInjector != nil {
  ctx = f.CallOptionInjector.InjectToCallContext(ctx, container)
 }

 g, groupCtx := errgroup.WithContext(ctx)
 g.Go(func() error {
  // groupCtx now inherits all injected call options.
  return f.queryAPI(groupCtx, container)
 })

 return g.Wait()
}

func (f *ResourceFetcher) queryAPI(ctx context.Context, container googlecloud.ResourceContainer) error {
 // API call using ctx...
 return nil
}
```

---

## 4. Raw HTTP Requests

When sending REST/HTTP requests rather than using gRPC client libraries, gRPC context call options do not take effect. Instead, inject the options into the HTTP headers using `ApplyToRawHTTPHeader`:

```go
req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
if err != nil {
 return err
}

if callOptionInjector != nil {
 callOptionInjector.ApplyToRawHTTPHeader(req.Header, container)
}

resp, err := httpClient.Do(req)
// ...
```

---

## 5. Testing Components with CallOptionInjector

When writing unit tests for tasks or components that use `CallOptionInjector`:

- Verify that `CallOptionInjector` executes and modifies the context using a mock `CallOptionInjectorOption`.
- Ensure the component functions correctly when `CallOptionInjector` is `nil` (defensive programming).

### Test Example

```go
type mockCallOptionInjectorOption struct {
 key   string
 value string
}

func (m *mockCallOptionInjectorOption) ApplyToCallContext(ctx context.Context, container googlecloud.ResourceContainer) context.Context {
 return context.WithValue(ctx, m.key, m.value)
}

func (m *mockCallOptionInjectorOption) ApplyToRawHTTPHeader(header http.Header, container googlecloud.ResourceContainer) {}

var _ googlecloud.CallOptionInjectorOption = (*mockCallOptionInjectorOption)(nil)

func TestComponent_CallOptionInjector(t *testing.T) {
 testCases := []struct {
  name        string
  injector    *googlecloud.CallOptionInjector
  wantValue   any
 }{
  {
   name: "injector modifies context",
   injector: googlecloud.NewCallOptionInjector(&mockCallOptionInjectorOption{
    key:   "test-key",
    value: "test-value",
   }),
   wantValue: "test-value",
  },
  {
   name:      "nil injector succeeds without error",
   injector:  nil,
   wantValue: nil,
  },
 }

 for _, tc := range testCases {
  t.Run(tc.name, func(t *testing.T) {
   component := NewResourceFetcher(tc.injector)
   err := component.FetchResources(context.Background(), googlecloud.Project("test-project"))
   if err != nil {
    t.Fatalf("unexpected error: %v", err)
   }
  })
 }
}
```
