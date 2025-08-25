package ossclusterk8s_impl

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

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

type OSSK8sEventFromK8sAudit struct {
}

// Dependencies implements parsertask.Parser.
func (o *OSSK8sEventFromK8sAudit) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parsertask.Parser.
func (o *OSSK8sEventFromK8sAudit) Description() string {
	return `The event log parser for OSS kubernetes from the audit log`
}

// GetParserName implements parsertask.Parser.
func (o *OSSK8sEventFromK8sAudit) GetParserName() string {
	return "OSS Kubernetes Event logs from JSONL audit log"
}

// Grouper implements parsertask.Parser.
func (o *OSSK8sEventFromK8sAudit) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// LogTask implements parsertask.Parser.
func (o *OSSK8sEventFromK8sAudit) LogTask() taskid.TaskReference[[]*log.Log] {
	return ossclusterk8s_contract.EventAuditLogFilterTaskID.Ref()
}

// Parse implements parsertask.Parser.
func (o *OSSK8sEventFromK8sAudit) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	apiVersion := l.ReadStringOrDefault("responseObject.involvedObject.apiVersion", "core/v1")
	kind := strings.ToLower(l.ReadStringOrDefault("responseObject.involvedObject.kind", "unknown"))
	namespace := l.ReadStringOrDefault("responseObject.involvedObject.namespace", "cluster-scope")
	name := l.ReadStringOrDefault("responseObject.involvedObject.name", "unknown")
	subresource := l.ReadStringOrDefault("responseObject.involvedObject.subresource", "")

	if subresource == "" {
		cs.RecordEvent(resourcepath.NameLayerGeneralItem(apiVersion, kind, namespace, name))
	} else {
		cs.RecordEvent(resourcepath.SubresourceLayerGeneralItem(apiVersion, kind, namespace, name, subresource))
	}

	reason := l.ReadStringOrDefault("responseObject.reason", "???")
	message := l.ReadStringOrDefault("responseObject.message", "")
	cs.RecordLogSummary(fmt.Sprintf("【%s】%s", reason, message))
	return nil
}

// TargetLogType implements parsertask.Parser.
func (o *OSSK8sEventFromK8sAudit) TargetLogType() enum.LogType {
	return enum.LogTypeAudit
}

var _ legacyparser.Parser = (*OSSK8sEventFromK8sAudit)(nil)

var OSSK8sEventLogParserTask = legacyparser.NewParserTaskFromParser(
	ossclusterk8s_contract.OSSK8sEventLogParserTaskID,
	&OSSK8sEventFromK8sAudit{}, 2000, true, []string{
		ossclusterk8s_contract.InspectionTypeID,
	},
)
