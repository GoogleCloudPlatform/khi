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

package task

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestProcessorTaskWithSuccessResult(t *testing.T) {
	processor := NewProcessorTask("foo-task", []string{"bar-output", "qux-output"}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		return "foo-value", nil
	})
	v := NewVariableSet(map[string]any{})
	err := processor.Runnable(0).Run(context.Background(), v)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	fooOutRaw, err := v.Get("foo-task")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if fooOutRaw.(string) != "foo-value" {
		t.Errorf("generated task didn't store returned value in variable. stored %s", fooOutRaw.(string))
	}
}

func TestProcessorTaskWithErroResult(t *testing.T) {
	errorFoo := fmt.Errorf("test error")
	processor := NewProcessorTask("foo-task", []string{"bar-output", "qux-output"}, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		return nil, errorFoo
	})
	v := NewVariableSet(map[string]any{})
	err := processor.Runnable(0).Run(context.Background(), v)

	if err != errorFoo {
		t.Errorf("expects an error. but no error returned")
	}

	resolved := v.IsResolved("foo-task")

	if resolved {
		t.Errorf("expects the output not to be resolved, but it has a value")
	}
}

func TestProcessorTaskToBeCreatedWithGivenProperties(t *testing.T) {
	taskId := "foo-task"
	sources := []string{"bar-output", "qux-output"}
	labels := []LabelOpt{WithLabel("label-foo", "foo-value"), WithLabel("label-bar", "bar-value")}
	processor := NewProcessorTask(taskId, sources, func(ctx context.Context, taskMode int, v *VariableSet) (any, error) {
		return "foo-value", nil
	}, labels...)

	v := NewVariableSet(map[string]any{})
	err := processor.Runnable(0).Run(context.Background(), v)

	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	processorDependenciesInStr := []string{}
	for _, dependency := range processor.Dependencies() {
		processorDependenciesInStr = append(processorDependenciesInStr, dependency.String())
	}

	cmpValues := []struct {
		Id           string
		Dependencies []string
		Label        *LabelSet
	}{
		{
			Id:           taskId,
			Dependencies: sources,
			Label:        NewLabelSet(labels...),
		}, {
			Id:           processor.ID().String(),
			Dependencies: processorDependenciesInStr,
			Label:        processor.Labels(),
		},
	}

	if diff := cmp.Diff(cmpValues[0], cmpValues[1], cmp.AllowUnexported(LabelSet{})); diff != "" {
		t.Errorf("processor was initialized with unexpected values\n%s", diff)
	}
}
