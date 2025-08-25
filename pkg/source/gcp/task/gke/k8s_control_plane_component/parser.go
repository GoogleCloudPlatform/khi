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

package k8scontrolplanecomponent

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/parsertask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_control_plane_component/componentparser"
	k8s_control_plane_component_taskid "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_control_plane_component/taskid"
)

type k8sControlPlaneComponentParser struct {
}

// TargetLogType implements parsertask.Parser.
func (k *k8sControlPlaneComponentParser) TargetLogType() enum.LogType {
	return enum.LogTypeControlPlaneComponent
}

// Dependencies implements parsertask.Parser.
func (k *k8sControlPlaneComponentParser) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// Description implements parsertask.Parser.
func (k *k8sControlPlaneComponentParser) Description() string {
	return `Gather Kubernetes control plane component(e.g kube-scheduler, kube-controller-manager,api-server) logs`
}

// GetParserName implements parsertask.Parser.
func (k *k8sControlPlaneComponentParser) GetParserName() string {
	return `Kubernetes Control plane component logs`
}

// Grouper implements parsertask.Parser.
func (k *k8sControlPlaneComponentParser) Grouper() grouper.LogGrouper {
	return grouper.NewSingleStringFieldKeyLogGrouper("resource.labels.component_name")
}

// LogTask implements parsertask.Parser.
func (k *k8sControlPlaneComponentParser) LogTask() taskid.TaskReference[[]*log.Log] {
	return k8s_control_plane_component_taskid.GKEK8sControlPlaneComponentQueryTaskID.Ref()
}

// Parse implements parsertask.Parser.
func (k *k8sControlPlaneComponentParser) Parse(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder) error {
	component := l.ReadStringOrDefault("resource.labels.component_name", "Unknown")
	for i := 0; i < len(componentparser.ComponentParsers); i++ {
		cp := componentparser.ComponentParsers[i]
		if cp.ShouldProcess(component) {
			next, err := cp.Process(ctx, l, cs, builder)
			if err != nil {
				return err
			}
			if !next {
				break
			}
		}
	}

	return nil
}

var _ parsertask.Parser = (*k8sControlPlaneComponentParser)(nil)

var GKEK8sControlPlaneComponentLogParseTask = parsertask.NewParserTaskFromParser(k8s_control_plane_component_taskid.GKEK8sControlPlaneComponentParserTaskID, &k8sControlPlaneComponentParser{}, true, inspectiontype.GCPK8sClusterInspectionTypes)
