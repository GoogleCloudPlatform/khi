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

package googlecloudlogcomposerapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcomposerapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomposerapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
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
	testCommonFieldSet := &log.CommonFieldSet{
		Timestamp: testTime,
	}

	testCases := []struct {
		desc          string
		inputResource googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet
		inputAudit    googlecloudcommon_contract.GCPAuditLogFieldSet
		setupTracker  func(tracker *googlecloudcommon_contract.GCPOperationTracker, envPath *khifilev6.TimelinePath)
		assert        func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath)
	}{
		{
			desc: "CreateEnvironment operation started adds provisioning revision and operation start",
			inputResource: googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
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
			setupTracker: func(tracker *googlecloudcommon_contract.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:    commonlogk8saudit_contract.VerbCreate,
						StateType:   googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentProvisioning,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `name: projects/test-project/locations/us-central1/environments/test-environment
config:
  softwareConfig:
    imageVersion: composer-3-airflow-2`).Node,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
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
			inputResource: googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-create-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
				Status:         0,
			},
			setupTracker: func(tracker *googlecloudcommon_contract.GCPOperationTracker, envPath *khifilev6.TimelinePath) {
				tracker.MarkStarted("op-create-1")
				tracker.MarkResourceRevision(envPath)
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentExisting,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "CreateEnvironment operation finished without start log seen prepends LogNotFound revisions",
			inputResource: googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-create-missing-start",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
				Status:         0,
			},
			setupTracker: func(tracker *googlecloudcommon_contract.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentProvisioningLogNotFound,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentExisting,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStartedLogNotFound,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: time.Unix(0, 0),
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "DeleteEnvironment operation started adds deleting revision",
			inputResource: googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-delete-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.DeleteEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
			},
			setupTracker: func(tracker *googlecloudcommon_contract.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentExistingLogNotFound,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentDeleting,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "DeleteEnvironment operation finished with start seen adds deleted revision",
			inputResource: googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-delete-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.DeleteEnvironment",
				PrincipalEmail: "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
				Status:         0,
			},
			setupTracker: func(tracker *googlecloudcommon_contract.GCPOperationTracker, envPath *khifilev6.TimelinePath) {
				tracker.MarkStarted("op-delete-1")
				tracker.MarkResourceRevision(envPath)
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(envPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    googlecloudclustercomposer_contract.RevisionStateManagedAirflowEnvironmentDeleted,
						Principal:    "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime:  testTime,
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(opPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "serviceAccount:khi-sa@test-project.iam.gserviceaccount.com",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "immediate operation creates event on environment timeline",
			inputResource: googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
				EnvironmentName: "test-environment",
				Location:        "us-central1",
				ProjectID:       "test-project",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "",
				OperationFirst: false,
				OperationLast:  false,
				MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.GetEnvironment",
				PrincipalEmail: "user@example.com",
			},
			setupTracker: func(tracker *googlecloudcommon_contract.GCPOperationTracker, envPath *khifilev6.TimelinePath) {},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet, envPath *khifilev6.TimelinePath, opPath *khifilev6.TimelinePath) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(envPath)
			},
		},
	}

	mapperSetting := &composerAuditLogLogToTimelineMapperSetting{}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			builder := khifilev6.NewBuilder()
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, tc.inputResource.ProjectID)
			envTimeline := googlecloudcommon_contract.MustManagedAirflowEnvironmentTimeline(ctx, projectTimeline, tc.inputResource.EnvironmentName)

			var opTimeline *khifilev6.TimelinePath
			if !tc.inputAudit.ImmediateOperation() {
				opTimeline = googlecloudcommon_contract.MustGCPOperationTimeline(ctx, envTimeline, "CreateEnvironment", tc.inputAudit.OperationID)
				if tc.inputAudit.MethodName == "google.cloud.orchestration.airflow.service.v1.Environments.DeleteEnvironment" {
					opTimeline = googlecloudcommon_contract.MustGCPOperationTimeline(ctx, envTimeline, "DeleteEnvironment", tc.inputAudit.OperationID)
				}
			}

			tracker := googlecloudcommon_contract.NewGCPOperationTracker()
			tc.setupTracker(tracker, envTimeline)

			l := log.NewLogWithFieldSetsForTest(testCommonFieldSet, &tc.inputAudit, &tc.inputResource)

			cs, _, err := mapperSetting.ProcessLogByGroup(ctx, l, tracker)
			if err != nil {
				t.Fatalf("ProcessLogByGroup() error = %v", err)
			}

			if envTimeline.Parent == nil || envTimeline.Parent.Type.GetId() != googlecloudcommon_contract.TimelineTypeGCPProject.GetId() {
				t.Errorf("expected envTimeline parent to be GCPProject timeline, got %v", envTimeline.Parent)
			}
			if diff := cmp.Diff(googlecloudcommon_contract.TimelineTypeManagedAirflowEnvironment.GetId(), envTimeline.Type.GetId()); diff != "" {
				t.Errorf("expected envTimeline type to be ComposerEnvironment, diff (-want +got):\n%s", diff)
			}

			tc.assert(t, cs, envTimeline, opTimeline)
		})
	}
}

func TestComposerAuditLogIngester(t *testing.T) {
	testTime := time.Date(2026, time.August, 10, 0, 23, 12, 0, time.UTC)
	ingester := googlecloudcommon_contract.NewGCPOperationLogIngester(
		googlecloudlogcomposerapiaudit_contract.FieldSetReaderTaskID.Ref(),
		googlecloudlogcomposerapiaudit_contract.LogTypeManagedAirflowAPI,
	)

	testCases := []struct {
		desc       string
		inputLog   *log.Log
		wantSumm   string
		wantStatus int
	}{
		{
			desc: "operation start log",
			inputLog: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				&inspectioncore_contract.DefaultSeverityFieldSet{Severity: inspectioncore_contract.SeverityInfo},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
					OperationFirst: true,
					OperationLast:  false,
					Status:         -1,
				},
				&googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
					EnvironmentName: "test-environment",
				},
			),
			wantSumm: "Start: google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
		},
		{
			desc: "operation succeeded log",
			inputLog: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				&inspectioncore_contract.DefaultSeverityFieldSet{Severity: inspectioncore_contract.SeverityInfo},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
					OperationFirst: false,
					OperationLast:  true,
					Status:         -1,
				},
				&googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
					EnvironmentName: "test-environment",
				},
			),
			wantSumm: "Succeeded: google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
		},
		{
			desc: "operation failed log",
			inputLog: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				&inspectioncore_contract.DefaultSeverityFieldSet{Severity: inspectioncore_contract.SeverityError},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.orchestration.airflow.service.v1.Environments.CreateEnvironment",
					OperationFirst: false,
					OperationLast:  true,
					Status:         3,
					StatusMessage:  "INVALID_ARGUMENT: invalid location",
				},
				&googlecloudlogcomposerapiaudit_contract.ComposerAuditLogResourceFieldSet{
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
				HasLogType(googlecloudlogcomposerapiaudit_contract.LogTypeManagedAirflowAPI)
		})
	}
}
