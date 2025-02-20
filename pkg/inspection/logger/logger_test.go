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
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestGlobalLoggerHandlerWithChildLogger(t *testing.T) {
	bufDefault := new(bytes.Buffer)
	buf1 := new(bytes.Buffer)
	buf1Handler := slog.NewTextHandler(buf1, nil)
	buf2 := new(bytes.Buffer)
	buf2Handler := slog.NewTextHandler(buf2, nil)
	lh := localInitInspectionLogger(slog.NewTextHandler(bufDefault, nil))
	ctx := context.Background()
	t1Ctx := context.WithValue(context.WithValue(context.WithValue(ctx, "tid", taskid.NewTaskImplementationId("task1")), "iid", "inspection1"), "rid", "r1")
	t2Ctx := context.WithValue(context.WithValue(context.WithValue(ctx, "tid", taskid.NewTaskImplementationId("task2")), "iid", "inspection2"), "rid", "r2")
	logger := slog.New(lh)

	logger.Info("default info")
	logger.InfoContext(ctx, "default info")
	logger.InfoContext(t1Ctx, "unknown task")
	lh.RegisterTaskLogger("inspection1", "task1", "r1", buf1Handler)
	lh.RegisterTaskLogger("inspection2", "task2", "r2", buf2Handler)
	logger.InfoContext(t1Ctx, "inspection1 task1 info")
	logger.InfoContext(t2Ctx, "inspection2 task2 info")

	expectedDefaultBuf := `level=INFO msg="default info"
level=INFO msg="default info"
level=INFO msg="unknown task"
`
	expectedBuf1 := `level=INFO msg="inspection1 task1 info"
`
	expectedBuf2 := `level=INFO msg="inspection2 task2 info"
`
	actualDefaultBuf := testutil.RemoveSlogTimestampFromLine(bufDefault.String())
	actualBuf1 := testutil.RemoveSlogTimestampFromLine(buf1.String())
	actualBuf2 := testutil.RemoveSlogTimestampFromLine(buf2.String())

	if actualDefaultBuf != expectedDefaultBuf {
		t.Errorf("the logs contained in the default logger is mismatched\nexpected:\n%s\nactual:\n%s", expectedDefaultBuf, actualDefaultBuf)
	}
	if actualBuf1 != expectedBuf1 {
		t.Errorf("the logs contained in the i1t1 logger is mismatched\nexpected:\n%s\nactual:\n%s", expectedBuf1, actualBuf1)
	}
	if actualBuf2 != expectedBuf2 {
		t.Errorf("the logs contained in the i2t2 logger is mismatched\nexpected:\n%s\nactual:\n%s", expectedBuf2, actualBuf2)
	}
}
