// Copyright 2025 Google LLC
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

package googlecloudlogk8saudit_impl_test

import (
	"context"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	googlecloudclustergke_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergke/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/taskrecord"
)

func setupAuditLogInspectionServer(t testing.TB) *coreinspection.InspectionTaskServer {
	t.Helper()
	logger.InitGlobalKHILogger()
	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create ioConfig: %v", err)
	}
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := generated.RegisterAllInspectionTasks(server); err != nil {
		t.Fatalf("failed to register all inspection tasks: %v", err)
	}
	return server
}

func getAuditLogJobTestConfig() *taskrecord.JobTestConfig {
	return &taskrecord.JobTestConfig{
		InspectionType: googlecloudclustergke_contract.InspectionTypeID,
		InspectionFeatures: []string{
			"khi.google.com/k8s-common-auditlog/k8s-auditlog-parser-tail#gcp",
		},
		InspectionValues: map[string]any{
			"cloud.google.com/common/input-duration":   "4h",
			"cloud.google.com/common/input-end-time":   "2026-02-18T09:53:08Z",
			"cloud.google.com/common/input-location":   "us-central1-a",
			"cloud.google.com/common/input-project-id": "khi-testing-with-auditlog",
			"cloud.google.com/common/input-query-resource-names/cloud.google.com/log/k8s-audit/audit-list-log-entries": "projects/khi-testing-with-auditlog",
			"cloud.google.com/k8s/input-cluster-name": "p0-gke-basic-1",
			"cloud.google.com/k8s/input-kinds": []any{
				"@default",
			},
			"cloud.google.com/k8s/input-namespaces": []any{
				"@all_cluster_scoped",
				"@all_namespaced",
			},
			"timezoneShift":      "9",
			"timezoneShiftHours": 9,
		},
		RecordedTasks: []taskid.UntypedTaskReference{
			googlecloudlogk8saudit_contract.GCPK8sAuditLogListLogEntriesTaskID.Ref(),
		},
		TargetTask: googlecloudlogk8saudit_contract.GCPK8sAuditLogCommonFieldSetReaderTaskID.Ref(),
	}
}

func BenchmarkGCPK8sAuditLogCommonFieldSetReaderTask(b *testing.B) {
	server := setupAuditLogInspectionServer(b)
	cfg := getAuditLogJobTestConfig()

	harness := taskrecord.NewJobTestHarness(b, server, cfg)

	if harness.IsRecordMode() {
		if _, err := harness.Run(context.Background()); err != nil {
			b.Fatalf("failed to record fixture: %v", err)
		}
		return
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, err := harness.Run(context.Background())
		if err != nil {
			b.Fatalf("failed to replay target task: %v", err)
		}
	}
}
