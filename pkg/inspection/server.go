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

package inspection

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/inspectiondata"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type PrepareInspectionServerFunc = func(inspectionServer *InspectionTaskServer) error

type InspectionType struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Priority    int    `json:"-"`

	// Document properties
	DocumentDescription string `json:"-"`
}

type FeatureListItem struct {
	Id          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type InspectionDryRunResult struct {
	Metadata interface{} `json:"metadata"`
}

type InspectionRunResult struct {
	Metadata    interface{}
	ResultStore inspectiondata.Store
}

// InspectionTaskServer manages tasks and provides apis to get task related information in JSON convertible type.
type InspectionTaskServer struct {
	// RootTaskSet is the set of the all definitions in KHI.
	RootTaskSet *task.DefinitionSet
	// inspectionTypes are kinds of tasks. Users will select this at first to filter togglable feature tasks.
	inspectionTypes []*InspectionType
	// tasks are generated tasks
	tasks map[string]*InspectionRunner
}

func NewServer() (*InspectionTaskServer, error) {
	ns, err := task.NewSet([]task.Definition{})
	if err != nil {
		return nil, err
	}
	return &InspectionTaskServer{
		RootTaskSet:     ns,
		inspectionTypes: make([]*InspectionType, 0),
		tasks:           map[string]*InspectionRunner{},
	}, nil
}

// AddInspectionType register a inspection type.
func (s *InspectionTaskServer) AddInspectionType(newInspectionType InspectionType) error {
	if strings.Contains(newInspectionType.Id, "/") {
		return fmt.Errorf("inspection type must not contain /")
	}
	idMap := map[string]interface{}{}
	for _, inspectionType := range s.inspectionTypes {
		idMap[inspectionType.Id] = struct{}{}
	}
	if _, exist := idMap[newInspectionType.Id]; exist {
		return fmt.Errorf("inspection type id:%s is duplicated. InspectionType ID must be unique", newInspectionType.Id)
	}
	inspectionTypesCandidate := append(s.inspectionTypes, &newInspectionType)
	slices.SortFunc(inspectionTypesCandidate, func(a *InspectionType, b *InspectionType) int {
		return b.Priority - a.Priority
	})
	s.inspectionTypes = inspectionTypesCandidate
	return nil
}

// AddTaskDefinition register a task definition usable for the inspection tasks
func (s *InspectionTaskServer) AddTaskDefinition(taskDefinition task.Definition) error {
	return s.RootTaskSet.Add(taskDefinition)
}

// CreateInspection generates an inspection and returns inspection ID
func (s *InspectionTaskServer) CreateInspection(inspectionType string) (string, error) {
	inspectionTask := NewInspectionRunner(s)
	err := inspectionTask.SetInspectionType(inspectionType)
	if err != nil {
		return "", err
	}
	s.tasks[inspectionTask.ID] = inspectionTask
	return inspectionTask.ID, nil
}

// Inspection returns an instance of an Inspection queried with given inspection ID.
func (s *InspectionTaskServer) GetTask(taskId string) *InspectionRunner {
	return s.tasks[taskId]
}

func (s *InspectionTaskServer) GetAllInspectionTypes() []*InspectionType {
	return append([]*InspectionType{}, s.inspectionTypes...)
}

func (s *InspectionTaskServer) GetInspectionType(inspectionTypeId string) *InspectionType {
	for _, registeredType := range s.inspectionTypes {
		if registeredType.Id == inspectionTypeId {
			return registeredType
		}
	}
	return nil
}

func (s *InspectionTaskServer) GetAllRunners() []*InspectionRunner {
	inspections := []*InspectionRunner{}
	for _, value := range s.tasks {
		inspections = append(inspections, value)
	}
	return inspections
}

// GetAllRegisteredTasks returns a cloned list of all definitions registered in this server.
func (s *InspectionTaskServer) GetAllRegisteredTasks() []task.Definition {
	return s.RootTaskSet.GetAll()
}
