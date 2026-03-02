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
	khifilev4 "github.com/GoogleCloudPlatform/khi/pkg/generated/proto/khifile/v4"
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

	severities     []*khifilev4.Severity
	verbs          []*khifilev4.Verb
	logTypes       []*khifilev4.LogType
	revisionStates []*khifilev4.RevisionState
	timelineTypes  []*khifilev4.TimelineType
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
		severities:            make([]*khifilev4.Severity, 0),
		verbs:                 make([]*khifilev4.Verb, 0),
		logTypes:              make([]*khifilev4.LogType, 0),
		revisionStates:        make([]*khifilev4.RevisionState, 0),
		timelineTypes:         make([]*khifilev4.TimelineType, 0),
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

// AddSeverity registers a Severity. The ID will be automatically assigned.
func (s *InspectionTaskServer) AddSeverity(severity *khifilev4.Severity) error {
	for _, existing := range s.severities {
		if existing == severity {
			return nil // Already registered
		}
	}
	if severity.Id != 0 {
		return fmt.Errorf("id must not be set when registering StyleData")
	}
	severity.Id = uint32(len(s.severities) + 1)
	s.severities = append(s.severities, severity)
	return nil
}

// AddVerb registers a Verb. The ID will be automatically assigned.
func (s *InspectionTaskServer) AddVerb(verb *khifilev4.Verb) error {
	for _, existing := range s.verbs {
		if existing == verb {
			return nil // Already registered
		}
	}
	if verb.Id != 0 {
		return fmt.Errorf("id must not be set when registering StyleData")
	}
	verb.Id = uint32(len(s.verbs) + 1)
	s.verbs = append(s.verbs, verb)
	return nil
}

// AddLogType registers a LogType. The ID will be automatically assigned.
func (s *InspectionTaskServer) AddLogType(logType *khifilev4.LogType) error {
	for _, existing := range s.logTypes {
		if existing == logType {
			return nil // Already registered
		}
	}
	if logType.Id != 0 {
		return fmt.Errorf("id must not be set when registering StyleData")
	}
	logType.Id = uint32(len(s.logTypes) + 1)
	s.logTypes = append(s.logTypes, logType)
	return nil
}

// AddRevisionState registers a RevisionState. The ID will be automatically assigned.
func (s *InspectionTaskServer) AddRevisionState(revisionState *khifilev4.RevisionState) error {
	for _, existing := range s.revisionStates {
		if existing == revisionState {
			return nil // Already registered
		}
	}
	if revisionState.Id != 0 {
		return fmt.Errorf("id must not be set when registering StyleData")
	}
	revisionState.Id = uint32(len(s.revisionStates) + 1)
	s.revisionStates = append(s.revisionStates, revisionState)
	return nil
}

// AddTimelineType registers a TimelineType. The ID will be automatically assigned.
func (s *InspectionTaskServer) AddTimelineType(timelineType *khifilev4.TimelineType) error {
	for _, existing := range s.timelineTypes {
		if existing == timelineType {
			return nil // Already registered
		}
	}
	if timelineType.Id != 0 {
		return fmt.Errorf("id must not be set when registering StyleData")
	}
	timelineType.Id = uint32(len(s.timelineTypes) + 1)
	s.timelineTypes = append(s.timelineTypes, timelineType)
	return nil
}

// GetStyleData returns the complete StyleData.
func (s *InspectionTaskServer) GetStyleData() *khifilev4.StyleData {
	return &khifilev4.StyleData{
		Severities:     append([]*khifilev4.Severity{}, s.severities...),
		Verbs:          append([]*khifilev4.Verb{}, s.verbs...),
		LogTypes:       append([]*khifilev4.LogType{}, s.logTypes...),
		RevisionStates: append([]*khifilev4.RevisionState{}, s.revisionStates...),
		TimelineTypes:  append([]*khifilev4.TimelineType{}, s.timelineTypes...),
	}
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

// Dispose resets the IDs of all registered items for another server to register them again.
// This is needed for testing purpose.
func (s *InspectionTaskServer) Dispose() {
	for _, severity := range s.severities {
		severity.Id = 0
	}
	for _, verb := range s.verbs {
		verb.Id = 0
	}
	for _, logType := range s.logTypes {
		logType.Id = 0
	}
	for _, revisionState := range s.revisionStates {
		revisionState.Id = 0
	}
	for _, timelineType := range s.timelineTypes {
		timelineType.Id = 0
	}
}

var _ InspectionTaskRegistry = (*InspectionTaskServer)(nil)
