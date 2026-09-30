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
	"fmt"
	"maps"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// Scope is a set of inspection type label conditions.
// Tasks in a module are available only for inspection types whose labels match every condition.
type Scope map[string]string

// Module declares the inspection types and tasks that a package provides, and the scope where the tasks are available.
// A sub-module inherits the scope of its parent and can only add conditions to it, so it can narrow the scope but never widen it.
type Module struct {
	// Name identifies the module in error messages. Sub-module names are joined to the parent name with "/".
	Name string
	// Scope is the set of conditions added to the scope of the parent module.
	Scope Scope
	// InspectionTypes are the inspection types that the module defines. They are not affected by the scope.
	InspectionTypes []InspectionType
	// Tasks are the tasks available in the scope of the module.
	Tasks []coretask.UntypedTask
	// SubModules are the modules whose scopes are narrowed from the scope of this module.
	SubModules []Module
}

// flattenedModule holds the tasks and inspection types of a module tree with the scopes applied to the tasks.
type flattenedModule struct {
	tasks           []coretask.UntypedTask
	inspectionTypes []InspectionType
}

// flatten resolves the scopes of the module tree and returns its tasks labeled with the resolved scopes.
// It returns an error when a sub-module scope gives a different value to a label key that the parent scope already has.
func (m Module) flatten() (*flattenedModule, error) {
	result := &flattenedModule{}
	if err := m.flattenInto(result, "", Scope{}); err != nil {
		return nil, err
	}
	return result, nil
}

func (m Module) flattenInto(result *flattenedModule, parentPath string, parentScope Scope) error {
	path := m.Name
	if parentPath != "" {
		path = parentPath + "/" + m.Name
	}
	scope := maps.Clone(parentScope)
	for key, value := range m.Scope {
		if parentValue, found := parentScope[key]; found && parentValue != value {
			return fmt.Errorf("module %s: scope condition %s=%q conflicts with %s=%q in the parent scope", path, key, value, key, parentValue)
		}
		scope[key] = value
	}
	result.inspectionTypes = append(result.inspectionTypes, m.InspectionTypes...)
	for _, task := range m.Tasks {
		result.tasks = append(result.tasks, applyScope(task, scope))
	}
	for _, subModule := range m.SubModules {
		if err := subModule.flattenInto(result, path, scope); err != nil {
			return err
		}
	}
	return nil
}

// applyScope labels the task with the inspection type selector of the scope.
// A task in an empty scope stays available for all inspection types.
func applyScope(task coretask.UntypedTask, scope Scope) coretask.UntypedTask {
	if len(scope) == 0 {
		return task
	}
	return &wrappedTaskWithLabels{
		UntypedTask: task,
		opts:        []coretask.LabelOpt{inspectioncore.InspectionTypeLabelSelector(scope)},
	}
}
