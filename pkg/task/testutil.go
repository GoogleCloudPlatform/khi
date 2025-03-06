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

package task

func newLocalCachedTaskRunnerForSingleTask(target Definition, cache TaskVariableCache, dependencies ...Definition) (*LocalRunner, error) {
	sourceDs, err := NewSet([]Definition{target})
	if err != nil {
		return nil, err
	}
	availableDs, err := NewSet(dependencies)
	if err != nil {
		return nil, err
	}

	resolved, err := sourceDs.ResolveTask(availableDs)
	if err != nil {
		return nil, err
	}

	localRunner, err := NewLocalRunner(resolved)
	if err != nil {
		return nil, err
	}
	localRunner.WithCacheProvider(cache)
	return localRunner, nil
}

// HasDependency check if 2 tasks have dependency between them when the task graph was resolved with given task set.
func HasDependency(taskSet *DefinitionSet, dependencyFrom Definition, dependencyTo Definition) (bool, error) {
	sourceSet, err := NewSet([]Definition{dependencyFrom})
	if err != nil {
		return false, err
	}
	resolvedSet, err := sourceSet.ResolveTask(taskSet)
	if err != nil {
		return false, err
	}
	dependentDefinitions := resolvedSet.GetAll()
	for _, definition := range dependentDefinitions {
		if definition.ID() == dependencyTo.ID() {
			return true, nil
		}
	}
	return false, nil
}
