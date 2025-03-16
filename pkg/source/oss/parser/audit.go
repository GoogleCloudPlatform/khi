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

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss/constant"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type OSSK8sAudit struct {
}

// Dependencies implements parser.Parser.
func (o *OSSK8sAudit) Dependencies() []string {
	return []string{}
}

// Description implements parser.Parser.
func (o *OSSK8sAudit) Description() string {
	return `The audit log parser for OSS kubernetes from the JSONL kube-apiserver audit log`
}

// GetParserName implements parser.Parser.
func (o *OSSK8sAudit) GetParserName() string {
	return "OSS Kubernetes Audit logs from JSONL audit log"
}

// Grouper implements parser.Parser.
func (o *OSSK8sAudit) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// LogTask implements parser.Parser.
func (o *OSSK8sAudit) LogTask() string {
	return OSSAuditLogNonEventFilterTaskID
}

// Parse implements parser.Parser.
func (o *OSSK8sAudit) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, variables *task.VariableSet) error {
	apiGroup := l.Fields.ReadStringOrDefault("objectRef.apiGroup", "core")
	apiVersion := l.Fields.ReadStringOrDefault("objectRef.apiVersion", "unknown")
	kind := l.Fields.ReadStringOrDefault("objectRef.resource", "unknown")
	namespace := l.Fields.ReadStringOrDefault("objectRef.namespace", "cluster-scope")
	name := l.Fields.ReadStringOrDefault("objectRef.name", "unknown")
	subresource := l.Fields.ReadStringOrDefault("objectRef.subresource", "")
	verb := l.Fields.ReadStringOrDefault("verb", "")

	if subresource == "status" {
		subresource = "" // status subresource response should contain the full body data of its parent
	}

	k8sOp := model.KubernetesObjectOperation{
		APIVersion:      fmt.Sprintf("%s/%s", apiGroup, apiVersion),
		PluralKind:      kind,
		Namespace:       namespace,
		Name:            name,
		SubResourceName: subresource,
		Verb:            verbStringToEnum(verb),
	}

	path := resourcepath.ResourcePath{
		Path:               k8sOp.CovertToResourcePath(),
		ParentRelationship: enum.RelationshipChild,
	}

	body, err := l.Fields.ToYaml("responseObject")
	if err != nil {
		body = "# failed to parse"
	}

	requestor := l.Fields.ReadStringOrDefault("user.username", "unknown")
	status := enum.RevisionStateExisting
	if verb == "delete" {
		status = enum.RevisionStateDeleted
	}
	deletionGracefulSeconds := l.Fields.ReadIntOrDefault("responseObject.metadata.deletionGracePeriodSeconds", -1)
	if deletionGracefulSeconds == 0 {
		status = enum.RevisionStateDeleted
	} else if deletionGracefulSeconds > 0 {
		status = enum.RevisionStateDeleting
	}

	cs.RecordRevision(path, &history.StagingResourceRevision{
		Verb:       k8sOp.Verb,
		Body:       body,
		Requestor:  requestor,
		ChangeTime: l.Timestamp(),
		State:      status,
	})

	requestURI := l.Fields.ReadStringOrDefault("requestURI", "")
	cs.RecordLogSummary(fmt.Sprintf("%s %s", verb, requestURI))
	return nil
}

func verbStringToEnum(verbStr string) enum.RevisionVerb {
	switch verbStr {
	case "create":
		return enum.RevisionVerbCreate
	case "update":
		return enum.RevisionVerbUpdate
	case "patch":
		return enum.RevisionVerbPatch
	case "delete":
		return enum.RevisionVerbDelete
	case "deletecollection":
		return enum.RevisionVerbDelete
	default:
		// Add verbs for get/list/watch
		return enum.RevisionVerbUpdate
	}
}

// TargetLogType implements parser.Parser.
func (o *OSSK8sAudit) TargetLogType() enum.LogType {
	return enum.LogTypeAudit
}

var _ parser.Parser = (*OSSK8sAudit)(nil)

var OSSK8sAuditParserTask = parser.NewParserTaskFromParser(
	constant.OSSTaskPrefix+"audit-parser",
	&OSSK8sAudit{}, true, []string{
		constant.OSSInspectionTypeID,
	},
)
