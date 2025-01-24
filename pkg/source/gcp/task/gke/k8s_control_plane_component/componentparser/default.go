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

package componentparser

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type DefaultK8sControlPlaneComponentParser struct {
}

// Process implements ControlPlaneComponentParser.
func (d *DefaultK8sControlPlaneComponentParser) Process(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder, v *task.VariableSet) (bool, error) {
	component := l.GetStringOrDefault("resource.labels.component_name", "Unknown")
	clusterName := l.GetStringOrDefault("resource.labels.cluster_name", "Unknown")
	msg, err := l.MainMessage()
	if err == nil {
		cs.RecordLogSummary(msg)
	}
	cs.RecordEvent(resourcepath.ControlplaneComponent(clusterName, component))
	return false, nil
}

// ShouldProcess implements ControlPlaneComponentParser.
func (d *DefaultK8sControlPlaneComponentParser) ShouldProcess(component_name string) bool {
	return true
}

var _ ControlPlaneComponentParser = (*DefaultK8sControlPlaneComponentParser)(nil)
