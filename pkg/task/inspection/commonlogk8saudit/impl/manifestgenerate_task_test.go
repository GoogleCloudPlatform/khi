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

package commonlogk8saudit_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/log"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	base_task "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/impl"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/impl/fieldextractor"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
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
				testlog.IntField("protoPayload.status.code", 1),
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
	}, {
		Name: "Patch operation with uid changing in middle",
		baseLog: `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.patch
  resourceName: core/v1/namespaces/default/pods/my-pod
  request:
    '@type': k8s.io/Patch
    metadata:
      uid: foo
  status:
    code: 0
timestamp: 2024-01-01T00:00:00+09:00`,
		logOpts: [][]testlog.TestLogOpt{{
			testlog.StringField("protoPayload.request.metadata.labels.foo", "bar"),
		}, {
			testlog.StringField("protoPayload.request.metadata.labels.bar", "qux"), // This patch newly adds metadata.labels.bar
			testlog.StringField("protoPayload.request.metadata.uid", "bar"),        // But this resource uid was changed, thus the previous labels.foo=bar must not be retained
		}},
		expectedComment:  []string{"", ""},
		expectedBodyBase: "",
		expectedBodyOpts: [][]testlog.TestLogOpt{{
			testlog.StringField("metadata.uid", "foo"),
			testlog.StringField("metadata.labels.foo", "bar"),
		}, {
			testlog.StringField("metadata.uid", "bar"),
			testlog.StringField("metadata.labels.bar", "qux"),
		}},
	}}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			logs := []*log.Log{}
			logBase := testlog.New(testlog.YAML(tc.baseLog))
			for _, logOpts := range tc.logOpts {
				logs = append(logs, logBase.With(logOpts...).MustBuildLogEntity(&gcpqueryutil.GCPCommonFieldSetReader{}, &gcpqueryutil.GCPMainMessageFieldSetReader{}))
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			result, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, ManifestGenerateTask, []base_task.UntypedTask{
				TimelineGroupingTask,
				CommonLogParserTask,
				tasktest.StubTaskFromReferenceID(commonlogk8saudit_contract.CommonAuitLogSource, &commonlogk8saudit_contract.AuditLogParserLogSource{
					Logs:      logs,
					Extractor: &fieldextractor.GCPAuditLogFieldExtractor{},
				}, nil),
				googlecloudk8scommon_impl.DefaultK8sResourceMergeConfigTask,
			}, inspectioncore_contract.TaskModeRun, map[string]any{})
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
			expectedBody := testlog.New(testlog.YAML(tc.expectedBodyBase))
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
