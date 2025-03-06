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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	gcp_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

func GenerateK8sControlPlaneQuery(clusterName string, projectId string, controlplaneComponentFilter *queryutil.SetFilterParseResult) string {
	return fmt.Sprintf(`resource.type="k8s_control_plane_component"
resource.labels.cluster_name="%s"
resource.labels.project_id="%s"
-sourceLocation.file="httplog.go"
%s`, clusterName, projectId, generateK8sControlPlaneComponentFilter(controlplaneComponentFilter))
}

const GKEK8sControlPlaneComponentQueryTaskID = query.GKEQueryPrefix + "k8s-controlplane"

var GKEK8sControlPlaneLogQueryTask = query.NewQueryGeneratorTask(GKEK8sControlPlaneComponentQueryTaskID, "K8s control plane logs", enum.LogTypeControlPlaneComponent, []string{
	gcp_task.InputProjectIdTaskID,
	gcp_task.InputClusterNameTaskID,
	InputControlPlaneComponentNameFilterTaskID,
}, func(ctx context.Context, i int, vs *task.VariableSet) ([]string, error) {
	clusterName, err := gcp_task.GetInputClusterNameFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	projectId, err := gcp_task.GetInputProjectIdFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	controlPlaneComponentNameFilter, err := GetInputControlPlaneComponentNameFilterFromTaskVariable(vs)
	if err != nil {
		return []string{}, err
	}
	return []string{GenerateK8sControlPlaneQuery(clusterName, projectId, controlPlaneComponentNameFilter)}, nil
}, GenerateK8sControlPlaneQuery("gcp-cluster-name", "gcp-project-id", &queryutil.SetFilterParseResult{
	SubtractMode: true,
}))

func generateK8sControlPlaneComponentFilter(filter *queryutil.SetFilterParseResult) string {
	if filter.ValidationError != "" {
		return fmt.Sprintf(`-- Failed to generate component name filter due to the validation error "%s"`, filter.ValidationError)
	}
	if filter.SubtractMode {
		if len(filter.Subtractives) == 0 {
			return "-- No component name filter"
		}
		return fmt.Sprintf(`-resource.labels.component_name:(%s)`, strings.Join(queryutil.WrapDoubleQuoteForStringArray(filter.Subtractives), " OR "))
	} else {
		if len(filter.Additives) == 0 {
			return `-- Invalid: none of the controlplane component will be selected. Ignoreing component name filter.`
		}
		return fmt.Sprintf(`resource.labels.component_name:(%s)`, strings.Join(queryutil.WrapDoubleQuoteForStringArray(filter.Additives), " OR "))
	}
}
