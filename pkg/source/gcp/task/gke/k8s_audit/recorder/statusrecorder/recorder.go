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

package statusrecorder

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/manifestutil"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"

	goyaml "gopkg.in/yaml.v3"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("resource-status", []taskid.UntypedTaskReference{}, func(ctx context.Context, resourcePath string, l *types.ResourceSpecificParserInput, prevState any, cs *history.ChangeSet, builder *history.Builder) (any, error) {
		var prevResourceStatus *model.K8sResourceContainingStatus
		if prevState != nil {
			prevResourceStatus = prevState.(*model.K8sResourceContainingStatus)
		}
		return recordChangeSetForLog(ctx, resourcePath, l, prevResourceStatus, cs, builder)
	}, recorder.AnyLogGroupFilter(), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}

func recordChangeSetForLog(ctx context.Context, resourcePath string, log *types.ResourceSpecificParserInput, prevStatus *model.K8sResourceContainingStatus, cs *history.ChangeSet, builder *history.Builder) (*model.K8sResourceContainingStatus, error) {
	var resourceContainingStatus model.K8sResourceContainingStatus
	err := log.ResourceBodyReader.ReadReflect("", &resourceContainingStatus)
	if err != nil {
		return prevStatus, err
	}
	if resourceContainingStatus.Status == nil || len(resourceContainingStatus.Status.Conditions) == 0 {
		// This resource has no status field or no conditions in status field
		return &resourceContainingStatus, nil
	}

	deletionStatus := manifestutil.ParseDeletionStatus(ctx, log.ResourceBodyReader, log.Operation)
	isDeletionRequest := deletionStatus == manifestutil.DeletionStatusDeleted
	for _, condition := range resourceContainingStatus.Status.Conditions {
		lastTransitionTime, err := time.Parse(time.RFC3339, condition.LastTransitionTime)
		if err != nil {
			continue
		}
		conditionTime := lastTransitionTime
		lastHeartbeatTime, err := time.Parse(time.RFC3339, condition.LastHeartbeatTime)
		if err == nil && lastHeartbeatTime.Sub(conditionTime) > 0 {
			conditionTime = lastHeartbeatTime
		}
		lastProbeTime, err := time.Parse(time.RFC3339, condition.LastProbeTime)
		if err == nil && lastProbeTime.Sub(lastHeartbeatTime) > 0 {
			conditionTime = lastProbeTime
		}
		// Ignore if the transition time was older than the last revision
		statusPath := resourcepath.Status(resourcepath.FromK8sOperation(*log.Operation), condition.Type)
		if log.Operation.SubResourceName != "" {
			parentOp := model.KubernetesObjectOperation{
				APIVersion: log.Operation.APIVersion,
				PluralKind: log.Operation.PluralKind,
				Namespace:  log.Operation.Namespace,
				Name:       log.Operation.Name,
				Verb:       log.Operation.Verb,
			}
			statusPath = resourcepath.Status(resourcepath.FromK8sOperation(parentOp), condition.Type)
		}
		tb := builder.GetTimelineBuilder(statusPath.Path)
		latest := tb.GetLatestRevision()
		latestTime := time.Time{}
		if latest != nil {
			latestTime = latest.ChangeTime
		} else {
			creationTime := manifestutil.ParseCreationTime(log.ResourceBodyReader, time.Time{})

			if err == nil && conditionTime.Sub(creationTime) != 0 {
				cs.RecordRevision(statusPath, &history.StagingResourceRevision{
					Verb:       enum.RevisionVerbStatusUnknown,
					Body:       "# status is unknown but existence is inferred from the later log.",
					Partial:    false,
					Inferred:   true,
					Requestor:  "",
					ChangeTime: creationTime,
					State:      enum.RevisionStateConditionUnknown,
				})
			}
		}
		prevCondition := lookUpConditionFromStatus(prevStatus, condition.Type)
		latestUpdateTimeIsLaterThanPrevRevision := conditionTime.Sub(latestTime) > 0 // The previous revision is older than the timestamps written on the condition. This should be recorded as the change even if the reason and state is same to show the heart beat timing.
		conditionUpdated := prevCondition != nil && (prevCondition.Status != condition.Status || prevCondition.Message != condition.Message || prevCondition.Reason != condition.Reason)
		if latestUpdateTimeIsLaterThanPrevRevision || conditionUpdated {
			conditionYaml, err := goyaml.Marshal(condition)
			if err != nil {
				continue
			}
			cs.RecordRevision(statusPath, &history.StagingResourceRevision{
				Verb:       conditionStateToRevisionVerb(condition.Status),
				Body:       string(conditionYaml),
				Partial:    false,
				Requestor:  "",
				ChangeTime: conditionTime,
				State:      conditionStateToRevisionState(condition.Status),
			})
		}
		if isDeletionRequest {
			cs.RecordRevision(statusPath, &history.StagingResourceRevision{
				Verb:       enum.RevisionVerbDelete,
				Body:       "",
				Partial:    false,
				Requestor:  log.PrincipalEmail,
				ChangeTime: log.Log.Timestamp(),
				State:      enum.RevisionStateDeleted,
			})
		}
	}
	if isDeletionRequest {
		return nil, nil
	}
	return &resourceContainingStatus, nil
}

func lookUpConditionFromStatus(status *model.K8sResourceContainingStatus, typeStr string) *model.K8sResourceStatusCondition {
	if status == nil || status.Status == nil || status.Status.Conditions == nil {
		return nil
	}
	for _, condition := range status.Status.Conditions {
		if condition.Type == typeStr {
			return condition
		}
	}
	return nil
}

func conditionStateToRevisionVerb(conditionState string) enum.RevisionVerb {
	if conditionState == "True" {
		return enum.RevisionVerbStatusTrue
	} else if conditionState == "False" {
		return enum.RevisionVerbStatusFalse
	}
	return enum.RevisionVerbStatusUnknown
}

func conditionStateToRevisionState(conditionState string) enum.RevisionState {
	if conditionState == "True" {
		return enum.RevisionStateConditionTrue
	} else if conditionState == "False" {
		return enum.RevisionStateConditionFalse
	}
	return enum.RevisionStateConditionUnknown
}
