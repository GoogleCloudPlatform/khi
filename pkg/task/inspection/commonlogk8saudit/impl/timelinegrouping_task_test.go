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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

type stubAuditLogFieldExtractor struct {
	Extractor func(ctx context.Context, log *log.Log) (*commonlogk8saudit_contract.AuditLogParserInput, error)
}

// ExtractFields implements commonlogk8saudit_contract.AuditLogFieldExtractor.
func (f *stubAuditLogFieldExtractor) ExtractFields(ctx context.Context, log *log.Log) (*commonlogk8saudit_contract.AuditLogParserInput, error) {
	return f.Extractor(ctx, log)
}

var _ commonlogk8saudit_contract.AuditLogFieldExtractor = (*stubAuditLogFieldExtractor)(nil)

// mustTestYAMLReader returns structured.NodeReader from given string.
func mustTestYAMLReader(t *testing.T, yaml string) *structured.NodeReader {
	n, err := structured.FromYAML(yaml)
	if err != nil {
		t.Fatalf("failed to parse yaml %s\n%s", err.Error(), yaml)
	}
	return structured.NewNodeReader(n)
}

func TestDefaultGroupDecider(t *testing.T) {
	decider := &defaultResourceGroupDecider{}
	testCases := []struct {
		desc      string
		inputOp   model.KubernetesObjectOperation
		wantGroup string
	}{
		{
			desc: "with a basic pod resource",
			inputOp: model.KubernetesObjectOperation{
				APIVersion: "core/v1",
				PluralKind: "pods",
				Namespace:  "default",
				Name:       "foo",
				Verb:       enum.RevisionVerbCreate,
			},
			wantGroup: "core/v1#pod#default#foo",
		},
		{
			desc: "with namepsace deleting delete collection",
			inputOp: model.KubernetesObjectOperation{
				APIVersion: "core/v1",
				PluralKind: "pods",
				Namespace:  "default",
				Name:       "",
				Verb:       enum.RevisionVerbDeleteCollection,
			},
			wantGroup: "core/v1#pod#default#", // TODO: This is OK for now. This will be fixed with #267 'Changes made by delete collection operation may generate wrong resource timeline'
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := decider.GetResourceGroup(&commonlogk8saudit_contract.AuditLogParserInput{
				Operation: &tc.inputOp,
			})
			if err != nil {
				t.Error(err)
			}
			if group != tc.wantGroup {
				t.Errorf("got %s, want %s", group, tc.wantGroup)
			}
		})
	}
}

func TestSubresourceGroupDecider(t *testing.T) {
	decider := &subresourceResourceGroupDecider{
		defaultBehaviorOverrides: map[string]subresourceDefaultBehavior{
			"status": Parent,
		},
	}

	testCases := []struct {
		desc                string
		inputOp             model.KubernetesObjectOperation
		inputRequestReader  *structured.NodeReader
		inputResponseReader *structured.NodeReader
		wantGroup           string
	}{
		{
			desc: "must ignore non subresource group",
			inputOp: model.KubernetesObjectOperation{
				APIVersion: "core/v1",
				PluralKind: "pods",
				Namespace:  "default",
				Name:       "foo",
				Verb:       enum.RevisionVerbCreate,
			},
			wantGroup: "",
		},
		{
			desc: "subresource respond with its parent resource",
			inputOp: model.KubernetesObjectOperation{
				APIVersion:      "certificates.k8s.io/v1",
				PluralKind:      "certificatesigningrequests",
				Namespace:       "default",
				Name:            "foo",
				SubResourceName: "approve",
				Verb:            enum.RevisionVerbPatch,
			},
			inputRequestReader: nil,
			inputResponseReader: mustTestYAMLReader(t, `apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
`),
			wantGroup: "certificates.k8s.io/v1#certificatesigningrequest#default#foo",
		},
		{
			desc: "subresource respond with its subresource",
			inputOp: model.KubernetesObjectOperation{
				APIVersion:      "v1",
				PluralKind:      "pods",
				Namespace:       "default",
				Name:            "foo",
				SubResourceName: "binding",
				Verb:            enum.RevisionVerbPatch,
			},
			inputRequestReader: nil,
			inputResponseReader: mustTestYAMLReader(t, `apiVersion: v1
kind: Binding`),
			wantGroup: "core/v1#pod#default#foo#binding",
		},
		{
			desc: "subresource respond with status resource",
			inputOp: model.KubernetesObjectOperation{
				APIVersion:      "v1",
				PluralKind:      "pods",
				Namespace:       "default",
				Name:            "foo",
				SubResourceName: "binding",
				Verb:            enum.RevisionVerbPatch,
			},
			inputResponseReader: mustTestYAMLReader(t, `apiVersion: v1
kind: Status
`),
			inputRequestReader: mustTestYAMLReader(t, `apiVersion: v1
kind: Binding`),
			wantGroup: "core/v1#pod#default#foo#binding",
		},
		{
			desc: "request and response wasn't enough informative to determine its associated resource type and its default behavior was overriden to use Parent",
			inputOp: model.KubernetesObjectOperation{
				APIVersion:      "v1",
				PluralKind:      "pods",
				Namespace:       "default",
				Name:            "foo",
				SubResourceName: "status",
				Verb:            enum.RevisionVerbPatch,
			},
			inputRequestReader: mustTestYAMLReader(t, `metadata:
  uid: foobar
status:
  phase: Running
`),
			inputResponseReader: nil,
			wantGroup:           "core/v1#pod#default#foo",
		},
		{
			desc: "request and response wasn't enough informative to determine its associated resource type and its default behavior wasn't overriden",
			inputOp: model.KubernetesObjectOperation{
				APIVersion:      "foo/v1",
				PluralKind:      "bars",
				Namespace:       "default",
				Name:            "qux",
				SubResourceName: "quux",
				Verb:            enum.RevisionVerbPatch,
			},
			inputRequestReader: mustTestYAMLReader(t, `metadata:
  uid: abcdefg
status:
  phase: Running
`),
			inputResponseReader: nil,
			wantGroup:           "foo/v1#bar#default#qux#quux",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			group, err := decider.GetResourceGroup(&commonlogk8saudit_contract.AuditLogParserInput{
				Operation: &tc.inputOp,
				Request:   tc.inputRequestReader,
				Response:  tc.inputResponseReader,
			})
			if err != nil {
				t.Error(err)
			}
			if group != tc.wantGroup {
				t.Errorf("got %s, want %s", group, tc.wantGroup)
			}
		})
	}
}

func TestGroupByTimelineTask(t *testing.T) {
	stubExtractor := &stubAuditLogFieldExtractor{
		Extractor: func(ctx context.Context, log *log.Log) (*commonlogk8saudit_contract.AuditLogParserInput, error) {
			resourceName := log.ReadStringOrDefault("protoPayload.resourceName", "")
			var podName, subresourceName string
			switch resourceName {
			case "core/v1/namespaces/default/pods/foo/binding":
				podName = "foo"
				subresourceName = "binding"
			case "core/v1/namespaces/default/pods/foo":
				podName = "foo"
			case "core/v1/namespaces/default/pods/bar":
				podName = "bar"
			}
			return &commonlogk8saudit_contract.AuditLogParserInput{
				Log: log,
				Operation: &model.KubernetesObjectOperation{
					APIVersion:      "core/v1",
					PluralKind:      "pods",
					Namespace:       "default",
					Name:            podName,
					SubResourceName: subresourceName,
					Verb:            enum.RevisionVerbCreate,
				},
			}, nil
		},
	}
	t.Run("it ignores dryrun mode", func(t *testing.T) {
		ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
		result, _, err := inspectiontest.RunInspectionTask(ctx, TimelineGroupingTask, inspectioncore_contract.TaskModeDryRun, map[string]any{},
			tasktest.NewTaskDependencyValuePair(commonlogk8saudit_contract.CommonLogParseTaskID.Ref(), nil))
		if err != nil {
			t.Error(err)
		}
		if result != nil {
			t.Errorf("the result is not valid")
		}
	})

	t.Run("it grups logs by timleines", func(t *testing.T) {
		baseLog := `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  status:
    code: 200
timestamp: 2024-01-01T00:00:00+09:00`
		logOpts := [][]testlog.TestLogOpt{
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/foo"),
			},
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/foo"),
			},
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/bar"),
			},
		}
		expectedLogCounts := map[string]int{
			"core/v1#pod#default#foo": 2,
			"core/v1#pod#default#bar": 1,
		}
		tl := testlog.New(testlog.YAML(baseLog))
		logs := []*log.Log{}
		for _, opt := range logOpts {
			logs = append(logs, tl.With(opt...).MustBuildLogEntity())
		}

		ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
		result, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, TimelineGroupingTask, []coretask.UntypedTask{
			CommonLogParserTask,
			tasktest.StubTaskFromReferenceID(commonlogk8saudit_contract.CommonAuitLogSource, &commonlogk8saudit_contract.AuditLogParserLogSource{
				Logs:      logs,
				Extractor: stubExtractor,
			}, nil),
		}, inspectioncore_contract.TaskModeRun, map[string]any{})
		if err != nil {
			t.Error(err)
		}
		for _, result := range result {
			if count, found := expectedLogCounts[result.TimelineResourcePath]; !found {
				t.Errorf("unexpected timeline %s not found", result.TimelineResourcePath)
			} else if count != len(result.PreParsedLogs) {
				t.Errorf("expected log count is not matching in a timeline:%s", result.TimelineResourcePath)
			}
		}
	})

	t.Run("generates its result with sorting timeline", func(t *testing.T) {
		baseLog := `insertId: foo
protoPayload:
  authenticationInfo:
    principalEmail: user@example.com
  methodName: io.k8s.core.v1.pods.create
  status:
    code: 200
timestamp: 2024-01-01T00:00:00+09:00`
		logOpts := [][]testlog.TestLogOpt{
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/foo"),
			},
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/bar"),
			},
			{
				testlog.StringField("protoPayload.resourceName", "core/v1/namespaces/default/pods/foo/binding"),
			},
		}
		tl := testlog.New(testlog.YAML(baseLog))
		logs := []*log.Log{}
		for _, opt := range logOpts {
			logs = append(logs, tl.With(opt...).MustBuildLogEntity())
		}
		wantTimelineInOrder := []string{
			"core/v1#pod#default#bar",
			"core/v1#pod#default#foo",
			"core/v1#pod#default#foo#binding",
		}

		ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
		gotGrouperResult, _, err := inspectiontest.RunInspectionTaskWithDependency(ctx, TimelineGroupingTask, []coretask.UntypedTask{
			CommonLogParserTask,
			tasktest.StubTaskFromReferenceID(commonlogk8saudit_contract.CommonAuitLogSource, &commonlogk8saudit_contract.AuditLogParserLogSource{
				Logs:      logs,
				Extractor: stubExtractor,
			}, nil),
		}, inspectioncore_contract.TaskModeRun, map[string]any{})
		if err != nil {
			t.Error(err)
		}

		if len(gotGrouperResult) != len(wantTimelineInOrder) {
			t.Fatalf("the count of timeline is not matching: got %d, want %d", len(gotGrouperResult), len(wantTimelineInOrder))
		}
		for i, g := range gotGrouperResult {
			if g.TimelineResourcePath != wantTimelineInOrder[i] {
				t.Errorf("the order of timeline is not matching at index %d: got %s, want %s", i, g.TimelineResourcePath, wantTimelineInOrder[i])
			}
		}
	})
}
