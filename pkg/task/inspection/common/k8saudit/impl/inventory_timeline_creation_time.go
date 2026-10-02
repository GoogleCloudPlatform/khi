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
	"slices"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// TimelineCreationTimeInventoryTask aggregates creation timestamps per timeline path discovered from audit logs.
var TimelineCreationTimeInventoryTask = inspectiontaskbase.NewInventoryTask(
	k8saudit.TimelineCreationTimeInventoryTaskID,
	k8saudit.TagTimelineCreationTimeDiscovery,
	mergeTimelineCreationTimes,
)

func mergeTimelineCreationTimes(results []k8saudit.TimelineCreationTimes) (k8saudit.TimelineCreationTimes, error) {
	result := k8saudit.TimelineCreationTimes{}
	for _, r := range results {
		for path, times := range r {
			result[path] = append(result[path], times...)
		}
	}
	for path, times := range result {
		result[path] = deduplicateAndSortTimes(times)
	}
	return result, nil
}

func deduplicateAndSortTimes(times []time.Time) []time.Time {
	if len(times) <= 1 {
		return slices.Clone(times)
	}
	sorted := slices.Clone(times)
	slices.SortFunc(sorted, func(a, b time.Time) int {
		return a.Compare(b)
	})
	return slices.CompactFunc(sorted, func(a, b time.Time) bool {
		return a.Equal(b)
	})
}

func extractLogCreationTime(l *k8saudit.ResourceManifestLog, verb *pb.Verb) (time.Time, bool) {
	if l.ResourceBodyReader != nil {
		if creationTime, found := GetCreationTimestamp(l.ResourceBodyReader); found {
			return creationTime, true
		}
	}
	if verb == k8saudit.VerbCreate {
		return l.Log.Timestamp, true
	}
	return time.Time{}, false
}

// ResourceTimelineCreationTimeDiscoveryTask extracts resource timeline creation timestamps from audit logs.
var ResourceTimelineCreationTimeDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	k8saudit.ResourceTimelineCreationTimeDiscoveryTaskID,
	[]coretask.Dependency{
		k8saudit.ManifestGeneratorTaskID.Ref(),
		k8saudit.K8sAuditLogExtractorRef.Ref(coretask.FromActiveGraph),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8saudit.TimelineCreationTimes, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return k8saudit.TimelineCreationTimes{}, nil
		}
		result := k8saudit.TimelineCreationTimes{}
		resourceLogs := coretask.GetTaskResult(ctx, k8saudit.ManifestGeneratorTaskID.Ref())
		tracker := progress.NewTracker(ctx, len(resourceLogs), progress.WithUnit("groups"))
		defer tracker.Done()
		for _, group := range resourceLogs {
			if group.Resource.Type() == k8saudit.Namespace {
				tracker.Inc()
				continue
			}
			for _, l := range group.Logs {
				k8sFieldSet, err := k8saudit.ExtractK8sAuditLog(ctx, l.Log.NodeReader)
				if err != nil || k8sFieldSet == nil || k8sFieldSet.IsDryRun {
					continue
				}
				creationTime, hasCreationTime := extractLogCreationTime(l, k8sFieldSet.Verb)
				if !hasCreationTime {
					continue
				}
				targetPath := MustResolveTimelinePath(ctx, k8sFieldSet.ClusterName, group.Resource)
				result[targetPath] = append(result[targetPath], creationTime)
			}
			tracker.Inc()
		}
		for path, times := range result {
			result[path] = deduplicateAndSortTimes(times)
		}
		return result, nil
	},
	coretask.ProvidesTag(k8saudit.TagTimelineCreationTimeDiscovery),
	coretask.WithFeatureGate(k8saudit.K8sAuditLogParserTailRef),
	progress.WithTitle("Discover resource timeline creation times"),
	coretask.WithTaskDescription("Extracts resource timeline creation timestamps from Kubernetes audit logs."),
)

// PodPhaseTimelineCreationTimeDiscoveryTask extracts Pod phase timeline creation timestamps from audit logs.
var PodPhaseTimelineCreationTimeDiscoveryTask = inspectiontaskbase.NewInspectionTask(
	k8saudit.PodPhaseTimelineCreationTimeDiscoveryTaskID,
	[]coretask.Dependency{
		k8saudit.ManifestGeneratorTaskID.Ref(),
		k8saudit.K8sAuditLogExtractorRef.Ref(coretask.FromActiveGraph),
	},
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (k8saudit.TimelineCreationTimes, error) {
		if taskMode == inspectioncore.TaskModeDryRun {
			return k8saudit.TimelineCreationTimes{}, nil
		}
		result := k8saudit.TimelineCreationTimes{}
		resourceLogs := coretask.GetTaskResult(ctx, k8saudit.ManifestGeneratorTaskID.Ref())
		tracker := progress.NewTracker(ctx, len(resourceLogs), progress.WithUnit("groups"))
		defer tracker.Done()
		for _, group := range resourceLogs {
			if group.Resource.Type() != k8saudit.Resource || group.Resource.APIVersion != "core/v1" || group.Resource.Kind != "pod" {
				tracker.Inc()
				continue
			}
			for _, l := range group.Logs {
				if l.ResourceBodyReader == nil {
					continue
				}
				k8sFieldSet, err := k8saudit.ExtractK8sAuditLog(ctx, l.Log.NodeReader)
				if err != nil || k8sFieldSet == nil || k8sFieldSet.IsDryRun {
					continue
				}
				creationTime, hasCreationTime := extractLogCreationTime(l, k8sFieldSet.Verb)
				if !hasCreationTime {
					continue
				}
				uid, foundUID := GetUID(l.ResourceBodyReader)
				nodeName, foundNode := GetNodeNameOfPod(l.ResourceBodyReader)
				if foundUID && uid != "" && foundNode && nodeName != "" {
					podPhasePath := MustPodPhaseTimelinePath(ctx, k8sFieldSet.ClusterName, nodeName, group.Resource.Namespace, group.Resource.Name, uid)
					result[podPhasePath] = append(result[podPhasePath], creationTime)
				}
			}
			tracker.Inc()
		}
		for path, times := range result {
			result[path] = deduplicateAndSortTimes(times)
		}
		return result, nil
	},
	coretask.ProvidesTag(k8saudit.TagTimelineCreationTimeDiscovery),
	coretask.WithFeatureGate(k8saudit.K8sAuditLogParserTailRef),
	progress.WithTitle("Discover Pod phase timeline creation times"),
	coretask.WithTaskDescription("Extracts Pod phase timeline creation timestamps from Kubernetes audit logs."),
)
