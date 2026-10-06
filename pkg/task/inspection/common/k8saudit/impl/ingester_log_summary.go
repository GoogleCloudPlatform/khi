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

package k8saudit_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// k8sAuditLogIngesterTask is the task to serialize and ingest k8s audit logs.
var k8sAuditLogIngesterTask = inspectiontaskbase.DefineLogIngesterTask(
	k8saudit.K8sAuditLogIngesterTaskID,
	k8saudit.K8sAuditLogProviderRef,
	func(b *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
		extractorInput := coretask.Use(b, k8saudit.K8sAuditLogExtractorRef)
		return func(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
			extractor := extractorInput.Get(ctx)
			return processK8sAuditLog(l, extractor)
		}
	},
)

// processK8sAuditLog parses a raw audit log entry and populates its LogChangeSet.
func processK8sAuditLog(l *log.Log, extractor k8saudit.K8sAuditLogExtractor) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetTimestamp(l.Timestamp)
	cs.SetLogType(k8saudit.LogTypeAudit)

	k8sFieldSet, err := k8saudit.ExtractK8sAuditLog(l.NodeReader, extractor)
	if err != nil {
		return nil, err
	}

	var summary string
	if k8sFieldSet.IsError {
		cs.SetSeverity(inspectioncore.SeverityError)
		summary = fmt.Sprintf("【%s(%d)】%s %s", k8sFieldSet.StatusMessage, k8sFieldSet.StatusCode, k8sFieldSet.VerbString(), k8sFieldSet.RequestURI)
	} else {
		cs.SetSeverity(inspectioncore.SeverityInfo)
		summary = fmt.Sprintf("%s %s", k8sFieldSet.VerbString(), k8sFieldSet.RequestURI)
	}
	if k8sFieldSet.IsDryRun {
		summary = "【DryRun】" + summary
	}
	cs.SetSummary(summary)

	return cs, nil
}
