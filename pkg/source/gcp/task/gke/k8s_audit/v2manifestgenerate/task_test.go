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

package v2manifestgenerate

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/k8saudittask"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/v2commonlogparse"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/v2timelinegrouping"
	base_task "github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testtask"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestBodyMergerTask(t *testing.T) {
	var testCases = []struct {
		Name             string
		baseLog          string
		logOpts          [][]testlog.TestLogOpt
		expectedComment  []string
		expectedBodyBase string
		expectedBodyOpts [][]testlog.TestLogOpt
	}{{
		Name: "Standard non patching merge",
		baseLog: `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  resourceName: core/v1/namespaces/default/pods/my-pod
  response:
    '@type': core.k8s.io/v1.Pod
    foo: bar
  status:
    code: 0
timestamp: 2024-01-01T00:00:00+09:00`,
		logOpts: [][]testlog.TestLogOpt{
			{
				testlog.StringField("protoPayload.response.foo", "bar1"),
			},
			{
				testlog.StringField("protoPayload.response.foo", "bar2"),
			},
		},
		expectedBodyBase: `foo: bar1
`,
		expectedBodyOpts: [][]testlog.TestLogOpt{
			{},
			{
				testlog.StringField("foo", "bar2"),
			},
		},
		expectedComment: []string{"", ""},
	}, {
		Name: "Standard patching merge",
		baseLog: `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  resourceName: core/v1/namespaces/default/pods/my-pod
  request:
    '@type': k8s.io/Patch
    foo: bar
  status:
    code: 0
timestamp: 2024-01-01T00:00:00+09:00`,
		logOpts: [][]testlog.TestLogOpt{
			{
				testlog.StringField("protoPayload.request.qux", "qux1"),
			},
			{
				testlog.StringField("protoPayload.request.quux", "quux1"),
			},
		},
		expectedComment: []string{"", ""},
		expectedBodyBase: `foo: bar
qux: qux1`,
		expectedBodyOpts: [][]testlog.TestLogOpt{
			{},
			{
				testlog.StringField("quux", "quux1"),
			},
		},
	}, {
		Name: "Standard patching merge with middle ignored failed patch request",
		baseLog: `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  resourceName: core/v1/namespaces/default/pods/my-pod
  request:
    '@type': k8s.io/Patch
    foo: bar
  status:
    code: 0
timestamp: 2024-01-01T00:00:00+09:00`,
		logOpts: [][]testlog.TestLogOpt{
			{
				testlog.StringField("protoPayload.request.qux", "qux1"),
			},
			{
				testlog.StringField("protoPayload.request.qux", "qux2"),
				testlog.StringField("protoPayload.status.code", "1"),
			},
			{
				testlog.StringField("protoPayload.request.quux", "quux1"),
			},
		},
		expectedComment: []string{"", "", ""},
		expectedBodyBase: `foo: bar
qux: qux1
`,
		expectedBodyOpts: [][]testlog.TestLogOpt{{}, {}, {testlog.StringField("quux", "quux1")}},
	}, {
		Name: "response field should be ignored when it was deleteoption",
		baseLog: `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  resourceName: core/v1/namespaces/default/pods/my-pod
  request:
    '@type': k8s.io/Patch
    foo: bar
  response:
    '@type': meta.k8s.io/__internal.DeleteOptions
    foo: wrong
  status:
    code: 0
timestamp: 2024-01-01T00:00:00+09:00`,
		logOpts: [][]testlog.TestLogOpt{
			{
				testlog.StringField("protoPayload.request.qux", "qux1"),
			},
			{
				testlog.StringField("protoPayload.request.quux", "quux1"),
			},
		},
		expectedComment: []string{"", ""},
		expectedBodyBase: `foo: bar
qux: qux1
`,
		expectedBodyOpts: [][]testlog.TestLogOpt{{}, {testlog.StringField("quux", "quux1")}},
	}, {
		Name: "Metadata level audit logs",
		baseLog: `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  resourceName: core/v1/namespaces/default/pods/my-pod
  status:
    code: 0
timestamp: 2024-01-01T00:00:00+09:00`,
		logOpts:          [][]testlog.TestLogOpt{{}, {}},
		expectedComment:  []string{bodyPlaceholderForMetadataLevelAuditLog, bodyPlaceholderForMetadataLevelAuditLog},
		expectedBodyBase: bodyPlaceholderForMetadataLevelAuditLog,
		expectedBodyOpts: [][]testlog.TestLogOpt{{}, {}},
	}}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			logs := []*log.LogEntity{}
			logBase := testlog.New(testlog.BaseYaml(tc.baseLog))
			for _, logOpts := range tc.logOpts {
				logs = append(logs, logBase.With(logOpts...).MustBuildLogEntity(&log.UnreachableCommonFieldExtractor{}))
			}

			result, err := testtask.RunMultipleTask[[]*types.TimelineGrouperResult](Task, []base_task.Definition{
				v2timelinegrouping.Task,
				v2commonlogparse.Task,
				gcp_task.GCPDefaultK8sResourceMergeConfigTask,
			}, task.TaskModeRun,
				testtask.PriorTaskResultFromID(task.MetadataVariableName, typedmap.NewTypedMap().AsReadonly()),
				testtask.PriorTaskResultFromID(task.ReaderFactoryGeneratorTaskID, structure.NewReaderFactory(&structuredatastore.OnMemoryStructureDataStore{})),
				testtask.PriorTaskResultFromID(k8saudittask.K8sAuditQueryTaskID, logs))

			if err != nil {
				t.Error(err)
			}
			if len(result) != 1 {
				t.Errorf("unexpected timeline count: %d", len(result))
			}
			timeline := result[0]
			if len(timeline.PreParsedLogs) != len(tc.expectedBodyOpts) {
				t.Errorf("unexpected log count: %d but expected %d", len(timeline.PreParsedLogs), len(tc.expectedBodyOpts))
			}
			expectedBody := testlog.New(testlog.BaseYaml(tc.expectedBodyBase))
			for i, log := range timeline.PreParsedLogs {
				if tc.expectedComment[i] != "" {
					if diff := cmp.Diff(tc.expectedComment[i], log.ResourceBodyYaml); diff != "" {
						t.Errorf("the result is not valid at %d/%d:\n%s", i, len(tc.expectedBodyOpts), diff)
					}
				} else {
					if diff := cmp.Diff(expectedBody.With(tc.expectedBodyOpts[i]...).MustBuildYamlString(), log.ResourceBodyYaml); diff != "" {
						t.Errorf("the result is not valid at %d/%d:\n%s", i, len(tc.expectedBodyOpts), diff)
					}
				}
			}
		})
	}
}
