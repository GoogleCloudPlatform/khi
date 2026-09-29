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

package progress

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

func TestResolveTitle(t *testing.T) {
	testCases := []struct {
		name      string
		baseID    string
		hash      string
		opts      []coretask.LabelOpt
		wantTitle string
	}{
		{
			name:      "explicit progress title label",
			baseID:    "khi.google.com/inspection/googlecloudcommon/list-log-entries",
			hash:      "k8s_audit",
			opts:      []coretask.LabelOpt{coretask.WithTitle("Fetch k8s_audit logs")},
			wantTitle: "Fetch k8s_audit logs",
		},
		{
			name:      "strip khi.google.com/inspection/ prefix with hash",
			baseID:    "khi.google.com/inspection/googlecloudcommon/list-log-entries",
			hash:      "k8s_audit",
			opts:      nil,
			wantTitle: "googlecloudcommon/list-log-entries#k8s_audit",
		},
		{
			name:      "strip khi.google.com/inspection/ prefix",
			baseID:    "khi.google.com/inspection/commonlogk8saudit/manifest-generator",
			opts:      nil,
			wantTitle: "commonlogk8saudit/manifest-generator",
		},
		{
			name:      "strip khi.google.com/ prefix",
			baseID:    "khi.google.com/core/config-loader",
			opts:      nil,
			wantTitle: "core/config-loader",
		},
		{
			name:      "custom task ID without prefix",
			baseID:    "custom-task-id",
			opts:      nil,
			wantTitle: "custom-task-id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var id taskid.TaskImplementationID[any]
			if tc.hash != "" {
				id = taskid.NewImplementationID(taskid.NewTaskReference[any](tc.baseID), tc.hash)
			} else {
				id = taskid.NewDefaultImplementationID[any](tc.baseID)
			}
			task := coretask.NewTask(id, nil, func(ctx context.Context) (any, error) {
				return nil, nil
			}, tc.opts...)

			got := ResolveTitle(task)
			if got != tc.wantTitle {
				t.Errorf("ResolveTitle() = %q, want %q", got, tc.wantTitle)
			}
		})
	}
}

func TestContextPropagationAndReport(t *testing.T) {
	testCases := []struct {
		name              string
		setupCtx          func(tp *inspectionmetadata.TaskProgressMetadata) context.Context
		action            func(ctx context.Context)
		wantRatio         float32
		wantIndeterminate bool
		wantMsg           string
	}{
		{
			name: "Report updates ratio and message on populated context",
			setupCtx: func(tp *inspectionmetadata.TaskProgressMetadata) context.Context {
				return WithContext(context.Background(), tp)
			},
			action: func(ctx context.Context) {
				Report(ctx, 0.5, "Halfway done")
			},
			wantRatio:         0.5,
			wantIndeterminate: false,
			wantMsg:           "Halfway done",
		},
		{
			name: "ReportIndeterminate sets indeterminate flag and message",
			setupCtx: func(tp *inspectionmetadata.TaskProgressMetadata) context.Context {
				return WithContext(context.Background(), tp)
			},
			action: func(ctx context.Context) {
				Report(ctx, 0.5, "Halfway done")
				ReportIndeterminate(ctx, "Searching resources...")
			},
			wantRatio:         0,
			wantIndeterminate: true,
			wantMsg:           "Searching resources...",
		},
		{
			name: "FromContext and Report on empty context safely no-op without panic",
			setupCtx: func(_ *inspectionmetadata.TaskProgressMetadata) context.Context {
				return context.Background()
			},
			action: func(ctx context.Context) {
				noop := FromContext(ctx)
				if noop == nil {
					t.Fatal("FromContext(empty) returned nil, want non-nil no-op instance")
				}
				Report(ctx, 0.8, "No panic on empty context")
				ReportIndeterminate(ctx, "No indeterminate mutation")
				noop.SetLabel("mutated")
				snap := noop.Snapshot()
				if snap.Ratio != 0 || snap.Message != "" || snap.Indeterminate || snap.Label != "noop" {
					t.Errorf("noop.Snapshot() = %+v, want unmodified noop snapshot", snap)
				}
			},
			wantRatio:         0,
			wantIndeterminate: false,
			wantMsg:           "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tp := inspectionmetadata.NewTaskProgressMetadata("test-task")
			ctx := tc.setupCtx(tp)
			tc.action(ctx)

			snap := tp.Snapshot()
			if snap.Ratio != tc.wantRatio {
				t.Errorf("snap.Ratio = %v, want %v", snap.Ratio, tc.wantRatio)
			}
			if snap.Indeterminate != tc.wantIndeterminate {
				t.Errorf("snap.Indeterminate = %v, want %v", snap.Indeterminate, tc.wantIndeterminate)
			}
			if snap.Message != tc.wantMsg {
				t.Errorf("snap.Message = %q, want %q", snap.Message, tc.wantMsg)
			}
		})
	}
}

func TestTaskInterceptor(t *testing.T) {
	testCases := []struct {
		name        string
		setupCtx    func() (context.Context, *inspectionmetadata.Progress)
		taskErr     error
		wantTaskErr bool
	}{
		{
			name: "attaches progress, cancels task context on exit, and resolves task",
			setupCtx: func() (context.Context, *inspectionmetadata.Progress) {
				prog := inspectionmetadata.NewProgress()
				prog.SetTotalTaskCount(1)
				md := typedmap.NewTypedMap()
				typedmap.Set(md, inspectionmetadata.ProgressMetadataKey, prog)
				ctx := khictx.WithValue(context.Background(), inspectionmetadata.MapContextKey, md.AsReadonly())
				return ctx, prog
			},
		},
		{
			name: "executes safely when metadata map is absent from context",
			setupCtx: func() (context.Context, *inspectionmetadata.Progress) {
				return context.Background(), nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, prog := tc.setupCtx()
			taskID := taskid.NewDefaultImplementationID[any]("test-task")
			task := coretask.NewTask(taskID, nil, func(ctx context.Context) (any, error) { return nil, nil }, coretask.WithTitle("Test Task"))

			var capturedCtx context.Context
			_, err := TaskInterceptor(ctx, task, func(tCtx context.Context) (any, error) {
				capturedCtx = tCtx
				_ = NewTracker(tCtx, 100)
				Report(tCtx, 0.5, "in progress")
				return "ok", tc.taskErr
			})
			if (err != nil) != tc.wantTaskErr {
				t.Errorf("TaskInterceptor() error = %v, wantErr %v", err, tc.wantTaskErr)
			}
			if capturedCtx.Err() == nil {
				t.Errorf("capturedCtx.Err() = nil, want context cancelled upon task exit")
			}
			if prog != nil {
				snap := prog.Snapshot()
				if len(snap.TaskProgresses) != 0 {
					t.Errorf("len(snap.TaskProgresses) = %d, want 0 after ResolveTask", len(snap.TaskProgresses))
				}
			}
		})
	}
}
