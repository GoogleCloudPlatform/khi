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

package commonlogk8sauditv2_impl

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

type lifeTimeTrackerGroupState struct {
	// WasCompletelyRemoved is true if the resource was completely removed.
	WasCompletelyRemoved bool
	// DeletionStarted is true if the deletion started.
	DeletionStarted bool
	// PrevUID is the previous UID of the resource.
	PrevUID string
}

type lifeTimeTrackerTaskSetting struct {
	// kindsToWaitExactDeletionToDetermineDeletion is the map of kinds to wait exact deletion to determine deletion.
	kindsToWaitExactDeletionToDetermineDeletion map[string]struct{}
}

// isDeletiveVerb returns true if the verb is delete or deletecollection.
func isDeletiveVerb(verb enum.RevisionVerb) bool {
	return verb == enum.RevisionVerbDelete || verb == enum.RevisionVerbDeleteCollection
}

// isCreativeVerb returns true if the verb is create or update. These are possible to create the resource.
func isCreativeVerb(verb enum.RevisionVerb) bool {
	return verb == enum.RevisionVerbCreate || verb == enum.RevisionVerbUpdate
}

// isPod returns true if the operation is for a pod.
func isPod(op *model.KubernetesObjectOperation) bool {
	return op.APIVersion == "core/v1" && op.GetSingularKindName() == "pod"
}

// DetectLifetimeLogEvent detects if the log is the timing to create or delete the timeline resource and update the log field.
func (r *lifeTimeTrackerTaskSetting) DetectLifetimeLogEvent(ctx context.Context, l *commonlogk8sauditv2_contract.ResourceManifestLog, prevGroupData *lifeTimeTrackerGroupState) (*lifeTimeTrackerGroupState, error) {
	k8sFieldSet := log.MustGetFieldSet(l.Log, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	isFirst := false
	if prevGroupData == nil {
		prevGroupData = &lifeTimeTrackerGroupState{}
		isFirst = true
	}

	// Mark the resource being created when it is a first log, or when it's non-deletive log and the resource was completely removed.
	if isFirst ||
		k8sFieldSet.K8sOperation.Verb == enum.RevisionVerbCreate ||
		(isCreativeVerb(k8sFieldSet.K8sOperation.Verb) && prevGroupData.WasCompletelyRemoved) {
		l.ResourceCreated = true
	}
	if isDeletiveVerb(k8sFieldSet.K8sOperation.Verb) && prevGroupData.WasCompletelyRemoved {
		return prevGroupData, nil
	}

	if l.ResourceBodyReader == nil {
		if isDeletiveVerb(k8sFieldSet.K8sOperation.Verb) {
			prevGroupData.DeletionStarted = true
			l.ResourceDeleted = true
		}
	} else {
		deletionStarted := false
		underGracefulPeriod := false
		deletionCompleted := false
		uid, _ := GetUID(l.ResourceBodyReader)
		if uid != prevGroupData.PrevUID {
			prevGroupData.PrevUID = uid
			if !isDeletiveVerb(k8sFieldSet.K8sOperation.Verb) {
				l.ResourceCreated = true
			}
			prevGroupData.DeletionStarted = false
			prevGroupData.WasCompletelyRemoved = false
		} else {
			deletionStarted = prevGroupData.DeletionStarted
			deletionCompleted = prevGroupData.WasCompletelyRemoved
		}

		if isDeletiveVerb(k8sFieldSet.K8sOperation.Verb) {
			prevGroupData.DeletionStarted = true
			deletionStarted = true
			if isPod(k8sFieldSet.K8sOperation) {
				phase, _ := GetPodPhase(l.ResourceBodyReader)
				switch phase {
				case "Failed", "Succeeded":
					deletionCompleted = true
				default:
					underGracefulPeriod = true
				}
			}
		}
		deletionGracefulPeriods, found := GetDeletionGracePeriodSeconds(l.ResourceBodyReader)
		if found {
			if deletionGracefulPeriods > 0 {
				underGracefulPeriod = true
			}
			if deletionGracefulPeriods == 0 {
				deletionCompleted = true
			}
			deletionStarted = true
		}

		finalizers, found := GetFinalizers(l.ResourceBodyReader)
		if found && len(finalizers) > 0 && deletionStarted {
			deletionCompleted = false
			underGracefulPeriod = true
		}

		_, found = GetDeletionTimestamp(l.ResourceBodyReader)
		if found {
			deletionStarted = true
			if !underGracefulPeriod { // if the graceful period seconds wasn't found and become zero, then the resource is deleted.
				deletionCompleted = true
			}
		}

		switch {
		case deletionCompleted:
			l.ResourceDeleted = true
			prevGroupData.WasCompletelyRemoved = true
			prevGroupData.DeletionStarted = false
		case underGracefulPeriod:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
		case deletionStarted:
			// The exact deletion proof is not found for this case.
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = true
			apiVersionKind := fmt.Sprintf("%s#%s", k8sFieldSet.K8sOperation.APIVersion, k8sFieldSet.K8sOperation.GetSingularKindName())
			if _, found := r.kindsToWaitExactDeletionToDetermineDeletion[apiVersionKind]; !found {
				l.ResourceDeleted = true
			}
		default:
			prevGroupData.WasCompletelyRemoved = false
			prevGroupData.DeletionStarted = false
		}
	}
	return prevGroupData, nil
}

// ResourceLifetimeTrackerTask is the task to track the lifetime of resources.
var ResourceLifetimeTrackerTask = inspectiontaskbase.NewProgressReportableInspectionTask[commonlogk8sauditv2_contract.ResourceManifestLogGroupMap](
	commonlogk8sauditv2_contract.ResourceLifetimeTrackerTaskID,
	[]taskid.UntypedTaskReference{
		commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref(),
		commonlogk8sauditv2_contract.K8sAuditLogIngesterTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, tp *inspectionmetadata.TaskProgressMetadata) (commonlogk8sauditv2_contract.ResourceManifestLogGroupMap, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			slog.DebugContext(ctx, "Skipping task because this is dry run mode")
			return commonlogk8sauditv2_contract.ResourceManifestLogGroupMap{}, nil
		}

		groupedLogs := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref())

		totalLogCount := 0
		var processedLogCount atomic.Uint32
		for _, group := range groupedLogs {
			totalLogCount += len(group.Logs)
		}

		updator := progressutil.NewProgressUpdator(tp, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
			current := processedLogCount.Load()
			tp.Percentage = float32(current) / float32(totalLogCount)
			tp.Message = fmt.Sprintf("%d/%d", current, totalLogCount)
		})
		updator.Start(ctx)

		processedLogCount.Store(0)
		setting := &lifeTimeTrackerTaskSetting{
			kindsToWaitExactDeletionToDetermineDeletion: map[string]struct{}{
				"core/v1#pod": {},
			},
		}

		pool := worker.NewPool(runtime.GOMAXPROCS(0))
		for _, group := range groupedLogs {
			pool.Run(func() {
				var groupData *lifeTimeTrackerGroupState
				// Lifetimetracker doesn't handle namespace resources.
				if group.Resource.Type() == commonlogk8sauditv2_contract.Namespace {
					return
				}
				for _, l := range group.Logs {
					var err error
					groupData, err = setting.DetectLifetimeLogEvent(ctx, l, groupData)
					if err != nil {
						var yaml string
						yamlBytes, err2 := l.Log.Serialize("", &structured.YAMLNodeSerializer{})
						if err2 != nil {
							yaml = "ERROR!! failed to dump in yaml"
						} else {
							yaml = string(yamlBytes)
						}
						slog.WarnContext(ctx, "parser ended with an error", "error", err, "logContent", yaml)
						continue
					}
				}
				processedLogCount.Add(uint32(len(group.Logs)))
			})
		}
		pool.Wait()
		updator.Done()

		return groupedLogs, nil
	},
)
