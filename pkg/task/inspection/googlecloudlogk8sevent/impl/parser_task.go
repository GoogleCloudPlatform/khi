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

package googlecloudlogk8sevent_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
)

// GKEK8sEventLogParseJob is the parser task for Kubernetes Event logs.
var GKEK8sEventLogParseJob = legacyparser.NewParserTaskFromParser(googlecloudlogk8sevent_contract.GKEK8sEventLogParserTaskID, &k8sEventParser{}, 2000, true, googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes)

type k8sEventParser struct {
}

// TargetLogType implements parsertask.Parser.
func (k *k8sEventParser) TargetLogType() enum.LogType {
	return enum.LogTypeEvent
}

// Description implements parsertask.Parser.
func (*k8sEventParser) Description() string {
	return `Gather kubernetes event logs and visualize these on the associated resource timeline.`
}

// GetParserName implements parsertask.Parser.
func (*k8sEventParser) GetParserName() string {
	return `Kubernetes Event Logs`
}

func (*k8sEventParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (*k8sEventParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8sevent_contract.GKEK8sEventLogQueryTaskID.Ref()
}

func (*k8sEventParser) Grouper() grouper.LogGrouper {
	return grouper.AllDependentLogGrouper
}

// Parse implements parsertask.Parser.
func (*k8sEventParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	if kind, err := l.ReadString("jsonPayload.kind"); err != nil {
		// Event exporter ingests cluster scoped logs without jsonPayload
		if textPayload, err := l.ReadString("textPayload"); err == nil {
			clusterName := l.ReadStringOrDefault("resource.labels.cluster_name", "Unknown")
			cs.AddEvent(resourcepath.Cluster(clusterName))
			cs.SetLogSummary(textPayload)
			return nil
		}
		return err
	} else if kind != "Event" {
		return fmt.Errorf("skipping kind:%s", kind)
	}
	apiVersion := l.ReadStringOrDefault("jsonPayload.involvedObject.apiVersion", "v1")

	kind := l.ReadStringOrDefault("jsonPayload.involvedObject.kind", "Unknown")

	name := l.ReadStringOrDefault("jsonPayload.involvedObject.name", "Unknown")

	namespace := l.ReadStringOrDefault("jsonPayload.involvedObject.namespace", "cluster-scope")
	if !strings.Contains(apiVersion, "/") {
		apiVersion = "core/" + apiVersion
	}

	cs.AddEvent(resourcepath.NameLayerGeneralItem(apiVersion, strings.ToLower(kind), namespace, name))
	cs.SetLogSummary(fmt.Sprintf("【%s】%s", l.ReadStringOrDefault("jsonPayload.reason", "Unknown"), l.ReadStringOrDefault("jsonPayload.message", "")))
	return nil
}

var _ legacyparser.Parser = (*k8sEventParser)(nil)
