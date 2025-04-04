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

package ownerreferencerecorder

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("owner-references", []taskid.UntypedTaskReference{}, func(ctx context.Context, resourcePath string, currentLog *types.AuditLogParserInput, prevStateInGroup any, cs *history.ChangeSet, builder *history.Builder) (any, error) {
		return nil, recordChangeSetForLog(ctx, resourcePath, currentLog, cs, builder)
	}, recorder.AnyLogGroupFilter(), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}

func recordChangeSetForLog(ctx context.Context, resourcePath string, log *types.AuditLogParserInput, cs *history.ChangeSet, builder *history.Builder) error {
	if !log.ResourceBodyReader.Has("metadata.ownerReferences") {
		return nil
	}
	ownerReferencesReaders, err := log.ResourceBodyReader.Reader("metadata.ownerReferences[]")
	if err != nil {
		return nil
	}
	for _, referenceReader := range ownerReferencesReaders {
		kind, err := referenceReader.ReadString("kind")
		if err != nil {
			continue
		}
		apiVersion, err := referenceReader.ReadString("apiVersion")
		if err != nil {
			continue
		}
		name, err := referenceReader.ReadString("name")
		if err != nil {
			continue
		}
		if !strings.Contains(apiVersion, "/") {
			apiVersion = "core/" + apiVersion
		}
		namespace := log.Operation.Namespace
		// TODO: Usually ownerReference don't contain the namespace field but the owner should be in the same namespace.
		// But node is a cluster scopd resource. There should be better implementation rather than hard coding this rule here.
		if kind == "Node" {
			namespace = "cluster-scope"
		}

		ownedResource := resourcepath.FromK8sOperation(*log.Operation)
		ownerResource := resourcepath.NameLayerGeneralItem(apiVersion, strings.ToLower(kind), namespace, name)
		ownerSubresource := resourcepath.OwnerSubresource(ownerResource, log.Operation.Name, log.Operation.GetSingularKindName())
		cs.RecordResourceAlias(ownedResource, ownerSubresource)
	}
	return nil
}
