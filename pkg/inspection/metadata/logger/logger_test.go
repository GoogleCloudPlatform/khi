// Copyright 2024 Google LLC
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

package logger

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/logger"
	inspection_task_test "github.com/GoogleCloudPlatform/khi/pkg/inspection/test"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	metadata_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/metadata"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestConformance(t *testing.T) {
	logger := NewLogger()
	metadata_test.ConformanceMetadataTypeTest(t, logger)
}

func testRecord(attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(time.Time{}, slog.LevelDebug, "", 0)
	for _, attr := range attrs {
		r.AddAttrs(attr)
	}
	return r
}

func TestChildLoggers(t *testing.T) {
	logger.InitGlobalKHILogger()
	logger := NewLogger()

	ctx := inspection_task_test.WithDefaultTestInspectionTaskContext(context.Background())
	tid1 := taskid.NewDefaultImplementationID[any]("task1").(taskid.UntypedTaskImplementationID)
	tid2 := taskid.NewDefaultImplementationID[any]("task2").(taskid.UntypedTaskImplementationID)
	log1Ctx := khictx.WithValue(ctx, task_contextkey.TaskImplementationIDContextKey, tid1)
	log2Ctx := khictx.WithValue(ctx, task_contextkey.TaskImplementationIDContextKey, tid2)
	log1 := logger.MakeTaskLogger(log1Ctx, slog.LevelDebug)
	if log1 == nil {
		t.Errorf("failed to generate task logger for ctx1")
	}
	log2 := logger.MakeTaskLogger(log2Ctx, slog.LevelDebug)
	if log2 == nil {
		t.Errorf("failed to generate task logger for ctx2")
	}

	slog.InfoContext(log1Ctx, "task1 message")
	slog.InfoContext(log2Ctx, "task2 message")

	actual1 := log1.Read()
	actual2 := log2.Read()
	expect1 := `task1#default > INFO task1 message
`
	expect2 := `task2#default > INFO task2 message
`

	if expect1 != actual1 {
		t.Errorf("actual1 != expect1\nexpect:%s\nactual:%s", expect1, actual1)
	}

	if expect2 != actual2 {
		t.Errorf("actual2 != expect2\nexpect:%s\nactual:%s", expect2, actual2)
	}
}

func TestTaskSlogHandler_getLogKind(t *testing.T) {
	testCases := []struct {
		Name     string
		Record   slog.Record
		Expected string
	}{
		{
			Name:     "simple example",
			Record:   testRecord(slog.String("foo", "bar"), slog.String(logger.LogKindAttrKey, "kind-foo"), slog.String("bar", "baz")),
			Expected: "kind-foo",
		},
		{
			Name:     "no matching attr",
			Record:   testRecord(slog.String("foo", "bar"), slog.String("bar", "baz")),
			Expected: "",
		},
		{
			Name:     "no attr",
			Record:   testRecord(),
			Expected: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			handler := TaskSlogHandler{}
			actual := handler.getLogKind(tc.Record)
			if tc.Expected != actual {
				t.Errorf(
					"actual:%s\nexpect:%s",
					actual,
					tc.Expected,
				)
			}
		})
	}
}
