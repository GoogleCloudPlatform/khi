// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMain(m *testing.M) {
	// The inspection runner registers per-task loggers on the global KHI logger.
	logger.InitGlobalKHILogger()
	os.Exit(m.Run())
}

func TestInspectionCreate_Golden(t *testing.T) {
	server, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	handler := NewInspectionHandler(server)

	testCases := []struct {
		name       string
		template   string
		goldenFile string
		data       any
	}{
		{
			name:       "InspectionTypes",
			template:   "inspection_types.md.tmpl",
			goldenFile: "testdata/inspection_types.golden.md",
			data: inspectionTypesData{
				Types: []inspectionTypeRow{
					{
						ID:          "gcp-gke",
						Name:        "Google Kubernetes Engine",
						Description: "Gather and parse Google Kubernetes Engine (GKE) cluster logs ...",
						Available:   "yes",
					},
					{
						ID:          "oss-kubernetes-from-files",
						Name:        "OSS Kubernetes Log Files",
						Description: "Parse uploaded OSS Kubernetes log files to visualize cluster operations on timelines.",
						Available:   "no: requires file uploads, which are not supported via MCP yet. Use the KHI Web UI.",
					},
				},
			},
		},
		{
			name:       "InspectionFeatures",
			template:   "inspection_features.md.tmpl",
			goldenFile: "testdata/inspection_features.golden.md",
			data: inspectionFeaturesData{
				ID:       "2026-09-24-0130-a1b2",
				Name:     "prod-cluster-1 restart investigation",
				TypeName: "Google Kubernetes Engine",
				TypeID:   "gcp-gke",
				Features: []featureRow{
					{
						ID:          "k8s-audit-log",
						Name:        "Kubernetes Audit Logs",
						Description: "Kubernetes audit logs from Cloud Logging.",
						Enabled:     "yes",
					},
					{
						ID:          "k8s-event-log",
						Name:        "Kubernetes Event Logs",
						Description: "Kubernetes events such as scheduling failures and OOM kills.",
						Enabled:     "yes",
					},
					{
						ID:          "k8s-node-log",
						Name:        "Kubernetes Node Logs",
						Description: "kubelet and container runtime logs from Cloud Logging.",
						Enabled:     "no",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := handler.templates.Render(tc.template, tc.data)
			if err != nil {
				t.Fatalf("failed to render %s: %v", tc.template, err)
			}

			want := readGolden(t, tc.goldenFile)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tc.template, diff)
			}
		})
	}
}

// readGolden reads a golden file and trims trailing newlines to match rendered template output.
func readGolden(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}
	return strings.TrimRight(string(b), "\r\n")
}

// extractInspectionID returns the inspection ID from the heading of a create_inspection result.
func extractInspectionID(t *testing.T, text string) string {
	t.Helper()
	const prefix = "# Inspection `"
	start := strings.Index(text, prefix)
	if start == -1 {
		t.Fatalf("could not find inspection ID start in %q", text)
	}
	rem := text[start+len(prefix):]
	end := strings.Index(rem, "`")
	if end == -1 {
		t.Fatalf("could not find inspection ID end in %q", text)
	}
	return rem[:end]
}

// toolResultText returns the single text content and the error flag of a tool result.
func toolResultText(t *testing.T, name string, res *mcpsdk.CallToolResult) (string, bool) {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("%s returned %d contents, want 1", name, len(res.Content))
	}
	content, ok := res.Content[0].(*mcpsdk.TextContent)
	if !ok {
		t.Fatalf("%s content type = %T, want *mcpsdk.TextContent", name, res.Content[0])
	}
	return content.Text, res.IsError
}

// callTool calls the named tool and returns its text content and error flag.
func callTool(t *testing.T, ctx context.Context, session *mcpsdk.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("session.CallTool(%s) failed: %v", name, err)
	}
	return toolResultText(t, name, res)
}

// checkToolResult compares the full text and error flag of a tool result.
func checkToolResult(t *testing.T, name string, gotText string, gotIsError bool, wantText string, wantIsError bool) {
	t.Helper()
	if gotIsError != wantIsError {
		t.Errorf("%s isError = %v, want %v", name, gotIsError, wantIsError)
	}
	if diff := cmp.Diff(wantText, gotText); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", name, diff)
	}
}

// requireInspectionID stops a dependent subtest when an earlier subtest failed to create the inspection.
func requireInspectionID(t *testing.T, id string) {
	t.Helper()
	if id == "" {
		t.Fatal("inspection ID is empty because an earlier subtest failed")
	}
}

// featuresText builds the expected create_inspection and update_inspection_features output.
func featuresText(id, name string, rows ...string) string {
	return fmt.Sprintf("# Inspection `%s`\n\n- Name: %s\n- Type: Google Kubernetes Engine (`gcp-gke`)\n\n"+
		"To change the enabled features, call `update_inspection_features` with the IDs of all features to enable. Then call `dry_run_inspection`.\n\n"+
		"## Features\n\n| ID | Name | Description | Enabled |\n| --- | --- | --- | --- |\n%s", id, name, strings.Join(rows, "\n"))
}

// toolsTestEnv is an MCP client session connected to an inspection server with test feature tasks.
type toolsTestEnv struct {
	server  *coreinspection.InspectionTaskServer
	session *mcpsdk.ClientSession
	ctx     context.Context
	// blockingTaskStarted receives a value each time the blocking feature task starts in run mode.
	blockingTaskStarted chan struct{}
	// releaseBlockingTask lets every running and future blocking feature task finish successfully.
	releaseBlockingTask func()
}

const (
	blockingFeatureID = "k8s-audit-log#default"
	authFailFeatureID = "auth-fail-feature#default"
	doneFeatureID     = "done-feature#default"
	failFeatureID     = "fail-feature#default"
	// untitledFailFeatureID fails in run mode and has no progress title.
	untitledFailFeatureID = "untitled-fail-feature#default"
)

// newToolsTestEnv starts an MCP server backed by an inspection server with these feature tasks:
// a default-enabled task that blocks until released and depends on a required cluster-name form,
// a task failing with a Google auth error, a task succeeding immediately, a task failing in run mode,
// and a task without a title failing in run mode.
func newToolsTestEnv(t *testing.T) *toolsTestEnv {
	t.Helper()
	server, err := coreinspection.NewServer(&inspectioncore.IOConfig{
		TemporaryFolder: t.TempDir(),
		DataDestination: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	inspectionTypes := []coreinspection.InspectionType{
		{
			Id:          "gcp-gke",
			Name:        "Google Kubernetes Engine",
			Description: "Gather and parse Google Kubernetes Engine (GKE) cluster logs",
			Priority:    100,
		},
		{
			Id:          "oss-kubernetes-from-files",
			Name:        "OSS Kubernetes Log Files",
			Description: "Parse uploaded OSS Kubernetes log files to visualize cluster operations on timelines.",
			Priority:    50,
			Labels: map[string]string{
				inspectioncore.InspectionTypeLabelKeyLogSource: "file",
			},
		},
	}
	for _, inspType := range inspectionTypes {
		if err := server.AddInspectionType(inspType); err != nil {
			t.Fatalf("failed to add inspection type %s: %v", inspType.Id, err)
		}
	}

	clusterNameTask := formtask.NewTextFormTaskBuilder(
		taskid.NewDefaultImplementationID[string]("cluster-name"),
		1,
		"Cluster Name",
	).WithValidator(func(ctx context.Context, value string) (string, error) {
		if value == "" {
			return "cluster name is required", nil
		}
		return "", nil
	}).Build()

	blockingTaskStarted := make(chan struct{}, 1)
	releaseCh := make(chan struct{})
	var releaseOnce sync.Once
	releaseBlockingTask := func() {
		releaseOnce.Do(func() { close(releaseCh) })
	}

	blockingFeatureTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("k8s-audit-log"),
		[]coretask.Dependency{clusterNameTask.UntypedID().GetUntypedReference()},
		func(ctx context.Context) (any, error) {
			mode, _ := khictx.GetValue(ctx, inspectioncore.InspectionTaskMode)
			if mode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}
			progress.Report(ctx, 0.6, "91200 logs fetched")
			select {
			case blockingTaskStarted <- struct{}{}:
			default:
			}
			select {
			case <-releaseCh:
				return nil, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
		inspectioncore.FeatureTaskLabel("Kubernetes Audit Logs", "Audit logs", 1, true),
		coretask.WithTitle("Query K8s audit logs"),
	)

	authFailTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("auth-fail-feature"),
		nil,
		func(ctx context.Context) (any, error) {
			// gRPC clients report a token fetch failure of expired credentials in this form.
			return nil, status.Error(codes.Unauthenticated, "transport: per-RPC creds failed due to error: auth: cannot fetch token: 400\nResponse: {\"error\":\"invalid_grant\"}")
		},
		inspectioncore.FeatureTaskLabel("Auth Fail Feature", "Fails with auth error", 3, false),
	)

	doneFeatureTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("done-feature"),
		nil,
		func(ctx context.Context) (any, error) {
			return "done", nil
		},
		inspectioncore.FeatureTaskLabel("Done Feature", "Succeeds immediately", 4, false),
	)

	failFeatureTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("fail-feature"),
		nil,
		func(ctx context.Context) (any, error) {
			mode, _ := khictx.GetValue(ctx, inspectioncore.InspectionTaskMode)
			if mode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}
			return nil, errors.New("network connection lost")
		},
		inspectioncore.FeatureTaskLabel("Fail Feature", "Fails in run mode", 5, false),
		coretask.WithTitle("Fail task"),
	)

	untitledFailFeatureTask := coretask.NewTask(
		taskid.NewDefaultImplementationID[any]("untitled-fail-feature"),
		nil,
		func(ctx context.Context) (any, error) {
			mode, _ := khictx.GetValue(ctx, inspectioncore.InspectionTaskMode)
			if mode == inspectioncore.TaskModeDryRun {
				return nil, nil
			}
			return nil, errors.New("disk quota exceeded")
		},
		inspectioncore.FeatureTaskLabel("Untitled Fail Feature", "Fails in run mode without a title", 6, false),
		coretask.WithTitle(""),
	)

	for _, task := range []coretask.UntypedTask{clusterNameTask, blockingFeatureTask, authFailTask, doneFeatureTask, failFeatureTask, untitledFailFeatureTask} {
		if err := server.AddTask(task); err != nil {
			t.Fatalf("failed to add task %s: %v", task.UntypedID(), err)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", NewServer(NewInspectionHandler(server)).HTTPHandler())
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)
	session, err := client.Connect(ctx, &mcpsdk.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	// Registered last so that it runs first and no task stays blocked after the test.
	t.Cleanup(releaseBlockingTask)

	return &toolsTestEnv{
		server:              server,
		session:             session,
		ctx:                 ctx,
		blockingTaskStarted: blockingTaskStarted,
		releaseBlockingTask: releaseBlockingTask,
	}
}

func TestInspectionTools_InspectionNotFound(t *testing.T) {
	env := newToolsTestEnv(t)
	want := "Error: INSPECTION_NOT_FOUND\n\n- Inspection `nonexistent-id` not found.\n- Read `khi://inspections` to see existing inspections."

	testCases := []struct {
		tool string
		args map[string]any
	}{
		{tool: "update_inspection_features", args: map[string]any{"inspectionId": "nonexistent-id", "enabledFeatureIds": []string{doneFeatureID}}},
		{tool: "dry_run_inspection", args: map[string]any{"inspectionId": "nonexistent-id"}},
		{tool: "run_inspection", args: map[string]any{"inspectionId": "nonexistent-id"}},
	}

	for _, tc := range testCases {
		t.Run(tc.tool, func(t *testing.T) {
			text, isError := callTool(t, env.ctx, env.session, tc.tool, tc.args)
			checkToolResult(t, tc.tool, text, isError, want, true)
		})
	}
}

func TestInspectionTools_CreateInspectionErrors(t *testing.T) {
	env := newToolsTestEnv(t)

	testCases := []struct {
		name   string
		typeID string
		want   string
	}{
		{
			name:   "type requiring file uploads",
			typeID: "oss-kubernetes-from-files",
			want:   "Error: INSPECTION_TYPE_NOT_AVAILABLE\n\n- Inspection type `oss-kubernetes-from-files` requires file uploads, which are not supported via MCP yet.\n- Ask the user to create this inspection in the KHI Web UI.",
		},
		{
			name:   "unknown type",
			typeID: "unknown-type",
			want:   "Error: UNKNOWN_INSPECTION_TYPE\n\n- Unknown inspection type ID: `unknown-type`\n- Read `khi://inspection-types` to see available types.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runnersBefore := len(env.server.GetAllRunners())
			text, isError := callTool(t, env.ctx, env.session, "create_inspection", map[string]any{
				"inspectionTypeId": tc.typeID,
			})
			checkToolResult(t, "create_inspection", text, isError, tc.want, true)
			if got := len(env.server.GetAllRunners()); got != runnersBefore {
				t.Errorf("runner count = %d, want %d", got, runnersBefore)
			}
		})
	}

	t.Run("duplicate name", func(t *testing.T) {
		text, isError := callTool(t, env.ctx, env.session, "create_inspection", map[string]any{
			"inspectionTypeId": "gcp-gke",
			"name":             "custom-name-test",
		})
		if isError {
			t.Fatalf("first create_inspection returned error: %s", text)
		}

		runnersBefore := len(env.server.GetAllRunners())
		text, isError = callTool(t, env.ctx, env.session, "create_inspection", map[string]any{
			"inspectionTypeId": "gcp-gke",
			"name":             "custom-name-test",
		})
		want := "Error: INVALID_INSPECTION_NAME\n\n- Inspection name \"custom-name-test\" is already in use. Choose another name."
		checkToolResult(t, "create_inspection", text, isError, want, true)
		if got := len(env.server.GetAllRunners()); got != runnersBefore {
			t.Errorf("runner count = %d, want %d", got, runnersBefore)
		}
	})
}

func TestInspectionTools_NamedInspectionLifecycle(t *testing.T) {
	env := newToolsTestEnv(t)
	const inspectionName = "prod-cluster-1 restart investigation"
	var id string

	t.Run("create returns default features", func(t *testing.T) {
		text, isError := callTool(t, env.ctx, env.session, "create_inspection", map[string]any{
			"inspectionTypeId": "gcp-gke",
			"name":             inspectionName,
		})
		if isError {
			t.Fatalf("create_inspection returned error: %s", text)
		}
		id = extractInspectionID(t, text)
		want := featuresText(id, inspectionName,
			fmt.Sprintf("| `%s` | Kubernetes Audit Logs | Audit logs | yes |", blockingFeatureID),
			fmt.Sprintf("| `%s` | Auth Fail Feature | Fails with auth error | no |", authFailFeatureID),
			fmt.Sprintf("| `%s` | Done Feature | Succeeds immediately | no |", doneFeatureID),
			fmt.Sprintf("| `%s` | Fail Feature | Fails in run mode | no |", failFeatureID),
			fmt.Sprintf("| `%s` | Untitled Fail Feature | Fails in run mode without a title | no |", untitledFailFeatureID),
		)
		checkToolResult(t, "create_inspection", text, isError, want, false)
	})

	t.Run("update rejects invalid feature lists", func(t *testing.T) {
		requireInspectionID(t, id)
		testCases := []struct {
			name       string
			featureIDs []string
			want       string
		}{
			{
				name:       "unknown feature",
				featureIDs: []string{"k8s-audit"},
				want:       "Error: UNKNOWN_FEATURE\n\n- Unknown feature IDs: `k8s-audit`\n- Use the IDs returned by `create_inspection`.",
			},
			{
				name:       "no features",
				featureIDs: []string{},
				want:       "Error: NO_FEATURES\n\n- Specify at least one feature ID to enable.\n- Use the IDs returned by `create_inspection`.",
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				text, isError := callTool(t, env.ctx, env.session, "update_inspection_features", map[string]any{
					"inspectionId":      id,
					"enabledFeatureIds": tc.featureIDs,
				})
				checkToolResult(t, "update_inspection_features", text, isError, tc.want, true)
			})
		}
	})

	t.Run("update accepts duplicate IDs", func(t *testing.T) {
		requireInspectionID(t, id)
		text, isError := callTool(t, env.ctx, env.session, "update_inspection_features", map[string]any{
			"inspectionId":      id,
			"enabledFeatureIds": []string{doneFeatureID, doneFeatureID},
		})
		want := featuresText(id, inspectionName,
			"| `k8s-audit-log#default` | Kubernetes Audit Logs | Audit logs | no |",
			"| `auth-fail-feature#default` | Auth Fail Feature | Fails with auth error | no |",
			"| `done-feature#default` | Done Feature | Succeeds immediately | yes |",
			"| `fail-feature#default` | Fail Feature | Fails in run mode | no |",
			"| `untitled-fail-feature#default` | Untitled Fail Feature | Fails in run mode without a title | no |",
		)
		checkToolResult(t, "update_inspection_features", text, isError, want, false)
	})

	t.Run("run and wait until done", func(t *testing.T) {
		requireInspectionID(t, id)
		env.runInspection(t, id, map[string]any{})
		<-env.server.GetInspection(id).Wait()
	})

	t.Run("list shows the name", func(t *testing.T) {
		requireInspectionID(t, id)
		res, err := env.session.ReadResource(env.ctx, &mcpsdk.ReadResourceParams{URI: "khi://inspections"})
		if err != nil {
			t.Fatalf("ReadResource(khi://inspections) failed: %v", err)
		}
		if len(res.Contents) != 1 {
			t.Fatalf("len(res.Contents) = %d, want 1", len(res.Contents))
		}
		rowPrefix := fmt.Sprintf("| `%s` |", id)
		var row string
		for line := range strings.SplitSeq(res.Contents[0].Text, "\n") {
			if strings.HasPrefix(line, rowPrefix) {
				row = line
				break
			}
		}
		// Only the prefix is checked because the time range and label columns depend on the run time and
		// on metadata written by framework tasks, which this test does not control.
		wantPrefix := fmt.Sprintf("| `%s` | %s | Google Kubernetes Engine | DONE |", id, inspectionName)
		if !strings.HasPrefix(row, wantPrefix) {
			t.Errorf("inspection row = %q, want prefix %q\nfull list:\n%s", row, wantPrefix, res.Contents[0].Text)
		}
	})

	t.Run("already started", func(t *testing.T) {
		requireInspectionID(t, id)
		want := fmt.Sprintf("Error: INSPECTION_ALREADY_STARTED\n\n- Inspection `%s` has already been started.\n- Create a new inspection with `create_inspection`.", id)
		testCases := []struct {
			tool string
			args map[string]any
		}{
			{tool: "update_inspection_features", args: map[string]any{"inspectionId": id, "enabledFeatureIds": []string{doneFeatureID}}},
			{tool: "dry_run_inspection", args: map[string]any{"inspectionId": id}},
			{tool: "run_inspection", args: map[string]any{"inspectionId": id}},
		}
		for _, tc := range testCases {
			t.Run(tc.tool, func(t *testing.T) {
				text, isError := callTool(t, env.ctx, env.session, tc.tool, tc.args)
				checkToolResult(t, tc.tool, text, isError, want, true)
			})
		}
	})
}
