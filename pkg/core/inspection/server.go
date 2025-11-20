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

package coreinspection

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	inspectioncore_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/impl"
	"golang.org/x/exp/slices"
)

type InspectionRegistrationFunc = func(registry InspectionTaskRegistry) error

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
	// Order is the criteria of sorting []FeatureListItem.
	Order int `json:"-"`
}

type InspectionDryRunResult struct {
	Metadata interface{} `json:"metadata"`
}

type InspectionRunResult struct {
	Metadata    interface{}
	ResultStore inspectioncore_contract.Store
}

// InspectionTaskServer manages tasks and provides apis to get task related information in JSON convertible type.
type InspectionTaskServer struct {
	// RootTaskSet is the set of the all tasks in KHI.
	RootTaskSet *coretask.TaskSet
	// inspectionTypes are kinds of tasks. Users will select this at first to filter togglable feature tasks.
	inspectionTypes []*InspectionType
	// inspections are generated inspection task runers
	inspections           map[string]*InspectionTaskRunner
	inspectionIDGenerator idgenerator.IDGenerator

	ioConfig *inspectioncore_contract.IOConfig

	runContextOptions      []RunContextOption
	inspectionIntercepters []InspectionInterceptor
}

func NewServer(ioConfig *inspectioncore_contract.IOConfig) (*InspectionTaskServer, error) {
	ns, err := coretask.NewTaskSet([]coretask.UntypedTask{})
	if err != nil {
		return nil, err
	}
	server := &InspectionTaskServer{
		RootTaskSet:           ns,
		inspectionTypes:       make([]*InspectionType, 0),
		inspections:           map[string]*InspectionTaskRunner{},
		inspectionIDGenerator: idgenerator.NewPrefixIDGenerator("inspection-"),
		ioConfig:              ioConfig,
	}

	// Register mandatory tasks for inspection task
	err = inspectioncore_impl.Register(server)
	if err != nil {
		return nil, err
	}
	return server, nil
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
	s.inspectionTypes = append(s.inspectionTypes, &newInspectionType)
	slices.SortFunc(s.inspectionTypes, func(a *InspectionType, b *InspectionType) int {
		return b.Priority - a.Priority
	})
	return nil
}

// AddTask register a task usable for the inspection task graph execution.
func (s *InspectionTaskServer) AddTask(task coretask.UntypedTask) error {
	return s.RootTaskSet.Add(task)
}

// AddInspectionInterceptor adds an interceptor that will be applied to all new inspection runners.
func (s *InspectionTaskServer) AddInspectionInterceptor(interceptor InspectionInterceptor) {
	s.inspectionIntercepters = append(s.inspectionIntercepters, interceptor)
}

// CreateInspection generates an inspection and returns inspection ID
func (s *InspectionTaskServer) CreateInspection(inspectionType string) (string, error) {
	id := s.inspectionIDGenerator.Generate()
	inspectionRunner := NewInspectionRunner(s, s.ioConfig, id, s.runContextOptions...)
	inspectionRunner.AddInterceptors(s.inspectionIntercepters...)
	err := inspectionRunner.SetInspectionType(inspectionType)
	if err != nil {
		return "", err
	}
	s.inspections[inspectionRunner.ID] = inspectionRunner
	return inspectionRunner.ID, nil
}

// Inspection returns an instance of an Inspection queried with given inspection ID.
func (s *InspectionTaskServer) GetInspection(inspectionID string) *InspectionTaskRunner {
	return s.inspections[inspectionID]
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

func (s *InspectionTaskServer) GetAllRunners() []*InspectionTaskRunner {
	inspections := []*InspectionTaskRunner{}
	for _, value := range s.inspections {
		inspections = append(inspections, value)
	}
	return inspections
}

// GetAllRegisteredTasks returns a cloned list of all tasks registered in this server.
func (s *InspectionTaskServer) GetAllRegisteredTasks() []coretask.UntypedTask {
	return s.RootTaskSet.GetAll()
}

// AddRunContextOption adds a RunContextOption that will be applied to all new inspection runners.
func (s *InspectionTaskServer) AddRunContextOption(option RunContextOption) {
	s.runContextOptions = append(s.runContextOptions, option)
}

var _ InspectionTaskRegistry = (*InspectionTaskServer)(nil)
