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

package googlecloudlogk8scontrolplane_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/legacyparser"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/grouper"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/impl/componentparser"
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
	return googlecloudlogk8scontrolplane_contract.GKEK8sControlPlaneComponentQueryTaskID.Ref()
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

var _ legacyparser.Parser = (*k8sControlPlaneComponentParser)(nil)

var GKEK8sControlPlaneComponentLogParseTask = legacyparser.NewParserTaskFromParser(googlecloudlogk8scontrolplane_contract.GKEK8sControlPlaneComponentParserTaskID, &k8sControlPlaneComponentParser{}, 9000, false, googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes)
