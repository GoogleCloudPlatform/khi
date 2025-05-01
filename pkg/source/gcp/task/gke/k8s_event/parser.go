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

package k8s_event

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/parser"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	k8s_event_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_event/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var GKEK8sEventLogParseJob = parser.NewParserTaskFromParser(k8s_event_taskid.GKEK8sEventLogParserTaskID, &k8sEventParser{}, true, inspectiontype.GCPK8sClusterInspectionTypes)

type k8sEventParser struct {
}

// TargetLogType implements parser.Parser.
func (k *k8sEventParser) TargetLogType() enum.LogType {
	return enum.LogTypeEvent
}

// Description implements parser.Parser.
func (*k8sEventParser) Description() string {
	return `Gather kubernetes event logs and visualize these on the associated resource timeline.`
}

// GetParserName implements parser.Parser.
func (*k8sEventParser) GetParserName() string {
	return `Kubernetes Event Logs`
}

func (*k8sEventParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*k8sEventParser) LogTask() taskid.TaskReference[[]*log.LogEntity] {
	return k8s_event_taskid.GKEK8sEventLogQueryTaskID.Ref()
}

func (*k8sEventParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parser.Parser.
func (*k8sEventParser) Parse(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) error {
	if kind, err := l.GetString("jsonPayload.kind"); err != nil {
		// Event exporter ingests cluster scoped logs without jsonPayload
		if textPayload, err := l.GetString("textPayload"); err == nil {
			clusterName := l.GetStringOrDefault("resource.labels.cluster_name", "Unknown")
			cs.RecordEvent(resourcepath.Cluster(clusterName))
			cs.RecordLogSummary(textPayload)
			return nil
		}
		return err
	} else if kind != "Event" {
		return fmt.Errorf("skipping kind:%s", kind)
	}
	apiVersion := l.GetStringOrDefault("jsonPayload.involvedObject.apiVersion", "v1")

	kind := l.GetStringOrDefault("jsonPayload.involvedObject.kind", "Unknown")

	name := l.GetStringOrDefault("jsonPayload.involvedObject.name", "Unknown")

	namespace := l.GetStringOrDefault("jsonPayload.involvedObject.namespace", "cluster-scope")
	if !strings.Contains(apiVersion, "/") {
		apiVersion = "core/" + apiVersion
	}

	cs.RecordEvent(resourcepath.NameLayerGeneralItem(apiVersion, strings.ToLower(kind), namespace, name))
	cs.RecordLogSummary(fmt.Sprintf("【%s】%s", l.GetStringOrDefault("jsonPayload.reason", "Unknown"), l.GetStringOrDefault("jsonPayload.message", "")))
	return nil
}

var _ parser.Parser = (*k8sEventParser)(nil)
