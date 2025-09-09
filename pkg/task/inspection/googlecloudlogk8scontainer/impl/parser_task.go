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

package googlecloudlogk8scontainer_impl

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
)

type k8sContainerParser struct {
}

// TargetLogType implements parsertask.Parser.
func (k *k8sContainerParser) TargetLogType() enum.LogType {
	return enum.LogTypeContainer
}

// Description implements parsertask.Parser.
func (*k8sContainerParser) Description() string {
	return `Gather stdout/stderr logs of containers on the cluster to visualize them on the timeline under an associated Pod. Log volume can be huge when the cluster has many Pods.`
}

// GetParserName implements parsertask.Parser.
func (*k8sContainerParser) GetParserName() string {
	return "Kubernetes container logs"
}

func (*k8sContainerParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*k8sContainerParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontainer_contract.GKEContainerLogQueryTaskID.Ref()
}

func (*k8sContainerParser) Grouper() grouper.LogGrouper {
	return grouper.NewSingleStringFieldKeyLogGrouper("resource.labels.pod_name")
}

// Parse implements parsertask.Parser.
func (*k8sContainerParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	mainMessageFieldSet := log.MustGetFieldSet(l, &log.MainMessageFieldSet{})
	mainMessage := mainMessageFieldSet.MainMessage
	namespace := l.ReadStringOrDefault("resource.labels.namespace_name", "unknown")
	podName := l.ReadStringOrDefault("resource.labels.pod_name", "unknown")
	containerName := l.ReadStringOrDefault("resource.labels.container_name", "unknown")
	if namespace == "" {
		namespace = "unknown"
	}
	if podName == "" {
		podName = "unknown"
	}
	if containerName == "" {
		containerName = "unknown"
	}

	if mainMessage == "" {
		yamlRaw, err := l.Serialize("", &structured.YAMLNodeSerializer{})
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("failed to extract main message from a container log then failed to serialize the log content.\nError message:\n%v", err))
		} else {
			slog.WarnContext(ctx, fmt.Sprintf("failed to extract main message from a container log.\nLog content:\n%s", string(yamlRaw)))
		}
		mainMessage = "(unknown)"
	}
	severityOverride := ParseSeverity(mainMessage)
	containerPath := resourcepath.Container(namespace, podName, containerName)
	cs.AddEvent(containerPath)
	cs.SetLogSummary(mainMessage)
	if severityOverride != enum.SeverityUnknown {
		cs.SetLogSeverity(severityOverride)
	}
	return nil
}

var _ legacyparser.Parser = (*k8sContainerParser)(nil)

// GKEContainerLogParseJob is a parser task for GKE container logs.
var GKEContainerLogParseJob = legacyparser.NewParserTaskFromParser(googlecloudlogk8scontainer_contract.GKEContainerParserTaskID, &k8sContainerParser{}, 4000, false, googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes)
