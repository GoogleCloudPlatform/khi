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

package googlecloudlogk8scontrolplane_impl

import (
	"context"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
)

// HpaControllerLogFilterTask filters logs for HPA controller.
var HpaControllerLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudlogk8scontrolplane_contract.HpaControllerLogFilterTaskID,
	googlecloudlogk8scontrolplane_contract.ListLogEntriesTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		parserType, err := googlecloudlogk8scontrolplane_contract.ExtractK8sControlplaneComponentParserType(l.NodeReader)
		if err != nil {
			return false
		}
		return parserType == googlecloudlogk8scontrolplane_contract.ComponentParserTypeHPAController
	},
)

// HpaControllerGrouperTask groups HPA controller logs.
var HpaControllerGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogk8scontrolplane_contract.HpaControllerLogGrouperTaskID,
	googlecloudlogk8scontrolplane_contract.HpaControllerLogFilterTaskID.Ref(),
	func(ctx context.Context, log *log.Log) string {
		return "" // No grouping needed
	},
)

// HpaControllerTimelineMapper maps HPA controller logs to timeline paths.
type HpaControllerTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (m *HpaControllerTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (m *HpaControllerTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontrolplane_contract.HpaControllerLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (m *HpaControllerTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return googlecloudlogk8scontrolplane_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (m *HpaControllerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	componentFieldSet, err := googlecloudlogk8scontrolplane_contract.ExtractK8sControlplaneComponent(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}
	hpaFieldSet, err := googlecloudlogk8scontrolplane_contract.ExtractK8sHPAControllerComponent(l.NodeReader)
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, componentFieldSet.ProjectID)
	gkeTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, componentFieldSet.ClusterName)
	compTimeline := googlecloudlogk8scontrolplane_contract.MustControlPlaneComponentTimeline(ctx, gkeTimeline, componentFieldSet.ComponentName)
	cs.AddEvent(compTimeline)

	hpaNamespace := ""
	hpaName := ""
	if hpaFieldSet.FinalRecommendation != nil {
		hpaNamespace = hpaFieldSet.FinalRecommendation.HPANamespace
		hpaName = hpaFieldSet.FinalRecommendation.HPAName
	} else if hpaFieldSet.AtomicRecommendation != nil {
		hpaNamespace = hpaFieldSet.AtomicRecommendation.HPANamespace
		hpaName = hpaFieldSet.AtomicRecommendation.HPAName
	}

	if hpaNamespace != "" && hpaName != "" {
		clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, componentFieldSet.ClusterName)
		apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "autoscaling/v2")
		kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "horizontalpodautoscaler")
		namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, hpaNamespace)
		hpaTimeline := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, hpaName)
		cs.AddEvent(hpaTimeline)
	}

	if hpaFieldSet.FinalRecommendation != nil && hpaFieldSet.FinalRecommendation.TargetRef.HasValidResourceIdentity() {
		targetRef := hpaFieldSet.FinalRecommendation.TargetRef
		targetAPIVersion := targetRef.APIVersion
		if targetAPIVersion == "v1" {
			targetAPIVersion = "core/v1"
		}
		targetResource := &commonlogk8saudit_contract.ResourceIdentity{
			APIVersion: targetAPIVersion,
			Kind:       strings.ToLower(targetRef.Kind),
			Namespace:  hpaNamespace,
			Name:       targetRef.Name,
		}
		targetTimeline := commonlogk8saudit_contract.MustResourceTimeline(ctx, componentFieldSet.ClusterName, targetResource)
		cs.AddEvent(targetTimeline)
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*HpaControllerTimelineMapper)(nil)

// HpaControllerLogToTimelineMapperTask creates timeline events for HPA controller logs.
var HpaControllerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(googlecloudlogk8scontrolplane_contract.HpaControllerLogToTimelineMapperTaskID, &HpaControllerTimelineMapper{})
