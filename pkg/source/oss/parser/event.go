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

package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss/constant"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type OSSK8sEventFromK8sAudit struct {
}

// Dependencies implements parser.Parser.
func (o *OSSK8sEventFromK8sAudit) Dependencies() []string {
	return []string{}
}

// Description implements parser.Parser.
func (o *OSSK8sEventFromK8sAudit) Description() string {
	return `The event log parser for OSS kubernetes from the audit log`
}

// GetParserName implements parser.Parser.
func (o *OSSK8sEventFromK8sAudit) GetParserName() string {
	return "OSS Kubernetes Event logs from JSONL audit log"
}

// Grouper implements parser.Parser.
func (o *OSSK8sEventFromK8sAudit) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// LogTask implements parser.Parser.
func (o *OSSK8sEventFromK8sAudit) LogTask() string {
	return OSSAuditLogEventLogsFilterTaskID
}

// Parse implements parser.Parser.
func (o *OSSK8sEventFromK8sAudit) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, variables *task.VariableSet) error {
	apiVersion := l.Fields.ReadStringOrDefault("responseObject.involvedObject.apiVersion", "core/v1")
	kind := strings.ToLower(l.Fields.ReadStringOrDefault("responseObject.involvedObject.kind", "unknown"))
	namespace := l.Fields.ReadStringOrDefault("responseObject.involvedObject.namespace", "cluster-scope")
	name := l.Fields.ReadStringOrDefault("responseObject.involvedObject.name", "unknown")
	subresource := l.Fields.ReadStringOrDefault("responseObject.involvedObject.subresource", "")

	if subresource == "" {
		cs.RecordEvent(resourcepath.NameLayerGeneralItem(apiVersion, kind, namespace, name))
	} else {
		cs.RecordEvent(resourcepath.SubresourceLayerGeneralItem(apiVersion, kind, namespace, name, subresource))
	}

	reason := l.Fields.ReadStringOrDefault("responseObject.reason", "???")
	message := l.Fields.ReadStringOrDefault("responseObject.message", "")
	cs.RecordLogSummary(fmt.Sprintf("【%s】%s", reason, message))
	return nil
}

// TargetLogType implements parser.Parser.
func (o *OSSK8sEventFromK8sAudit) TargetLogType() enum.LogType {
	return enum.LogTypeAudit
}

var _ parser.Parser = (*OSSK8sEventFromK8sAudit)(nil)

var OSSK8sEventLogParserTask = parser.NewParserTaskFromParser(
	constant.OSSTaskPrefix+"event-parser",
	&OSSK8sEventFromK8sAudit{}, true, []string{
		constant.OSSInspectionTypeID,
	},
)
