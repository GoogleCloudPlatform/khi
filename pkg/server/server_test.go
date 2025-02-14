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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/popup"
	"github.com/GoogleCloudPlatform/khi/pkg/server/config"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	task_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/task"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/GoogleCloudPlatform/khi/pkg/task"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testPopupForm struct{}

// GetMetadata implements popup.PopupForm.
func (t testPopupForm) GetMetadata() popup.PopupFormMetadata {
	return popup.PopupFormMetadata{
		Title:       "foo",
		Type:        "bar",
		Description: "baz",
	}
}

// Validate implements popup.PopupForm.
func (t testPopupForm) Validate(req *popup.PopupAnswerResponse) string {
	if strings.Contains(req.Value, "ok") {
		return ""
	} else {
		return "answer for test popup must contain ok"
	}
}

var _ popup.PopupForm = testPopupForm{}

type testScenarioStep struct {
	RequestMethod    string
	RequestPath      string
	ExpectedCode     int
	BodyValidator    func(t *testing.T, body string, stat map[string]string)
	RequestGenerator func(t *testing.T, stat map[string]string) any
	WaitAfter        time.Duration
	Before           func()
	After            func(stat map[string]string)
}

func createTestInspectionServer() (*inspection.InspectionTaskServer, error) {
	inspectionServer, err := inspection.NewServer()
	if err != nil {
		return nil, err
	}
	taskDefinitions := []task.Definition{
		task_test.MockProcessorTaskFromTaskID(inspection_task.BuilderGeneratorTask.ID().String(), history.NewBuilder(&ioconfig.IOConfig{
			ApplicationRoot: "/",
			DataDestination: "/tmp/",
			TemporaryFolder: "/tmp/",
		})),
		inspection_task.NewInspectionProcessor("neverend", []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
			tp.Update(0.5, "test")
			select {
			case <-time.After(time.Hour * time.Duration(1000000)):
				return nil, nil
			case <-ctx.Done():
				return nil, nil
			}
		}, inspection_task.InspectionTypeLabel("foo", "bar", "qux")),
		task.NewProcessorTask("errorend", []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
			return nil, fmt.Errorf("test error")
		}, inspection_task.InspectionTypeLabel("foo", "bar", "qux")),
		form.NewInputFormDefinitionBuilder("foo-input", 0, "A input field for foo").WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
			if value == "foo-input-invalid-value" {
				return "invalid value", nil
			}
			return "", nil
		}).Build(inspection_task.InspectionTypeLabel("foo")),
		task_test.MockProcessorTaskFromTaskID(gcp_task.TimeZoneShiftInputTaskID, time.UTC),
		form.NewInputFormDefinitionBuilder("bar-input", 1, "A input field for bar").Build(inspection_task.InspectionTypeLabel("bar")),
		inspection_task.NewInspectionProcessor("feature-foo1", []string{"foo-input"}, func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
			return "feature-foo1-value", nil
		}, inspection_task.InspectionTypeLabel("foo"), inspection_task.FeatureTaskLabel("foo feature1", "test-feature", false)),
		inspection_task.NewInspectionProcessor("feature-foo2", []string{"foo-input"}, func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
			return "feature-foo2-value", nil
		}, inspection_task.InspectionTypeLabel("foo"), inspection_task.FeatureTaskLabel("foo feature2", "test-feature", false)),
		inspection_task.NewInspectionProcessor("feature-bar", []string{"bar-input", "neverend"}, func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
			return "feature-bar1-value", nil
		}, inspection_task.InspectionTypeLabel("bar"), inspection_task.FeatureTaskLabel("bar feature1", "test-feature", false)),
		inspection_task.NewInspectionProcessor("feature-qux", []string{"errorend"}, func(ctx context.Context, taskMode int, v *task.VariableSet, tp *progress.TaskProgress) (any, error) {
			return "feature-bar1-value", nil
		}, inspection_task.InspectionTypeLabel("qux"), inspection_task.FeatureTaskLabel("qux feature1", "test-feature", false)),
		ioconfig.TestIOConfig,
	}

	for _, def := range taskDefinitions {
		err = inspectionServer.AddTaskDefinition(def)
		if err != nil {
			return nil, err
		}
	}
	inspectionTypes := []inspection.InspectionType{
		{
			Id:          "foo",
			Name:        "foo-name",
			Description: "foo-description",
			Icon:        "foo-icon",
			Priority:    1,
		},
		{
			Id:          "bar",
			Name:        "bar-name",
			Description: "bar-description",
			Icon:        "bar-icon",
			Priority:    2,
		},
		{
			Id:          "qux",
			Name:        "qux-name",
			Description: "qux-description",
			Icon:        "qux-icon",
			Priority:    3,
		},
	}
	for _, t := range inspectionTypes {
		err = inspectionServer.AddInspectionType(t)
		if err != nil {
			return nil, err
		}
	}
	return inspectionServer, nil
}

func bodyCompareWithStringExpectedValue(expected string, options ...cmp.Option) func(t *testing.T, body string, stat map[string]string) {
	return func(t *testing.T, body string, stat map[string]string) {
		if diff := cmp.Diff(expected, body, options...); diff != "" {
			t.Errorf("the result is not matching with the expected response\n%s\nexpected:\n%s\nactual:%s", diff, expected, body)
		}
	}
}

func bodyCompareWithStruct[T any](expected *T, options ...cmp.Option) func(t *testing.T, body string, stat map[string]string) {
	return func(t *testing.T, body string, stat map[string]string) {
		parsedActual := new(T)
		err := json.Unmarshal([]byte(body), parsedActual)
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		if diff := cmp.Diff(expected, parsedActual, options...); diff != "" {
			t.Errorf("the result is not matching with the expected response\n%s", diff)
		}
	}
}

func metadataIgnoredBodyCompare(expected string, ignoredMetadata ...string) func(t *testing.T, body string, stat map[string]string) {
	return func(t *testing.T, body string, stat map[string]string) {
		var unmarshalledResponse struct {
			Metadata map[string]interface{} `json:"metadata"`
		} = struct {
			Metadata map[string]interface{} "json:\"metadata\""
		}{Metadata: map[string]interface{}{}}
		err := json.Unmarshal([]byte(body), &unmarshalledResponse)
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		for _, ignore := range ignoredMetadata {
			delete(unmarshalledResponse.Metadata, ignore)
		}
		filteredResponseBinary, err := json.Marshal(&unmarshalledResponse)
		if err != nil {
			t.Errorf("unexpected error\n%v", err)
		}
		bodyCompareWithStringExpectedValue(expected)(t, string(filteredResponseBinary), stat)
	}
}

func taskCompare(taskPlaceholder string, expected string, ignoredMetadata ...string) func(t *testing.T, body string, stat map[string]string) {
	return func(t *testing.T, body string, stat map[string]string) {
		var response GetInspectionTasksResponse

		err := json.Unmarshal([]byte(body), &response)
		if err != nil {
			t.Errorf("The response is not parsable\n%s", err)
		}
		taskId := stat[taskPlaceholder]
		for _, ignored := range ignoredMetadata {
			delete(response.Tasks[taskId], ignored)
		}
		serialized, err := json.Marshal(response.Tasks[taskId])
		if err != nil {
			t.Errorf("The task is not serializable\n%s", err)
		}
		if string(serialized) != expected {
			t.Errorf("the result is not matching with the expected response\n%s\n\n%s", expected, serialized)
		}
	}
}

func TestApiResponses(t *testing.T) {
	logger.InitGlobalKHILogger()
	inspectionServer, err := createTestInspectionServer()
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}
	serverConfig := ServerConfig{
		ViewerMode:       false,
		StaticFolderPath: "../../dist",
		ResourceMonitor:  &ResourceMonitorMock{UsedMemory: 1000},
		ServerBasePath:   "/foo",
	}
	engine := CreateKHIServer(inspectionServer, &serverConfig)

	// Perform requests with following oinvalidrder and verify if responses are matching with the expected values.
	scenarioSteps := []testScenarioStep{
		{
			// 000
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/types",
			BodyValidator: bodyCompareWithStringExpectedValue(`{"types":[{"id":"qux","name":"qux-name","description":"qux-description","icon":"qux-icon"},{"id":"bar","name":"bar-name","description":"bar-description","icon":"bar-icon"},{"id":"foo","name":"foo-name","description":"foo-description","icon":"foo-icon"}]}`),
		},
		{
			// 001
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks",
			BodyValidator: bodyCompareWithStringExpectedValue(`{"tasks":{},"serverStat":{"totalMemoryAvailable":1000}}`),
		},
		{
			// 002
			ExpectedCode:  404,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/types/not-existing-task",
		},
		{
			// 003
			ExpectedCode:  202,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/types/foo",
			BodyValidator: func(t *testing.T, body string, stat map[string]string) {
				var response PostInspectionTaskResponse
				err := json.Unmarshal([]byte(body), &response)
				if err != nil {
					t.Errorf("failed to decode response json\n%v", err)
				}
				stat["task-1"] = response.InspectionId
			},
		},
		{
			// 004
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/features",
			BodyValidator: bodyCompareWithStringExpectedValue(`{"features":[{"id":"feature-foo1","label":"foo feature1","description":"test-feature","enabled":false},{"id":"feature-foo2","label":"foo feature2","description":"test-feature","enabled":false}]}`),
		},
		{
			// 005
			ExpectedCode:  202,
			RequestMethod: "PUT",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/features",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return PutInspectionTaskFeatureRequest{
					Features: []string{
						"feature-foo2",
					},
				}
			},
			BodyValidator: bodyCompareWithStringExpectedValue(`ok`),
		},
		{
			// 006
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/features",
			BodyValidator: bodyCompareWithStringExpectedValue(`{"features":[{"id":"feature-foo1","label":"foo feature1","description":"test-feature","enabled":false},{"id":"feature-foo2","label":"foo feature2","description":"test-feature","enabled":true}]}`),
		},
		{
			// 007
			// Dryrun without any parameter
			ExpectedCode:  200,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/dryrun",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return map[string]any{}
			},
			BodyValidator: metadataIgnoredBodyCompare(`{"metadata":{"form":[{"allowEdit":true,"default":"","description":"","hint":"","hintType":"info","id":"foo-input","label":"A input field for foo","suggestions":null,"type":"Text","validationError":""}]}}`, "plan"),
		},
		{
			// 008
			// Dryrun with a value without a validation error
			ExpectedCode:  200,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/dryrun",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return map[string]any{
					"foo-input": "foo-input-value",
				}
			},
			BodyValidator: metadataIgnoredBodyCompare(`{"metadata":{"form":[{"allowEdit":true,"default":"","description":"","hint":"","hintType":"info","id":"foo-input","label":"A input field for foo","suggestions":null,"type":"Text","validationError":""}]}}`, "plan"),
		},
		{
			// 009
			// Dryrun with a value with a validation error
			ExpectedCode:  200,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/dryrun",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return map[string]any{
					"foo-input": "foo-input-invalid-value",
				}
			},
			BodyValidator: metadataIgnoredBodyCompare(`{"metadata":{"form":[{"allowEdit":true,"default":"","description":"","hint":"","hintType":"info","id":"foo-input","label":"A input field for foo","suggestions":null,"type":"Text","validationError":"invalid value"}]}}`, "plan"),
		}, {
			// 010
			// Attempting to access non started task result
			ExpectedCode:  400,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/data",
			BodyValidator: bodyCompareWithStringExpectedValue("this task is not yet started"),
		},
		{
			// 011
			// Attempting to access non started task metadata
			ExpectedCode:  400,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/metadata",
			BodyValidator: bodyCompareWithStringExpectedValue("this task is not yet started"),
		},
		{
			// 012
			// Attempting to cancel non started task result
			ExpectedCode:  400,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/cancel",
			BodyValidator: bodyCompareWithStringExpectedValue("this task is not yet started"),
		},
		{
			// 013
			ExpectedCode:  202,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/run",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return map[string]any{
					"foo-input": "foo-input-value",
				}
			},
			BodyValidator: bodyCompareWithStringExpectedValue("ok"),
			WaitAfter:     time.Second,
		},
		{
			// 014
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks",
			BodyValidator: taskCompare("task-1", `{"error":{"errorMessages":[]},"progress":{"phase":"DONE","progresses":[],"totalProgress":{"id":"Total","indeterminate":false,"label":"Total","message":"2 of 2 tasks complete","percentage":1}}}`, "header"),
		},
		{
			// 015
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/metadata",
		},
		{
			// 016
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/data",
			BodyValidator: func(t *testing.T, body string, stat map[string]string) {
				if !strings.HasPrefix(body, "KHI") {
					t.Errorf("the inspection data is not starting with KHI magic bytes\n%s", body)
				}
			},
		},
		{
			// 017
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/data?start=1",
			BodyValidator: func(t *testing.T, body string, stat map[string]string) {
				if !strings.HasPrefix(body, "HI") {
					t.Errorf("server didn't respond data with respecting start query parameter\n%s", body)
				}
			},
		},
		{
			// 018
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/data?start=1&maxSize=1",
			BodyValidator: func(t *testing.T, body string, stat map[string]string) {
				if body != "H" {
					t.Errorf("server didn't respond data with respecting start query and max size parameter\n%s", body)
				}
			},
		},
		{
			// 019
			ExpectedCode:  400,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-1>/cancel",
		},
		{
			// 020
			ExpectedCode:  202,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/types/bar",
			BodyValidator: func(t *testing.T, body string, stat map[string]string) {
				var response PostInspectionTaskResponse
				err := json.Unmarshal([]byte(body), &response)
				if err != nil {
					t.Errorf("failed to decode response json\n%v", err)
				}
				stat["task-2"] = response.InspectionId
			},
		},
		{
			// 021
			ExpectedCode:  202,
			RequestMethod: "PUT",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-2>/features",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return PutInspectionTaskFeatureRequest{
					Features: []string{
						"feature-bar",
					},
				}
			},
			BodyValidator: bodyCompareWithStringExpectedValue(`ok`),
		},
		{
			// 022
			ExpectedCode:  202,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-2>/run",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return map[string]any{}
			},
			BodyValidator: bodyCompareWithStringExpectedValue("ok"),
			WaitAfter:     time.Second,
		},
		{
			// 023
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks",
			BodyValidator: taskCompare("task-2", `{"error":{"errorMessages":[]},"progress":{"phase":"RUNNING","progresses":[{"id":"neverend","indeterminate":false,"label":"neverend","message":"test","percentage":0.5}],"totalProgress":{"id":"Total","indeterminate":false,"label":"Total","message":"0 of 3 tasks complete","percentage":0}}}`, "header"),
		},
		{
			// 024
			ExpectedCode:  400,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-2>/data",
			BodyValidator: bodyCompareWithStringExpectedValue("this task runner hasn't finished yet"),
		},
		{
			// 025
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-2>/metadata",
		},
		{
			// 026
			ExpectedCode:  200,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-2>/cancel",
			WaitAfter:     time.Second,
		},
		{
			// 027
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks",
			BodyValidator: taskCompare("task-2", `{"error":{"errorMessages":[]},"progress":{"phase":"CANCELLED","progresses":[],"totalProgress":{"id":"Total","indeterminate":false,"label":"Total","message":"1 of 3 tasks complete","percentage":0.33333334}}}`, "header"),
		}, {
			// 028
			ExpectedCode:  202,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/types/qux",
			BodyValidator: func(t *testing.T, body string, stat map[string]string) {
				var response PostInspectionTaskResponse
				err := json.Unmarshal([]byte(body), &response)
				if err != nil {
					t.Errorf("failed to decode response json\n%v", err)
				}
				stat["task-3"] = response.InspectionId
			},
		},
		{
			// 029
			ExpectedCode:  202,
			RequestMethod: "PUT",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-3>/features",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return PutInspectionTaskFeatureRequest{
					Features: []string{
						"feature-qux",
					},
				}
			},
			BodyValidator: bodyCompareWithStringExpectedValue(`ok`),
		},
		{
			// 030
			ExpectedCode:  202,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/inspection/tasks/<task-3>/run",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return map[string]any{}
			},
			BodyValidator: bodyCompareWithStringExpectedValue("ok"),
			WaitAfter:     time.Second,
		},
		{
			// 031
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/inspection/tasks",
			BodyValidator: taskCompare("task-3", `{"error":{"errorMessages":[]},"progress":{"phase":"ERROR","progresses":[],"totalProgress":{"id":"Total","indeterminate":false,"label":"Total","message":"0 of 2 tasks complete","percentage":0}}}`, "header"),
		},
		{
			// 032
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/popup",
			BodyValidator: bodyCompareWithStringExpectedValue(""),
			After: func(stat map[string]string) {
				go func() {
					popup.Instance.ShowPopup(testPopupForm{})
				}()
				<-time.After(time.Second)
				p := popup.Instance.GetCurrentPopup()
				stat["popup-id"] = p.Id
			},
		},
		{
			// 033
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/popup",
			BodyValidator: bodyCompareWithStruct(
				&popup.PopupFormRequest{
					Title:       "foo",
					Type:        "bar",
					Description: "baz",
				},
				cmpopts.IgnoreFields(popup.PopupFormRequest{}, "Id"),
			),
		},
		{
			// 034
			ExpectedCode:  200,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/popup/validate",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return popup.PopupAnswerResponse{
					Id:    stat["popup-id"],
					Value: "ng",
				}
			},
			BodyValidator: bodyCompareWithStruct(
				&popup.PopupAnswerValidationResult{
					ValidationError: "answer for test popup must contain ok",
				}, cmpopts.IgnoreFields(popup.PopupAnswerValidationResult{}, "Id"),
			),
		},
		{
			// 035
			ExpectedCode:  200,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/popup/validate",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return popup.PopupAnswerResponse{
					Id:    stat["popup-id"],
					Value: "ok",
				}
			},
			BodyValidator: bodyCompareWithStruct(
				&popup.PopupAnswerValidationResult{
					ValidationError: "",
				}, cmpopts.IgnoreFields(popup.PopupAnswerValidationResult{}, "Id"),
			),
		},
		{
			// 036
			ExpectedCode:  400,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/popup/validate",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return popup.PopupAnswerResponse{
					Id:    "non-valid-id",
					Value: "ok",
				}
			},
			BodyValidator: bodyCompareWithStringExpectedValue("given id is not matching with the current popup"),
		},
		{
			// 037
			ExpectedCode:  400,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/popup/answer",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return popup.PopupAnswerResponse{
					Id:    "non-valid-id",
					Value: "ok",
				}
			},
			BodyValidator: bodyCompareWithStringExpectedValue("given id is not matching with the current popup"),
		},
		{
			// 038
			ExpectedCode:  200,
			RequestMethod: "POST",
			RequestPath:   "/foo/api/v2/popup/answer",
			RequestGenerator: func(t *testing.T, stat map[string]string) any {
				return popup.PopupAnswerResponse{
					Id:    stat["popup-id"],
					Value: "ok",
				}
			},
			BodyValidator: bodyCompareWithStringExpectedValue(""),
			After: func(stat map[string]string) {
				delete(stat, "popup-id")
			},
		},
		{
			// 039
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/popup",
			BodyValidator: bodyCompareWithStringExpectedValue(""),
		},
		{
			// 040
			ExpectedCode:  200,
			RequestMethod: "GET",
			RequestPath:   "/foo/api/v2/config",
			Before: func() {
				parameters.Server.ViewerMode = testutil.P(true)
			},
			BodyValidator: bodyCompareWithStruct(&config.GetConfigResponse{
				ViewerMode: true,
			}),
		},
	}

	stat := map[string]string{}
	for i, step := range scenarioSteps {
		t.Run(fmt.Sprintf("step-%03d-%d-%s %s", i, step.ExpectedCode, step.RequestMethod, step.RequestPath), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			var requestReader io.Reader
			if step.RequestGenerator != nil {
				payload := step.RequestGenerator(t, stat)
				request, err := json.Marshal(payload)
				if err != nil {
					t.Errorf("unexpected error\n%v", err)
				}
				requestReader = bytes.NewReader(request)
			}
			path := step.RequestPath
			TASK_COUNT := 3
			for i := 0; i < TASK_COUNT; i++ {
				path = strings.ReplaceAll(path, fmt.Sprintf("<task-%d>", i+1), stat[fmt.Sprintf("task-%d", i+1)])
			}
			if step.Before != nil {
				step.Before()
			}
			req, _ := http.NewRequest(step.RequestMethod, path, requestReader)
			engine.ServeHTTP(recorder, req)
			if step.ExpectedCode != recorder.Code {
				t.Errorf("expected %d, actual: %d\n%s", step.ExpectedCode, recorder.Code, recorder.Body)
			}
			if step.BodyValidator != nil {
				step.BodyValidator(t, recorder.Body.String(), stat)
			}
			if step.After != nil {
				step.After(stat)
			}
			<-time.After(step.WaitAfter)
		})
	}
}

func TestKHIServer_EndpointExistsWithConfigs(t *testing.T) {
	testCases := []struct {
		name           string
		serverBasePath string
		viewerMode     bool
		requestMethod  string
		requestPath    string
		wantCode       int
	}{
		{
			name:           "custom server base path on non-viewer mode",
			serverBasePath: "/custom/base/path/foo",
			requestMethod:  "GET",
			requestPath:    "/custom/base/path/foo/api/v2/inspection/types",
			wantCode:       200,
		},
		{
			name:          "viewer mode should serve the static resource",
			viewerMode:    true,
			requestMethod: "GET",
			requestPath:   "/session/100",
			wantCode:      200,
		},
		{
			name:          "static resource must be served",
			requestMethod: "GET",
			requestPath:   "/test.html",
			wantCode:      200,
		},
		{
			name:           "static resource must be served with server base path",
			serverBasePath: "/custom/base/path/foo",
			requestMethod:  "GET",
			requestPath:    "/custom/base/path/foo/test.html",
			wantCode:       200,
		},
		{
			name:          "viewer mode shouldn't serve task related endpoints",
			viewerMode:    true,
			requestMethod: "GET",
			requestPath:   "/api/v2/inspection/tasks",
			wantCode:      404,
		},
		{
			name:           "viewer mode should serve the static resource with custom server base path",
			viewerMode:     true,
			serverBasePath: "/custom/base/path/foo",
			requestMethod:  "GET",
			requestPath:    "/custom/base/path/foo/session/100",
			wantCode:       200,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.InitGlobalKHILogger()
			inspectionServer, err := createTestInspectionServer()
			if err != nil {
				t.Fatalf("unexpected error %s", err)
			}
			defer testutil.MustPlaceTemporalFile("../../dist/test.html", "")()
			recorer := httptest.NewRecorder()
			config := ServerConfig{
				ViewerMode:       tc.viewerMode,
				StaticFolderPath: "../../dist",
				ResourceMonitor:  &ResourceMonitorMock{UsedMemory: 1000},
				ServerBasePath:   tc.serverBasePath,
			}
			engine := CreateKHIServer(inspectionServer, &config)
			req, _ := http.NewRequest(tc.requestMethod, tc.requestPath, bytes.NewReader([]byte{}))
			engine.ServeHTTP(recorer, req)
			if recorer.Code != tc.wantCode {
				t.Errorf("got response code %d, want %d", recorer.Code, tc.wantCode)
			}
		})
	}
}

func TestKHIServerRedirects(t *testing.T) {
	testCases := []struct {
		name           string
		serverBasePath string
		viewerMode     bool
		requestMethod  string
		requestPath    string
		wantCode       int
		redirectTo     string
	}{
		{
			name:          "the root path should be redirected to the default session path",
			viewerMode:    false,
			requestMethod: "GET",
			requestPath:   "/",
			redirectTo:    "/session/0",
			wantCode:      302,
		},
		{
			name:           "the root path should be redirected to the default session path with custom server base path",
			viewerMode:     false,
			serverBasePath: "/custom/base/path",
			requestMethod:  "GET",
			requestPath:    "/custom/base/path/",
			redirectTo:     "/custom/base/path/session/0",
			wantCode:       302,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.InitGlobalKHILogger()
			inspectionServer, err := createTestInspectionServer()
			if err != nil {
				t.Fatalf("unexpected error %s", err)
			}
			recorer := httptest.NewRecorder()
			config := ServerConfig{
				ViewerMode:       tc.viewerMode,
				StaticFolderPath: "../../dist",
				ResourceMonitor:  &ResourceMonitorMock{UsedMemory: 1000},
				ServerBasePath:   tc.serverBasePath,
			}
			engine := CreateKHIServer(inspectionServer, &config)
			req, _ := http.NewRequest(tc.requestMethod, tc.requestPath, bytes.NewReader([]byte{}))
			engine.ServeHTTP(recorer, req)
			if recorer.Code != tc.wantCode {
				t.Errorf("got response code %d, want %d", recorer.Code, tc.wantCode)
			}
			gotRedirectTo := recorer.Result().Header.Get("Location")
			if gotRedirectTo != tc.redirectTo {
				t.Errorf("got redirect to %s, want %s", gotRedirectTo, tc.redirectTo)
			}
		})
	}
}
