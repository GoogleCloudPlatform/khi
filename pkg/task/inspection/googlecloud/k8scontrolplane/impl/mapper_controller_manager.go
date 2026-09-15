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

package k8scontrolplane_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontrolplane"
)

func kindToKLogFieldPair(apiVersion string, kind string, klogField string, isNamespaced bool) *k8scontrolplane.KindToKLogFieldPairData {
	return &k8scontrolplane.KindToKLogFieldPairData{
		APIVersion:   apiVersion,
		KindName:     kind,
		KLogField:    klogField,
		IsNamespaced: isNamespaced,
	}
}

var defaultControllerManagerExtractor = &k8scontrolplane.K8sControllerManagerComponentExtractor{
	WellKnownSourceLocationToControllerMap: map[string]string{
		"namespace_controller.go":      "namespace-controller",
		"resource_quota_controller.go": "resourcequota-controller",
		"requestheader_controller.go":  "requestheader-controller",
		"pv_protection_controller.go":  "persistentvolume-protection-controller",
	},
	WellKnownKindToKLogFieldPairs: []*k8scontrolplane.KindToKLogFieldPairData{
		kindToKLogFieldPair("apps/v1", "deployment", "deployment", true),
		kindToKLogFieldPair("apps/v1", "replicaset", "replicaSet", true),
		kindToKLogFieldPair("apps/v1", "statefulset", "statefulSet", true),
		kindToKLogFieldPair("apps/v1", "daemonset", "daemonSet", true),
		kindToKLogFieldPair("batch/v1", "cronjob", "cronjob", true),
		kindToKLogFieldPair("batch/v1", "job", "job", true),
		kindToKLogFieldPair("policy/v1", "poddisruptionbudget", "podDisruptionBudget", true),
		kindToKLogFieldPair("certificates.k8s.io/v1", "certificatesigningrequest", "csr", false),
		kindToKLogFieldPair("core/v1", "persistentvolumeclaim", "PVC", true),
		kindToKLogFieldPair("core/v1", "persistentvolume", "volumeName", false),
		kindToKLogFieldPair("core/v1", "service", "service", true),
		kindToKLogFieldPair("core/v1", "node", "node", false),
		kindToKLogFieldPair("core/v1", "pod", "pod", true),
		kindToKLogFieldPair("core/v1", "namespace", "namespace", false),
	},
}

var ControllerManagerFilterTask = inspectiontaskbase.NewLogFilterTask(
	k8scontrolplane.ControllerManagerLogFilterTaskID,
	k8scontrolplane.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		parserType, err := k8scontrolplane.ExtractK8sControlplaneComponentParserType(l.NodeReader)
		if err != nil {
			return false
		}
		return parserType == k8scontrolplane.ComponentParserTypeControllerManager
	},
)

var ControllerManagerGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	k8scontrolplane.ControllerManagerLogGrouperTaskID,
	k8scontrolplane.ControllerManagerLogFilterTaskID.Ref(),
	func(ctx context.Context, log *log.Log) string {
		return "" // No grouping needed
	},
)

var ControllerManagerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](k8scontrolplane.ControllerManagerLogToTimelineMapperTaskID, &ControllerManagerTimelineMapper{})

// ControllerManagerTimelineMapper maps controller manager logs to timeline paths.
type ControllerManagerTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	uidPrefixTokenCandidates []rune
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (o *ControllerManagerTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8saudit.ResourceUIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *ControllerManagerTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return k8scontrolplane.ControllerManagerLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (o *ControllerManagerTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return k8scontrolplane.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (o *ControllerManagerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	finder := coretask.GetTaskResult(ctx, k8saudit.ResourceUIDPatternFinderTaskID.Ref())
	componentFieldSet, err := k8scontrolplane.ExtractK8sControlplaneComponent(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}
	commonMainMessage, err := k8scontrolplane.ExtractK8sControlplaneCommonMessage(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}
	controllerManagerFieldSet, err := defaultControllerManagerExtractor.Extract(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	resources := patternfinder.FindAllWithStarterRunes(commonMainMessage, finder, false, o.uidPrefixTokenCandidates...)
	writtenResourcePaths := map[uint32]struct{}{}

	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, componentFieldSet.ProjectID)
	gkeTimeline := gcpcommon.MustGKEClusterTimeline(ctx, projectTimeline, componentFieldSet.ClusterName)
	compTimeline := k8scontrolplane.MustControllerManagerControlPlaneTimeline(ctx, gkeTimeline, controllerManagerFieldSet.Controller)
	cs.AddEvent(compTimeline)

	for _, tPath := range controllerManagerFieldSet.AssociatedResourceTimelines(ctx, componentFieldSet.ClusterName) {
		cs.AddEvent(tPath)
		writtenResourcePaths[tPath.ID] = struct{}{}
	}

	for _, resource := range resources {
		tPath := k8saudit.MustResourceTimeline(ctx, componentFieldSet.ClusterName, resource.Value)
		if _, ok := writtenResourcePaths[tPath.ID]; ok {
			continue
		}
		cs.AddEvent(tPath)
		writtenResourcePaths[tPath.ID] = struct{}{}
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*ControllerManagerTimelineMapper)(nil)
