// Copyright 2026 Google LLC
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

package coreinspection

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
)

// scopedTaskRegistry is a wrapper of InspectionTaskRegistry that adds shared labels to all registered tasks.
type scopedTaskRegistry struct {
	InspectionTaskRegistry
	opts []coretask.LabelOpt
}

func (s *scopedTaskRegistry) AddTask(task coretask.UntypedTask) error {
	wrapped := &wrappedTaskWithLabels{
		UntypedTask: task,
		opts:        s.opts,
	}
	return s.InspectionTaskRegistry.AddTask(wrapped)
}

// NewScopedRegistry Creates a new ScopedTaskRegistry with the given LabelOpts.
func NewScopedRegistry(reg InspectionTaskRegistry, opts ...coretask.LabelOpt) InspectionTaskRegistry {
	return &scopedTaskRegistry{
		InspectionTaskRegistry: reg,
		opts:                   opts,
	}
}

type wrappedTaskWithLabels struct {
	coretask.UntypedTask
	opts []coretask.LabelOpt
}

// Labels returns the merged labels of the base task and additional options.
func (w *wrappedTaskWithLabels) Labels() *typedmap.ReadonlyTypedMap {
	baseLabels := w.UntypedTask.Labels()

	optMap := typedmap.NewTypedMap()
	for _, opt := range w.opts {
		opt.Write(optMap)
	}

	return typedmap.Merge(baseLabels, optMap)
}

func (w *wrappedTaskWithLabels) UntypedRun(ctx context.Context) (any, error) {
	return w.UntypedTask.UntypedRun(ctx)
}

var _ coretask.UntypedTask = (*wrappedTaskWithLabels)(nil)
