package inspectiontaskbase

import (
	"context"
	"fmt"
	"testing"

	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func TestDefineInspectionTask(t *testing.T) {
	sourceRef := taskid.NewTaskReference[string]("define-inspection-test.source")
	task := DefineInspectionTask(taskid.NewDefaultImplementationID[string]("define-inspection-test.task"), func(b *coretask.Binder) InspectionTaskFunc[string] {
		source := coretask.Use(b, sourceRef)
		return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (string, error) {
			return fmt.Sprintf("%s:%d", source.Get(ctx), taskMode), nil
		}
	})

	testCases := []struct {
		name     string
		taskMode inspectioncore.InspectionTaskModeType
		want     string
	}{
		{
			name:     "passes the run mode and reads the declared input",
			taskMode: inspectioncore.TaskModeRun,
			want:     "source-value:2",
		},
		{
			name:     "passes the dry run mode and reads the declared input",
			taskMode: inspectioncore.TaskModeDryRun,
			want:     "source-value:1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			got, _, err := inspectiontest.RunInspectionTask(ctx, task, tc.taskMode, map[string]any{}, tasktest.NewTaskDependencyValuePair(sourceRef, "source-value"))
			if err != nil {
				t.Fatalf("RunInspectionTask() returned unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("RunInspectionTask() = %q, want %q", got, tc.want)
			}
		})
	}
}
