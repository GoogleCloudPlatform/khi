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

package composerapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"

	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func testReaderFromYAML(t *testing.T, yaml string) *structured.NodeReader {
	t.Helper()
	node, err := structured.FromYAML(yaml)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	return structured.NewNodeReader(node)
}

var compareNodeOption = cmp.Transformer("StructuredNodeToYAML", func(n structured.Node) string {
	if n == nil {
		return ""
	}
	serializer := &structured.YAMLNodeSerializer{}
	bytes, err := serializer.Serialize(n)
	if err != nil {
		return "serialization error"
	}
	return string(bytes)
})

func TestComposerAuditLogToTimelineMapper(t *testing.T) {
	testTime := time.Date(2026, time.August, 10, 0, 23, 12, 0, time.UTC)

	testCases := []struct {
		desc          string
		inputResource composerapiaudit.ComposerAuditLogResourceFieldSet
		inputAudit    gcpcommon.GCPAuditLogFieldSet
		setupTracker  func(tracker *gcpcommon.GCPOperationTracker, envPath *khifilev6.TimelinePath)
		assert        func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath)
	}{
		{
			desc: "CreateEnvironment operation started adds provisioning revision and operation start",
			inputResource: composerapiaudit.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-create-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
				Request: testReaderFromYAML(t, `environment:
  name: projects/test-project/locations/us-central1/environments/test-environment
  config:
    softwareConfig:
      imageVersion: composer-3-airflow-2`),
			},
			setupTracker: func(tracker *gcpcommon.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbCreate,
						StateType:   composercluster.RevisionStateManagedAirflowEnvironmentProvisioning,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `name: projects/test-project/locations/us-central1/environments/test-environment
config:
  softwareConfig:
    imageVersion: composer-3-airflow-2`).Node,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    gcpcommon.VerbOperationStart,
						StateType:   gcpcommon.RevisionStateOperationStarted,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `environment:
  name: projects/test-project/locations/us-central1/environments/test-environment
  config:
    softwareConfig:
      imageVersion: composer-3-airflow-2`).Node,
					}, compareNodeOption)
			},
		},
		{
			desc: "CreateEnvironment operation finished with start seen adds existing revision and operation succeed",
			inputResource: composerapiaudit.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-create-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
				Status:         0,
			},
			setupTracker: func(tracker *gcpcommon.GCPOperationTracker, envPath *khifilev6.TimelinePath) {
				tracker.MarkStarted("op-create-1")
				tracker.MarkResourceRevision(envPath)
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbCreate,
						StateType:    composercluster.RevisionStateManagedAirflowEnvironmentExisting,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    gcpcommon.VerbOperationFinish,
						StateType:   gcpcommon.RevisionStateOperationSucceed,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "CreateEnvironment operation finished without start log seen prepends LogNotFound revisions",
			inputResource: composerapiaudit.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-create-missing-start",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
				Status:         0,
			},
			setupTracker: func(tracker *gcpcommon.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbCreate,
						StateType:    composercluster.RevisionStateManagedAirflowEnvironmentProvisioningLogNotFound,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbCreate,
						StateType:    composercluster.RevisionStateManagedAirflowEnvironmentExisting,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    gcpcommon.VerbOperationStart,
						StateType:   gcpcommon.RevisionStateOperationStartedLogNotFound,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: time.Unix(0, 0),
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    gcpcommon.VerbOperationFinish,
						StateType:   gcpcommon.RevisionStateOperationSucceed,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "DeleteEnvironment operation started adds deleting revision",
			inputResource: composerapiaudit.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-delete-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.DeleteEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
			},
			setupTracker: func(tracker *gcpcommon.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbCreate,
						StateType:    composercluster.RevisionStateManagedAirflowEnvironmentExistingLogNotFound,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbDelete,
						StateType:    composercluster.RevisionStateManagedAirflowEnvironmentDeleting,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    gcpcommon.VerbOperationStart,
						StateType:   gcpcommon.RevisionStateOperationStarted,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "DeleteEnvironment operation finished with start seen adds deleted revision",
			inputResource: composerapiaudit.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-delete-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.DeleteEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
				Status:         0,
			},
			setupTracker: func(tracker *gcpcommon.GCPOperationTracker, envPath *khifilev6.TimelinePath) {
				tracker.MarkStarted("op-delete-1")
				tracker.MarkResourceRevision(envPath)
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbDelete,
						StateType:    composercluster.RevisionStateManagedAirflowEnvironmentDeleted,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    gcpcommon.VerbOperationFinish,
						StateType:   gcpcommon.RevisionStateOperationSucceed,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "immediate operation creates event on environment timeline",
			inputResource: composerapiaudit.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "",
				OperationFirst: false,
				OperationLast:  false,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.GetEnvironment",
				PrincipalEmail: "user@example.com",
			},
			setupTracker: func(tracker *gcpcommon.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(envPath)
			},
		},
	}

	mapperSetting := &composerAuditLogLogToTimelineMapperSetting{}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			builder := khifilev6.NewTestBuilder(id.NewGenerator())
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, tc.inputResource.ProjectID)
			envTimeline := gcpcommon.MustManagedAirflowEnvironmentTimeline(ctx, projectTimeline, tc.inputResource.EnvironmentName)

			var opTimeline *khifilev6.TimelinePath
			if !tc.inputAudit.ImmediateOperation() {
				opTimeline = gcpcommon.MustGCPOperationTimeline(ctx, envTimeline, "CreateEnvironment", tc.inputAudit.OperationID)
				if tc.inputAudit.MethodName == "google.cloud.orchestration.airflow.service.v1.Environments.DeleteEnvironment" {
					opTimeline = gcpcommon.MustGCPOperationTimeline(ctx, envTimeline, "DeleteEnvironment", tc.inputAudit.OperationID)
				}
			}

			tracker := gcpcommon.NewGCPOperationTracker()
			tc.setupTracker(tracker, envTimeline)

			l := testlog.NewMockLog(testTime, tc.inputAudit, tc.inputResource)

			cs, _, err := mapperSetting.ProcessLogByGroup(ctx, l, tracker)
			if err != nil {
				t.Fatalf("ProcessLogByGroup() error = %v", err)
			}

			if envTimeline.Parent == nil || envTimeline.Parent.Type.GetId() != gcpcommon.TimelineTypeGCPProject.GetId() {
				t.Errorf("expected envTimeline parent to be GCPProject timeline, got %v", envTimeline.Parent)
			}
			if diff := cmp.Diff(gcpcommon.TimelineTypeManagedAirflowEnvironment.GetId(), envTimeline.Type.GetId()); diff != "" {
				t.Errorf("expected envTimeline type to be ComposerEnvironment, diff (-want +got):\n%s", diff)
			}

			tc.assert(t, cs, envTimeline, opTimeline)
		})
	}
}

func TestComposerAuditLogIngester(t *testing.T) {
	testTime := time.Date(2026, time.August, 10, 0, 23, 12, 0, time.UTC)
	ingester := gcpcommon.NewGCPOperationLogIngester(
		composerapiaudit.ListLogEntriesTaskID.Ref(),
		composerapiaudit.LogTypeManagedAirflowAPI,
	)

	testCases := []struct {
		desc       string
		inputLog   *log.Log
		wantSumm   string
		wantStatus int
	}{
		{
			desc: "operation start log",
			inputLog: testlog.NewMockLog(
				testTime,
				inspectioncore.DefaultSeverityFieldSet{Severity: inspectioncore.SeverityInfo},
				gcpcommon.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
					OperationFirst: true,
					OperationLast:  false,
					Status:         -1,
				},
				composerapiaudit.ComposerAuditLogResourceFieldSet{
					EnvironmentName: "test-environment",
				},
			),
			wantSumm: "Start: google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
		},
		{
			desc: "operation succeeded log",
			inputLog: testlog.NewMockLog(
				testTime,
				inspectioncore.DefaultSeverityFieldSet{Severity: inspectioncore.SeverityInfo},
				gcpcommon.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
					OperationFirst: false,
					OperationLast:  true,
					Status:         -1,
				},
				composerapiaudit.ComposerAuditLogResourceFieldSet{
					EnvironmentName: "test-environment",
				},
			),
			wantSumm: "Succeeded: google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
		},
		{
			desc: "operation failed log",
			inputLog: testlog.NewMockLog(
				testTime,
				inspectioncore.DefaultSeverityFieldSet{Severity: inspectioncore.SeverityError},
				gcpcommon.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
					OperationFirst: false,
					OperationLast:  true,
					Status:         3,
					StatusMessage:  "INVALID_ARGUMENT: invalid location",
				},
				composerapiaudit.ComposerAuditLogResourceFieldSet{
					EnvironmentName: "test-environment",
				},
			),
			wantSumm: "Failed: [3: INVALID_ARGUMENT: invalid location] google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.inputLog)
			if err != nil {
				t.Fatalf("ProcessLog() error = %v", err)
			}
			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSumm).
				HasLogType(composerapiaudit.LogTypeManagedAirflowAPI)
		})
	}
}
