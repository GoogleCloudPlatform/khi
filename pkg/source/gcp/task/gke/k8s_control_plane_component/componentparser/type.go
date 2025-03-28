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
)

// ControlPlaneComponentParser is an abstraction type to define the customized process for each components
type ControlPlaneComponentParser interface {
	// ShouldProcess return if the component must be processed by this parser or not
	ShouldProcess(component_name string) bool
	// Process handle the given logs to ingest to the ChangeSet. This method return false if the logs shouldn't be processed in the later parsers.
	Process(ctx context.Context, l *log.LogEntity, cs *history.ChangeSet, builder *history.Builder) (bool, error)
}

var ComponentParsers []ControlPlaneComponentParser = []ControlPlaneComponentParser{
	&ControllerManagerComponentParser{},
	&SchedulerComponentParser{},
	&DefaultK8sControlPlaneComponentParser{},
}
