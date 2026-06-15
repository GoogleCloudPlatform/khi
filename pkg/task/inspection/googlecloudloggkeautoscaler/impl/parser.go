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
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"gopkg.in/yaml.v3"
)

// FieldSetReaderTask reads autoscaler fields from the raw logs.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudloggkeautoscaler_contract.FieldSetReaderTaskID, googlecloudloggkeautoscaler_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSetReader{},
})

type autoscalerLogIngester struct{}

// RawLogTask returns the task that provides raw logs for ingestion.
func (i *autoscalerLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudloggkeautoscaler_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns additional dependencies of the ingester.
func (i *autoscalerLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog ingests metadata for a single log.
func (i *autoscalerLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudloggkeautoscaler_contract.LogTypeAutoscaler)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	autoscalerFieldSet, err := log.GetFieldSet(l, &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{})
	if err != nil {
		return nil, err
	}

	switch {
	case autoscalerFieldSet.DecisionLog != nil:
		cs.SetSeverity(inspectioncore_contract.SeverityWarning)
		cs.SetSummary(getDecisionSummary(autoscalerFieldSet.DecisionLog))
	case autoscalerFieldSet.NoDecisionLog != nil:
		cs.SetSeverity(inspectioncore_contract.SeverityInfo)
		cs.SetSummary(getNoDecisionSummary(autoscalerFieldSet.NoDecisionLog))
	case autoscalerFieldSet.ResultInfoLog != nil:
		cs.SetSeverity(inspectioncore_contract.SeverityInfo)
		cs.SetSummary(getResultInfoSummary(autoscalerFieldSet.ResultInfoLog))
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*autoscalerLogIngester)(nil)

// LogIngesterTask serializes the ingested log metadata into the builder.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(
	googlecloudloggkeautoscaler_contract.LogIngesterTaskID,
	&autoscalerLogIngester{},
)

// LogGrouperTask groups logs (no grouping needed for autoscaler logs).
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudloggkeautoscaler_contract.LogGrouperTaskID, googlecloudloggkeautoscaler_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		return "" // No grouping
	},
)

type autoscalerTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

// LogIngesterTask returns the prerequisite log serializer task.
func (m *autoscalerTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudloggkeautoscaler_contract.LogIngesterTaskID.Ref()
}

// Dependencies returns additional dependencies of the mapper.
func (m *autoscalerTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// GroupedLogTask returns the log grouper task reference.
func (m *autoscalerTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudloggkeautoscaler_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup processes a single log and registers its timeline event/revision mutations.
func (m *autoscalerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	autoscalerFieldSet, err := log.GetFieldSet(l, &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	clusterName := autoscalerFieldSet.ClusterName

	cs := khifilev6.NewTimelineChangeSet(l)

	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, autoscalerFieldSet.ProjectID)
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, clusterName)

	if autoscalerFieldSet.DecisionLog != nil {
		err := mapDecision(ctx, clusterName, clusterTimeline, autoscalerFieldSet.DecisionLog, cs)
		if err != nil {
			return nil, struct{}{}, err
		}
	}
	if autoscalerFieldSet.NoDecisionLog != nil {
		err := mapNoDecision(ctx, clusterName, clusterTimeline, autoscalerFieldSet.NoDecisionLog, cs)
		if err != nil {
			return nil, struct{}{}, err
		}
	}
	if autoscalerFieldSet.ResultInfoLog != nil {
		err := mapResultInfo(ctx, clusterTimeline, autoscalerFieldSet.ResultInfoLog, cs)
		if err != nil {
			return nil, struct{}{}, err
		}
	}
	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*autoscalerTimelineMapper)(nil)

// LogToTimelineMapperTask maps the autoscaler logs to respective resource timelines.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(
	googlecloudloggkeautoscaler_contract.LogToTimelineMapperTaskID,
	&autoscalerTimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(
		`GKE Autoscaler Logs`,
		`Gather logs related to cluster autoscaler behavior to show them on the timelines of resources related to the autoscaler decision.`,
		8000,
		true,
	),
)

func getDecisionSummary(decision *googlecloudloggkeautoscaler_contract.DecisionLog) string {
	if decision.ScaleUp != nil {
		scaleUp := decision.ScaleUp
		nodepoolNames := []string{}
		requestedSum := 0
		for _, mig := range scaleUp.IncreasedMigs {
			nodepoolNames = append(nodepoolNames, mig.Mig.Nodepool)
			requestedSum += mig.RequestedNodes
		}
		return fmt.Sprintf("Scaling up nodepools by autoscaler: %s (requested: %d in total)", strings.Join(common.DedupStringArray(nodepoolNames), ","), requestedSum)
	}
	if decision.ScaleDown != nil {
		scaleDown := decision.ScaleDown
		nodepoolNames := []string{}
		for _, nodeToBeRemoved := range scaleDown.NodesToBeRemoved {
			nodepoolNames = append(nodepoolNames, nodeToBeRemoved.Node.Mig.Nodepool)
		}
		return fmt.Sprintf("Scaling down nodepools by autoscaler: %s (Removing %d nodes in total)", strings.Join(common.DedupStringArray(nodepoolNames), ","), len(scaleDown.NodesToBeRemoved))
	}
	if decision.NodePoolCreated != nil {
		nodePoolCreated := decision.NodePoolCreated
		nodepools := []string{}
		for _, nodepool := range nodePoolCreated.NodePools {
			nodepools = append(nodepools, nodepool.Name)
		}
		return fmt.Sprintf("Nodepool created by node auto provisioner: %s", strings.Join(nodepools, ","))
	}
	if decision.NodePoolDeleted != nil {
		nodepoolDeleted := decision.NodePoolDeleted
		return fmt.Sprintf("Nodepool deleted by node auto provisioner: %s", strings.Join(nodepoolDeleted.NodePoolNames, ","))
	}
	return ""
}

func getNoDecisionSummary(noDecision *googlecloudloggkeautoscaler_contract.NoDecisionStatusLog) string {
	if noDecision.NoScaleUp != nil {
		return "autoscaler decided not to scale up"
	}
	if noDecision.NoScaleDown != nil {
		parameterStr := strings.Join(noDecision.NoScaleDown.Reason.Parameters, ",")
		if parameterStr != "" {
			parameterStr = fmt.Sprintf("(%s)", parameterStr)
		}
		return fmt.Sprintf("autoscaler decided not to scale down: %s%s", noDecision.NoScaleDown.Reason.MessageId, parameterStr)
	}
	return ""
}

func getResultInfoSummary(resultInfo *googlecloudloggkeautoscaler_contract.ResultInfoLog) string {
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
	return fmt.Sprintf("autoscaler finished events: %s", strings.Join(statuses, ","))
}

func getPodTimeline(ctx context.Context, clusterName string, namespace string, podName string) *khifilev6.TimelinePath {
	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, namespace)
	return commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, podName)
}

func getNodeTimeline(ctx context.Context, clusterName string, nodeName string) *khifilev6.TimelinePath {
	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
	return commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kindTimeline, nodeName)
}

func mapDecision(ctx context.Context, clusterName string, clusterTimeline *khifilev6.TimelinePath, decision *googlecloudloggkeautoscaler_contract.DecisionLog, cs *khifilev6.TimelineChangeSet) error {
	if decision.ScaleUp != nil {
		scaleUp := decision.ScaleUp
		for _, mig := range scaleUp.IncreasedMigs {
			nodepoolPath := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, mig.Mig.Nodepool)
			migPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolPath, mig.Mig.Name)
			cs.AddEvent(migPath)
		}
		for _, pod := range scaleUp.TriggeringPods {
			podPath := getPodTimeline(ctx, clusterName, pod.Namespace, pod.Name)
			cs.AddEvent(podPath)
		}
	}
	if decision.ScaleDown != nil {
		scaleDown := decision.ScaleDown
		for _, nodeToBeRemoved := range scaleDown.NodesToBeRemoved {
			nodePath := getNodeTimeline(ctx, clusterName, nodeToBeRemoved.Node.Name)
			cs.AddEvent(nodePath)
			nodepoolPath := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, nodeToBeRemoved.Node.Mig.Nodepool)
			migPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolPath, nodeToBeRemoved.Node.Mig.Name)
			cs.AddEvent(migPath)
			for _, pod := range nodeToBeRemoved.EvictedPods {
				podPath := getPodTimeline(ctx, clusterName, pod.Namespace, pod.Name)
				cs.AddEvent(podPath)
			}
		}
	}
	if decision.NodePoolCreated != nil {
		nodePoolCreated := decision.NodePoolCreated
		for _, nodepool := range nodePoolCreated.NodePools {
			nodepoolPath := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, nodepool.Name)
			cs.AddEvent(nodepoolPath)
			for _, mig := range nodepool.Migs {
				migPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolPath, mig.Name)
				cs.AddEvent(migPath)
			}
		}
	}
	if decision.NodePoolDeleted != nil {
		nodepoolDeleted := decision.NodePoolDeleted
		for _, nodepool := range nodepoolDeleted.NodePoolNames {
			nodepoolPath := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, nodepool)
			cs.AddEvent(nodepoolPath)
		}
	}
	autoscalerPath := googlecloudloggkeautoscaler_contract.MustAutoscalerTimeline(ctx, clusterTimeline)
	cs.AddEvent(autoscalerPath)
	return nil
}

func mapNoDecision(ctx context.Context, clusterName string, clusterTimeline *khifilev6.TimelinePath, noDecision *googlecloudloggkeautoscaler_contract.NoDecisionStatusLog, cs *khifilev6.TimelineChangeSet) error {
	if noDecision.NoScaleUp != nil {
		noScaleUp := noDecision.NoScaleUp
		for _, mig := range noScaleUp.SkippedMigs {
			nodepoolPath := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, mig.Mig.Nodepool)
			migPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolPath, mig.Mig.Name)
			cs.AddEvent(migPath)
		}
		for _, groupItem := range noScaleUp.UnhandledPodGroups {
			podPath := getPodTimeline(ctx, clusterName, groupItem.PodGroup.SamplePod.Namespace, groupItem.PodGroup.SamplePod.Name)
			cs.AddEvent(podPath)
			for _, rejectedMig := range groupItem.RejectedMigs {
				nodepoolPath := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, rejectedMig.Mig.Nodepool)
				migPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolPath, rejectedMig.Mig.Name)
				cs.AddEvent(migPath)
			}
		}
	}

	if noDecision.NoScaleDown != nil {
		noScaleDown := noDecision.NoScaleDown
		migs := map[string]googlecloudloggkeautoscaler_contract.MIGItem{}
		for _, node := range noScaleDown.Nodes {
			nodePath := getNodeTimeline(ctx, clusterName, node.Node.Name)
			cs.AddEvent(nodePath)
			migs[node.Node.Mig.Id()] = node.Node.Mig
		}
		for _, mig := range migs {
			nodepoolPath := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, mig.Nodepool)
			migPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolPath, mig.Name)
			cs.AddEvent(migPath)
		}
	}
	autoscalerPath := googlecloudloggkeautoscaler_contract.MustAutoscalerTimeline(ctx, clusterTimeline)
	cs.AddEvent(autoscalerPath)
	return nil
}

func mapResultInfo(ctx context.Context, clusterTimeline *khifilev6.TimelinePath, resultInfo *googlecloudloggkeautoscaler_contract.ResultInfoLog, cs *khifilev6.TimelineChangeSet) error {
	commonFieldSet, err := log.GetFieldSet(cs.Log, &log.CommonFieldSet{})
	if err != nil {
		return err
	}
	revisionState := googlecloudloggkeautoscaler_contract.RevisionAutoscalerNoError
	if resultInfoHasErrors(resultInfo) {
		revisionState = googlecloudloggkeautoscaler_contract.RevisionAutoscalerHasErrors
	}

	serializedResultsRaw, err := yaml.Marshal(resultInfo)
	if err != nil {
		return err
	}

	bodyNode, err := structured.FromYAML(string(serializedResultsRaw))
	if err != nil {
		return err
	}

	autoscalerPath := googlecloudloggkeautoscaler_contract.MustAutoscalerTimeline(ctx, clusterTimeline)
	cs.AddRevision(autoscalerPath, &khifilev6.StagingRevision{
		ChangedTime:  commonFieldSet.Timestamp,
		StateType:    revisionState,
		Principal:    "cluster-autoscaler",
		ResourceBody: bodyNode,
	})
	return nil
}

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
