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

package bindingrecorder

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("binding", []taskid.UntypedTaskReference{}, func(ctx context.Context, resourcePath string, l *types.AuditLogParserInput, prevState any, cs *history.ChangeSet, builder *history.Builder) (any, error) {
		return nil, recordChangeSetForLog(ctx, resourcePath, l, cs)
	}, recorder.SubresourceLogGroupFilter("binding"), recorder.AnyLogFilter())
	return nil
}

func recordChangeSetForLog(ctx context.Context, resourcePath string, l *types.AuditLogParserInput, cs *history.ChangeSet) error {
	if l.ResourceBodyReader == nil {
		return nil
	}
	commonField, _ := log.GetFieldSet(l.Log, &log.CommonFieldSet{})
	target := l.ResourceBodyReader.ReadStringOrDefault("target.name", "unknown")

	podScheduledStatusPath := resourcepath.Status(resourcepath.Pod(l.Operation.Namespace, l.Operation.Name), "PodScheduled")
	nodeBindingResourcePath := resourcepath.NodeBinding(target, l.Operation.Namespace, l.Operation.Name)
	if l.Operation.Verb == enum.RevisionVerbCreate {
		cs.RecordRevision(nodeBindingResourcePath, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbCreate,
			Body:       l.ResourceBodyYaml,
			Partial:    false,
			Requestor:  l.Requestor,
			ChangeTime: commonField.Timestamp,
			State:      enum.RevisionStateExisting,
		})
		cs.RecordRevision(podScheduledStatusPath, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbStatusTrue,
			Body:       "# PodScheduled status was inferred to be `True` from a binding resource",
			Partial:    false,
			Requestor:  "",
			ChangeTime: commonField.Timestamp,
			State:      enum.RevisionStateConditionTrue,
		})
	} else {
		cs.RecordRevision(nodeBindingResourcePath, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbDelete,
			Body:       l.ResourceBodyYaml,
			Partial:    false,
			Requestor:  l.Requestor,
			ChangeTime: commonField.Timestamp,
			State:      enum.RevisionStateDeleted,
		})
		cs.RecordRevision(podScheduledStatusPath, &history.StagingResourceRevision{
			Verb:       enum.RevisionVerbStatusFalse,
			Body:       "# PodScheduled status was inferred to be `False` from a binding resource",
			Partial:    false,
			Requestor:  "",
			ChangeTime: commonField.Timestamp,
			State:      enum.RevisionStateConditionFalse,
		})
	}
	return nil
}
