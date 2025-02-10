package label

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

func TestQueryTaskLabelOpt(t *testing.T) {
	labelOpt := NewQueryTaskLabelOpt(enum.LogTypeComputeApi, "sample query")
	label := task.NewLabelSet(labelOpt)

	anyQueryTask, exists := label.Get(TaskLabelKeyIsQueryTask)
	if !exists {
		t.Errorf("TaskLabel %s is expected to be set, but it is not", TaskLabelKeyIsQueryTask)
	}
	if anyQueryTask.(bool) != true {
		t.Errorf("TaskLabel %s is expected to be true, but it is %v", TaskLabelKeyIsQueryTask, anyQueryTask)
	}

	targetLogType, exists := label.Get(TaskLabelKeyQueryTaskTargetLogType)
	if !exists {
		t.Errorf("TaskLabel %s is expected to be set, but it is not", TaskLabelKeyQueryTaskTargetLogType)
	}
	if targetLogType.(enum.LogType) != enum.LogTypeComputeApi {
		t.Errorf("TaskLabel %s is expected to be %v, but it is %v", TaskLabelKeyQueryTaskTargetLogType, enum.LogTypeComputeApi, targetLogType)
	}

	sampleQuery, exists := label.Get(TaskLabelKeyQueryTaskSampleQuery)
	if !exists {
		t.Errorf("TaskLabel %s is expected to be set, but it is not", TaskLabelKeyQueryTaskSampleQuery)
	}
	if sampleQuery.(string) != "sample query" {
		t.Errorf("TaskLabel %s is expected to be sample query, but it is %v", TaskLabelKeyQueryTaskSampleQuery, sampleQuery)
	}
}
