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

package autoscaler

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type autoscalerLogParser struct {
}

// TargetLogType implements parser.Parser.
func (p *autoscalerLogParser) TargetLogType() enum.LogType {
	return enum.LogTypeAutoscaler
}

// Dependencies implements parser.Parser.
func (*autoscalerLogParser) Dependencies() []string {
	return []string{
		gcp_task.InputClusterNameTaskID,
	}
}

// Description implements parser.Parser.
func (*autoscalerLogParser) Description() string {
	return `Gather logs related to cluster autoscaler behavior to show them on the timelines of resources related to the autoscaler decision.`
}

// GetParserName implements parser.Parser.
func (*autoscalerLogParser) GetParserName() string {
	return `Autoscaler Logs`
}

// LogTask implements parser.Parser.
func (*autoscalerLogParser) LogTask() string {
	return AutoscalerQueryTaskID
}

func (*autoscalerLogParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parser.Parser.
func (p *autoscalerLogParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, variables *task.VariableSet) error {
	clusterName, err := gcp_task.GetInputClusterNameFromTaskVariable(variables)
	if err != nil {
		return err
	}
	// scaleUp,scaleDown,nodePoolCreated,nodePoolDeleted
	if l.Has("jsonPayload.decision") {
		err := parseDecision(ctx, clusterName, l, cs, builder)
		if err != nil {
			yaml, err := l.Fields.ToYaml("")
			if err != nil {
				yaml = "ERROR!! Failed to dump in YAML"
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
	cs.RecordEvent(resourcepath.Autoscaler(clusterName))
	return nil
}

func parseDecision(ctx context.Context, clusterName string, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	jsonDecisionReader, err := l.Fields.ReaderSingle("jsonPayload.decision")
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
			cs.RecordEvent(migResourcePath)
			nodepoolNames = append(nodepoolNames, mig.Mig.Nodepool)
			requestedSum += mig.RequestedNodes
		}
		for _, pod := range scaleUp.TriggeringPods {
			cs.RecordEvent(resourcepath.Pod(pod.Namespace, pod.Name))
		}
		cs.RecordLogSummary(fmt.Sprintf("Scaling up nodepools by autoscaler: %s (requested: %d in total)", strings.Join(common.DedupStringArray(nodepoolNames), ","), requestedSum))
	}
	// Parse scale down event
	if decision.ScaleDown != nil {
		scaleDown := decision.ScaleDown
		nodepoolNames := []string{}
		for _, nodeToBeRemoved := range scaleDown.NodesToBeRemoved {
			migResourcePath := resourcepath.Mig(clusterName, nodeToBeRemoved.Node.Mig.Nodepool, nodeToBeRemoved.Node.Name)
			cs.RecordEvent(resourcepath.Node(nodeToBeRemoved.Node.Name))
			cs.RecordEvent(migResourcePath)
			for _, pod := range nodeToBeRemoved.EvictedPods {
				cs.RecordEvent(resourcepath.Pod(pod.Namespace, pod.Name))
			}
			nodepoolNames = append(nodepoolNames, nodeToBeRemoved.Node.Mig.Nodepool)
		}
		cs.RecordLogSummary(fmt.Sprintf("Scaling down nodepools by autoscaler: %s (Removing %d nodes in total)", strings.Join(common.DedupStringArray(nodepoolNames), ","), len(scaleDown.NodesToBeRemoved)))
	}
	// Nodepool creation event
	if decision.NodePoolCreated != nil {
		nodePoolCreated := decision.NodePoolCreated
		nodepools := []string{}
		for _, nodepool := range nodePoolCreated.NodePools {
			cs.RecordEvent(resourcepath.Nodepool(clusterName, nodepool.Name))
			for _, mig := range nodepool.Migs {
				migResourcePath := resourcepath.Mig(clusterName, mig.Nodepool, mig.Name)
				cs.RecordEvent(migResourcePath)
			}
			nodepools = append(nodepools, nodepool.Name)
		}
		cs.RecordLogSummary(fmt.Sprintf("Nodepool created by node auto provisioner: %s", strings.Join(nodepools, ",")))
	}
	if decision.NodePoolDeleted != nil {
		nodepoolDeleted := decision.NodePoolDeleted
		for _, nodepool := range nodepoolDeleted.NodePoolNames {
			cs.RecordEvent(resourcepath.Nodepool(clusterName, nodepool))
		}
		cs.RecordLogSummary(fmt.Sprintf("Nodepool deleted by node auto provisioner: %s", strings.Join(nodepoolDeleted.NodePoolNames, ",")))
	}
	cs.RecordLogSeverity(enum.SeverityWarning)
	return nil
}

func parseNoDecision(ctx context.Context, clusterName string, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	jsonNoDecisionReader, err := l.Fields.ReaderSingle("jsonPayload.noDecisionStatus")
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
			cs.RecordEvent(migResourcePath)
		}
		cs.RecordLogSummary("autoscaler decided not to scale up")
		// TODO: support unhandled migs
	}

	if noDecision.NoScaleDown != nil {
		noScaleDown := noDecision.NoScaleDown
		migs := map[string]mig{}
		for _, node := range noScaleDown.Nodes {
			cs.RecordEvent(resourcepath.Node(node.Node.Name))
			migs[node.Node.Mig.Id()] = node.Node.Mig
		}
		for _, mig := range migs {
			migResourcePath := resourcepath.Mig(clusterName, mig.Nodepool, mig.Name)
			cs.RecordEvent(migResourcePath)
		}
		cs.RecordLogSummary("autoscaler decided not to scale down")
	}
	cs.RecordLogSeverity(enum.SeverityInfo)
	return nil
}

func parseResultInfo(ctx context.Context, clusterName string, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	jsonResultInfoReader, err := l.Fields.ReaderSingle("jsonPayload.resultInfo")
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
	cs.RecordLogSeverity(enum.SeverityInfo)
	cs.RecordLogSummary(fmt.Sprintf("autoscaler finished events: %s", strings.Join(statuses, ",")))
	return nil
}

var _ parser.Parser = (*autoscalerLogParser)(nil)

var AutoscalerParserTask = parser.NewParserTaskFromParser(gcp_task.GCPPrefix+"feature/autoscaler-parser", &autoscalerLogParser{}, true, inspection_task.InspectionTypeLabel(gke.InspectionTypeId, composer_task.InspectionTypeId))
