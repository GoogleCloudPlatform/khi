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

package commonlogk8saudit_impl

import (
	"context"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// K8sAuditLogIngesterTask is the V2 task to serialize and ingest k8s audit logs.
var K8sAuditLogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	commonlogk8saudit_contract.K8sAuditLogIngesterTaskID,
	&k8sAuditLogIngesterV2{},
)

type k8sAuditLogIngesterV2 struct{}

// RawLogTask implements inspectiontaskbase.LogIngesterV2.
func (i *k8sAuditLogIngesterV2) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return commonlogk8saudit_contract.K8sAuditLogProviderRef
}

// Dependencies implements inspectiontaskbase.LogIngesterV2.
func (i *k8sAuditLogIngesterV2) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and populates the LogChangeSet.
func (i *k8sAuditLogIngesterV2) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	commonFs, err := log.GetFieldSet(l, &log.CommonFieldSet{})
	if err != nil {
		return nil, err
	}
	cs.SetTimestamp(commonFs.Timestamp)
	cs.SetLogType(commonlogk8saudit_contract.LogTypeAudit)

	k8sFieldSet, err := log.GetFieldSet(l, &commonlogk8saudit_contract.K8sAuditLogFieldSet{})
	if err != nil {
		return nil, err
	}

	if k8sFieldSet.IsError {
		cs.SetSeverity(inspectioncore_contract.SeverityError)
		cs.SetSummary(fmt.Sprintf("【%s(%d)】%s %s", k8sFieldSet.StatusMessage, k8sFieldSet.StatusCode, k8sFieldSet.VerbString(), k8sFieldSet.RequestURI))
	} else {
		cs.SetSeverity(inspectioncore_contract.SeverityInfo)
		cs.SetSummary(fmt.Sprintf("%s %s", k8sFieldSet.VerbString(), k8sFieldSet.RequestURI))
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*k8sAuditLogIngesterV2)(nil)
