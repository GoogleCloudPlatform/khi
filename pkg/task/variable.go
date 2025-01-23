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

import (
	"context"
	"fmt"
	"sync"
)

// localBlockableVariable is a wrapper of a variable to wait the variable to be set by dependent task unit.
type localBlockableVariable struct {
	id       string
	value    any
	waiter   chan interface{}
	resolved bool
}

func newTaskVariable(id string) *localBlockableVariable {
	result := &localBlockableVariable{
		id:       id,
		value:    nil,
		waiter:   make(chan interface{}),
		resolved: false,
	}
	return result
}

// Set releases the lock of the variable to read and store the value.
// If the variable has already been set, an error is returned.
func (v *localBlockableVariable) Set(value any) error {
	if v.resolved {
		return fmt.Errorf("task variable `%s` set twice time", v.id)
	}
	v.resolved = true
	v.value = value
	close(v.waiter)
	return nil
}

// Wait waits the variable to be set and read it.
// It will wait the value to be set when the variable wasn't set yet.
func (v *localBlockableVariable) Wait(ctx context.Context) (any, error) {
	select {
	case <-v.waiter:
		return v.value, nil
	case <-ctx.Done():
		return nil, context.Canceled
	}
}

func (v *localBlockableVariable) Get() (any, error) {
	if v.resolved {
		return v.value, nil
	} else {
		return nil, fmt.Errorf("variable `%s` is not yet resolved", v.id)
	}
}

func (v *localBlockableVariable) IsResolved() bool {
	return v.resolved
}

// VariableSet contain the list of variables get/set by tasks.
type VariableSet struct {
	variables sync.Map
}

func NewVariableSet(initialVariables map[string]any) *VariableSet {
	vs := &VariableSet{
		variables: sync.Map{},
	}
	for variableKey, data := range initialVariables {
		vs.Set(variableKey, data)
	}
	return vs
}

// Get returns a value or waits the variables to be assigned from the other task.
func (s *VariableSet) Wait(ctx context.Context, key string) (any, error) {
	return s.getVariable(key).Wait(ctx)
}

// Get returns a value or waits the variables to be assigned from the other task.
func (s *VariableSet) Get(key string) (any, error) {
	return s.getVariable(key).Get()
}

func (s *VariableSet) Set(key string, value any) error {
	return s.getVariable(key).Set(value)
}

func (s *VariableSet) DeleteItems(selector func(key string) bool) {
	keys := map[string]struct{}{}
	s.variables.Range(func(key, value any) bool {
		keyString := key.(string)
		if selector(keyString) {
			keys[keyString] = struct{}{}
		}
		return true
	})
	for k := range keys {
		s.variables.Delete(k)
	}
}

func (s *VariableSet) IsResolved(key string) bool {
	return s.getVariable(key).IsResolved()
}

func (s *VariableSet) getVariable(key string) *localBlockableVariable {
	variable, _ := s.variables.LoadOrStore(key, newTaskVariable(key))
	return variable.(*localBlockableVariable)
}

// GetTypedVariableFromTaskVariable returns the specified variable from given variable set with type cast.
func GetTypedVariableFromTaskVariable[T any](tv *VariableSet, variableId string, defaultValue T) (T, error) {
	valueAny, err := tv.Get(variableId)
	if err != nil {
		return defaultValue, err
	}
	if value, convertible := valueAny.(T); convertible {
		return value, nil
	} else {
		return defaultValue, fmt.Errorf("the given value %v in %s couldn't be converted to %T", valueAny, variableId, *new(T))
	}
}
