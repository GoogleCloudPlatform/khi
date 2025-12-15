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

package googlecloudloggkeautoscaler_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"gopkg.in/yaml.v3"
)

var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudloggkeautoscaler_contract.FieldSetReaderTaskID, googlecloudloggkeautoscaler_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSetReader{},
})

var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(googlecloudloggkeautoscaler_contract.LogIngesterTaskID, googlecloudloggkeautoscaler_contract.ListLogEntriesTaskID.Ref())

var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudloggkeautoscaler_contract.LogGrouperTaskID, googlecloudloggkeautoscaler_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return "" // No grouping
	},
)

var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[struct{}](googlecloudloggkeautoscaler_contract.LogToTimelineMapperTaskID, &autoscalerLogToTimelineMapperTaskSetting{},
	inspectioncore_contract.FeatureTaskLabel(`GKE Autoscaler Logs`,
		`Gather logs related to cluster autoscaler behavior to show them on the timelines of resources related to the autoscaler decision.`,
		enum.LogTypeAutoscaler,
		8000,
		true,
		googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...),
)

type autoscalerLogToTimelineMapperTaskSetting struct{}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (a *autoscalerLogToTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (a *autoscalerLogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudloggkeautoscaler_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (a *autoscalerLogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudloggkeautoscaler_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (a *autoscalerLogToTimelineMapperTaskSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData struct{}) (struct{}, error) {
	autoscalerFieldSet := log.MustGetFieldSet(l, &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{})
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())

	if autoscalerFieldSet.DecisionLog != nil {
		parseDecision(clusterName, autoscalerFieldSet.DecisionLog, cs)
	}
	if autoscalerFieldSet.NoDecisionLog != nil {
		parseNoDecision(clusterName, autoscalerFieldSet.NoDecisionLog, cs)
	}
	if autoscalerFieldSet.ResultInfoLog != nil {
		err := parseResultInfo(clusterName, autoscalerFieldSet.ResultInfoLog, cs)
		if err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*autoscalerLogToTimelineMapperTaskSetting)(nil)

func parseDecision(clusterName string, decision *googlecloudloggkeautoscaler_contract.DecisionLog, cs *history.ChangeSet) {
	// Parse scale up event
	if decision.ScaleUp != nil {
		scaleUp := decision.ScaleUp
		nodepoolNames := []string{}
		requestedSum := 0
		for _, mig := range scaleUp.IncreasedMigs {
			cs.AddEvent(resourcepath.Mig(clusterName, mig.Mig.Nodepool, mig.Mig.Name))
			nodepoolNames = append(nodepoolNames, mig.Mig.Nodepool)
			requestedSum += mig.RequestedNodes
		}
		for _, pod := range scaleUp.TriggeringPods {
			cs.AddEvent(resourcepath.Pod(pod.Namespace, pod.Name))
		}
		cs.SetLogSummary(fmt.Sprintf("Scaling up nodepools by autoscaler: %s (requested: %d in total)", strings.Join(common.DedupStringArray(nodepoolNames), ","), requestedSum))
	}
	// Parse scale down event
	if decision.ScaleDown != nil {
		scaleDown := decision.ScaleDown
		nodepoolNames := []string{}
		for _, nodeToBeRemoved := range scaleDown.NodesToBeRemoved {
			cs.AddEvent(resourcepath.Node(nodeToBeRemoved.Node.Name))
			cs.AddEvent(resourcepath.Mig(clusterName, nodeToBeRemoved.Node.Mig.Nodepool, nodeToBeRemoved.Node.Mig.Name))
			for _, pod := range nodeToBeRemoved.EvictedPods {
				cs.AddEvent(resourcepath.Pod(pod.Namespace, pod.Name))
			}
			nodepoolNames = append(nodepoolNames, nodeToBeRemoved.Node.Mig.Nodepool)
		}
		cs.SetLogSummary(fmt.Sprintf("Scaling down nodepools by autoscaler: %s (Removing %d nodes in total)", strings.Join(common.DedupStringArray(nodepoolNames), ","), len(scaleDown.NodesToBeRemoved)))
	}
	// Nodepool creation event
	if decision.NodePoolCreated != nil {
		nodePoolCreated := decision.NodePoolCreated
		nodepools := []string{}
		for _, nodepool := range nodePoolCreated.NodePools {
			cs.AddEvent(resourcepath.Nodepool(clusterName, nodepool.Name))
			for _, mig := range nodepool.Migs {
				cs.AddEvent(resourcepath.Mig(clusterName, mig.Nodepool, mig.Name))
			}
			nodepools = append(nodepools, nodepool.Name)
		}
		cs.SetLogSummary(fmt.Sprintf("Nodepool created by node auto provisioner: %s", strings.Join(nodepools, ",")))
	}
	if decision.NodePoolDeleted != nil {
		nodepoolDeleted := decision.NodePoolDeleted
		for _, nodepool := range nodepoolDeleted.NodePoolNames {
			cs.AddEvent(resourcepath.Nodepool(clusterName, nodepool))
		}
		cs.SetLogSummary(fmt.Sprintf("Nodepool deleted by node auto provisioner: %s", strings.Join(nodepoolDeleted.NodePoolNames, ",")))
	}
	cs.SetLogSeverity(enum.SeverityWarning)
	cs.AddEvent(resourcepath.Autoscaler(clusterName))
}

func parseNoDecision(clusterName string, noDecision *googlecloudloggkeautoscaler_contract.NoDecisionStatusLog, cs *history.ChangeSet) {
	if noDecision.NoScaleUp != nil {
		noScaleUp := noDecision.NoScaleUp
		for _, mig := range noScaleUp.SkippedMigs {
			cs.AddEvent(resourcepath.Mig(clusterName, mig.Mig.Nodepool, mig.Mig.Name))
		}
		for _, groupItem := range noScaleUp.UnhandledPodGroups {
			cs.AddEvent(resourcepath.Pod(groupItem.PodGroup.SamplePod.Namespace, groupItem.PodGroup.SamplePod.Name))
			for _, rejectedMig := range groupItem.RejectedMigs {
				cs.AddEvent(resourcepath.Mig(clusterName, rejectedMig.Mig.Nodepool, rejectedMig.Mig.Name))
			}
		}
		cs.SetLogSummary("autoscaler decided not to scale up")
	}

	if noDecision.NoScaleDown != nil {
		noScaleDown := noDecision.NoScaleDown
		migs := map[string]googlecloudloggkeautoscaler_contract.MIGItem{}
		for _, node := range noScaleDown.Nodes {
			cs.AddEvent(resourcepath.Node(node.Node.Name))
			migs[node.Node.Mig.Id()] = node.Node.Mig
		}
		for _, mig := range migs {
			migResourcePath := resourcepath.Mig(clusterName, mig.Nodepool, mig.Name)
			cs.AddEvent(migResourcePath)
		}
		parameterStr := strings.Join(noDecision.NoScaleDown.Reason.Parameters, ",")
		if parameterStr != "" {
			parameterStr = fmt.Sprintf("(%s)", parameterStr)
		}
		cs.SetLogSummary(fmt.Sprintf("autoscaler decided not to scale down: %s%s", noDecision.NoScaleDown.Reason.MessageId, parameterStr))
	}
	cs.SetLogSeverity(enum.SeverityInfo)
	cs.AddEvent(resourcepath.Autoscaler(clusterName))
}

func parseResultInfo(clusterName string, resultInfo *googlecloudloggkeautoscaler_contract.ResultInfoLog, cs *history.ChangeSet) error {
	commonFieldSet := log.MustGetFieldSet(cs.Log, &log.CommonFieldSet{})
	statuses := []string{}
	for _, r := range resultInfo.Results {
		status := r.EventID
		if r.ErrorMsg != nil {
			parameersStr := ""
			if len(r.ErrorMsg.Parameters) > 0 {
				parameersStr = fmt.Sprintf("(%s)", strings.Join(r.ErrorMsg.Parameters, ","))
			}
			status += fmt.Sprintf("(Error:%s%s)", r.ErrorMsg.MessageId, parameersStr)
		} else {
			status += "(Success)"
		}
		statuses = append(statuses, status)
	}
	revisionState := enum.RevisionAutoscalerNoError
	if resultInfoHasErrors(resultInfo) {
		revisionState = enum.RevisionAutoscalerHasErrors
	}

	serializedResultsRaw, err := yaml.Marshal(resultInfo)
	if err != nil {
		return err
	}

	cs.AddRevision(resourcepath.Autoscaler(clusterName), &history.StagingResourceRevision{
		ChangeTime: commonFieldSet.Timestamp,
		State:      revisionState,
		Requestor:  "cluster-autoscaler",
		Body:       string(serializedResultsRaw),
	})
	cs.SetLogSeverity(enum.SeverityInfo)
	cs.SetLogSummary(fmt.Sprintf("autoscaler finished events: %s", strings.Join(statuses, ",")))
	return nil
}

// resultInfoHasErrors returns if the given resultInfo contains any error events.
func resultInfoHasErrors(resultInfo *googlecloudloggkeautoscaler_contract.ResultInfoLog) bool {
	if resultInfo == nil {
		return false
	}
	for _, r := range resultInfo.Results {
		if r.ErrorMsg != nil {
			return true
		}
	}
	return false
}
