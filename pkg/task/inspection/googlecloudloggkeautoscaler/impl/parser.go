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
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
)

type autoscalerLogParser struct {
}

// TargetLogType implements parsertask.Parser.
func (p *autoscalerLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeAutoscaler
}

// Dependencies implements parsertask.Parser.
func (*autoscalerLogParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	}
}

// Description implements parsertask.Parser.
func (*autoscalerLogParser) Description() string {
	return `Gather logs related to cluster autoscaler behavior to show them on the timelines of resources related to the autoscaler decision.`
}

// GetParserName implements parsertask.Parser.
func (*autoscalerLogParser) GetParserName() string {
	return `Autoscaler Logs`
}

// LogTask implements parsertask.Parser.
func (*autoscalerLogParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudloggkeautoscaler_contract.AutoscalerQueryTaskID.Ref()
}

func (*autoscalerLogParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parsertask.Parser.
func (p *autoscalerLogParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())

	// scaleUp,scaleDown,nodePoolCreated,nodePoolDeleted
	if l.Has("jsonPayload.decision") {
		err := parseDecision(ctx, clusterName, l, cs, builder)
		if err != nil {
			var yaml string
			yamlBytes, err := l.Serialize("", &structured.YAMLNodeSerializer{})
			if err != nil {
				yaml = "ERROR!! Failed to dump in YAML"
			} else {
				yaml = string(yamlBytes)
			}
			return fmt.Errorf("Failed to parse decision log:\nERROR:%s\n\n:SOURCE LOG:\n%s", err, yaml)
		}
	}
	if l.Has("jsonPayload.noDecisionStatus") {
		err := parseNoDecision(ctx, clusterName, l, cs, builder)
		if err != nil {
			return err
		}
	}
	if l.Has("jsonPayload.resultInfo") {
		err := parseResultInfo(ctx, clusterName, l, cs, builder)
		if err != nil {
			return err
		}
	}
	cs.AddEvent(resourcepath.Autoscaler(clusterName))
	return nil
}

func parseDecision(ctx context.Context, clusterName string, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	jsonDecisionReader, err := l.GetReader("jsonPayload.decision")
	if err != nil {
		return err
	}
	decision, err := parseDecisionFromReader(jsonDecisionReader)
	if err != nil {
		return err
	}
	// Parse scale up event
	if decision.ScaleUp != nil {
		scaleUp := decision.ScaleUp
		nodepoolNames := []string{}
		requestedSum := 0
		for _, mig := range scaleUp.IncreasedMigs {
			migResourcePath := resourcepath.Mig(clusterName, mig.Mig.Nodepool, mig.Mig.Name)
			cs.AddEvent(migResourcePath)
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
			migResourcePath := resourcepath.Mig(clusterName, nodeToBeRemoved.Node.Mig.Nodepool, nodeToBeRemoved.Node.Name)
			cs.AddEvent(resourcepath.Node(nodeToBeRemoved.Node.Name))
			cs.AddEvent(migResourcePath)
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
				migResourcePath := resourcepath.Mig(clusterName, mig.Nodepool, mig.Name)
				cs.AddEvent(migResourcePath)
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
	return nil
}

func parseNoDecision(ctx context.Context, clusterName string, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	jsonNoDecisionReader, err := l.GetReader("jsonPayload.noDecisionStatus")
	if err != nil {
		return err
	}
	noDecision, err := parseNoDecisionFromReader(jsonNoDecisionReader)
	if err != nil {
		return err
	}
	if noDecision.NoScaleUp != nil {
		noScaleUp := noDecision.NoScaleUp
		for _, mig := range noScaleUp.SkippedMigs {
			migResourcePath := resourcepath.Mig(clusterName, mig.Mig.Nodepool, mig.Mig.Name)
			cs.AddEvent(migResourcePath)
		}
		cs.SetLogSummary("autoscaler decided not to scale up")
		// TODO: support unhandled migs
	}

	if noDecision.NoScaleDown != nil {
		noScaleDown := noDecision.NoScaleDown
		migs := map[string]mig{}
		for _, node := range noScaleDown.Nodes {
			cs.AddEvent(resourcepath.Node(node.Node.Name))
			migs[node.Node.Mig.Id()] = node.Node.Mig
		}
		for _, mig := range migs {
			migResourcePath := resourcepath.Mig(clusterName, mig.Nodepool, mig.Name)
			cs.AddEvent(migResourcePath)
		}
		cs.SetLogSummary("autoscaler decided not to scale down")
	}
	cs.SetLogSeverity(enum.SeverityInfo)
	return nil
}

func parseResultInfo(ctx context.Context, clusterName string, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	jsonResultInfoReader, err := l.GetReader("jsonPayload.resultInfo")
	if err != nil {
		return err
	}
	resultInfo, err := parseResultInfoFromReader(jsonResultInfoReader)
	if err != nil {
		return err
	}
	statuses := []string{}
	for _, r := range resultInfo.Results {
		status := r.EventID
		if r.ErrorMsg != nil {
			status += fmt.Sprintf("(Error:%s)", r.ErrorMsg.MessageId)
		} else {
			status += "(Success)"
		}
		statuses = append(statuses, status)
	}
	cs.SetLogSeverity(enum.SeverityInfo)
	cs.SetLogSummary(fmt.Sprintf("autoscaler finished events: %s", strings.Join(statuses, ",")))
	return nil
}

var _ legacyparser.Parser = (*autoscalerLogParser)(nil)

var AutoscalerParserTask = legacyparser.NewParserTaskFromParser(googlecloudloggkeautoscaler_contract.AutoscalerParserTaskID, &autoscalerLogParser{}, 8000, true, googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes)
