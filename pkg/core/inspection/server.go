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
	"sync"

	"golang.org/x/exp/slices"

	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	inspectioncore_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/impl"
)

// InspectionType describes a selectable log inspection mode and its display metadata.
type InspectionType struct {
	// Id uniquely identifies this inspection type.
	Id string `json:"id"`
	// Name is the human-readable display name of the inspection type.
	Name string `json:"name"`
	// Description explains what logs and features this inspection type supports.
	Description string `json:"description"`
	// Icon is the path or identifier of the icon displayed in the UI.
	Icon string `json:"icon"`
	// Priority determines the sort order of inspection types in descending order.
	Priority int `json:"-"`
	// Labels holds key-value labels matched against task LabelKeyInspectionTypeLabelSelector values.
	Labels map[string]string `json:"labels,omitempty"`

	// DocumentDescription provides an extended description used for generated documentation.
	DocumentDescription string `json:"-"`
}

// FeatureListItem describes a toggleable feature task presented in the UI feature list.
type FeatureListItem struct {
	// Id is the task implementation ID of the feature task.
	Id string `json:"id"`
	// Label is the display title shown for the feature toggle.
	Label string `json:"label"`
	// Description explains what the feature collects or analyzes.
	Description string `json:"description"`
	// Enabled reports whether the feature is currently enabled in the runner.
	Enabled bool `json:"enabled"`
	// Order is the criteria of sorting []FeatureListItem.
	Order int `json:"-"`
}

// InspectionDryRunResult holds the serializable metadata produced by a dry-run execution.
type InspectionDryRunResult struct {
	// Metadata contains the serialized dry-run metadata entries such as form fields and queries.
	Metadata interface{} `json:"metadata"`
}

// InspectionRunResult holds the serialized metadata and result store produced by a completed inspection run.
type InspectionRunResult struct {
	// Metadata contains the serialized run metadata entries.
	Metadata interface{}
	// ResultStore provides access to the serialized KHI inspection output data.
	ResultStore inspectioncore.Store
}

// InspectionTaskServer manages tasks and provides apis to get task related information in JSON convertible type.
type InspectionTaskServer struct {
	// RootTaskSet is the set of the all tasks in KHI.
	RootTaskSet *coretask.TaskSet
	// inspectionTypes are kinds of tasks. Users will select this at first to filter togglable feature tasks.
	inspectionTypes []*InspectionType
	// inspections are generated inspection task runers
	inspections           map[string]*InspectionTaskRunner
	inspectionsMu         sync.RWMutex
	inspectionIDGenerator idgenerator.IDGenerator

	ioConfig *inspectioncore.IOConfig

	inspectionNameRegistry inspectioncore.InspectionNameRegistry

	runContextOptions      []RunContextOption
	inspectionIntercepters []InspectionInterceptor
}

// NewServer creates and initializes an InspectionTaskServer with core inspection tasks.
func NewServer(ioConfig *inspectioncore.IOConfig) (*InspectionTaskServer, error) {
	ns, err := coretask.NewTaskSet([]coretask.UntypedTask{})
	if err != nil {
		return nil, err
	}
	server := &InspectionTaskServer{
		RootTaskSet:            ns,
		inspectionTypes:        make([]*InspectionType, 0),
		inspections:            map[string]*InspectionTaskRunner{},
		inspectionIDGenerator:  idgenerator.NewPrefixIDGenerator("inspection-"),
		ioConfig:               ioConfig,
		inspectionNameRegistry: inspectioncore.NewInMemoryInspectionNameRegistry(),
	}

	// Register mandatory tasks for inspection task
	err = server.AddModules(Module{Name: "inspectioncore", Tasks: inspectioncore_impl.Tasks})
	if err != nil {
		return nil, err
	}
	return server, nil
}

// AddInspectionType registers an inspection type with the server.
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

// AddTask registers a task usable for the inspection task graph execution.
func (s *InspectionTaskServer) AddTask(task coretask.UntypedTask) error {
	return s.RootTaskSet.Add(task)
}

// AddModules registers the inspection types and the scoped tasks of the given modules.
// It returns an error when a module scope conflicts with its parent scope, or when a task ID or an inspection type ID is duplicated.
func (s *InspectionTaskServer) AddModules(modules ...Module) error {
	for _, module := range modules {
		flattened, err := module.flatten()
		if err != nil {
			return err
		}
		for _, inspectionType := range flattened.inspectionTypes {
			if err := s.AddInspectionType(inspectionType); err != nil {
				return fmt.Errorf("module %s: %w", module.Name, err)
			}
		}
		for _, task := range flattened.tasks {
			if err := s.AddTask(task); err != nil {
				return fmt.Errorf("module %s: %w", module.Name, err)
			}
		}
	}
	return nil
}

// AddInspectionInterceptor adds an interceptor that will be applied to all new inspection runners.
func (s *InspectionTaskServer) AddInspectionInterceptor(interceptor InspectionInterceptor) {
	s.inspectionIntercepters = append(s.inspectionIntercepters, interceptor)
}

// CreateInspection generates an inspection and returns inspection ID.
func (s *InspectionTaskServer) CreateInspection(inspectionType string) (string, error) {
	id := s.inspectionIDGenerator.Generate()
	inspectionRunner := NewInspectionRunner(s, s.ioConfig, id, s.runContextOptions...)
	inspectionRunner.AddInterceptors(s.inspectionIntercepters...)
	err := inspectionRunner.SetInspectionType(inspectionType)
	if err != nil {
		return "", err
	}
	s.inspectionsMu.Lock()
	s.inspections[inspectionRunner.ID] = inspectionRunner
	s.inspectionsMu.Unlock()
	return inspectionRunner.ID, nil
}

// GetInspection returns an instance of an Inspection queried with given inspection ID.
func (s *InspectionTaskServer) GetInspection(inspectionID string) *InspectionTaskRunner {
	s.inspectionsMu.RLock()
	defer s.inspectionsMu.RUnlock()
	return s.inspections[inspectionID]
}

// GetAllInspectionTypes returns a copy of all inspection types registered on the server.
func (s *InspectionTaskServer) GetAllInspectionTypes() []*InspectionType {
	return append([]*InspectionType{}, s.inspectionTypes...)
}

// GetInspectionType returns the registered InspectionType matching inspectionTypeId, or nil if not found.
func (s *InspectionTaskServer) GetInspectionType(inspectionTypeId string) *InspectionType {
	for _, registeredType := range s.inspectionTypes {
		if registeredType.Id == inspectionTypeId {
			return registeredType
		}
	}
	return nil
}

// GetAllRunners returns a slice of all inspection runners currently registered on the server.
func (s *InspectionTaskServer) GetAllRunners() []*InspectionTaskRunner {
	s.inspectionsMu.RLock()
	defer s.inspectionsMu.RUnlock()
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

// IOConfig returns the IOConfig associated with the server.
func (s *InspectionTaskServer) IOConfig() *inspectioncore.IOConfig {
	return s.ioConfig
}

// InspectionNameRegistry returns the registry managing unique inspection names for this server.
func (s *InspectionTaskServer) InspectionNameRegistry() inspectioncore.InspectionNameRegistry {
	return s.inspectionNameRegistry
}

// RegisterImportedInspection registers a completed imported inspection with the given ID, store, and metadata.
func (s *InspectionTaskServer) RegisterImportedInspection(id string, store inspectioncore.Store, metadata *typedmap.ReadonlyTypedMap) *InspectionTaskRunner {
	if metadata != nil {
		if header, found := typedmap.Get(metadata, inspectionmetadata.HeaderMetadataKey); found && header != nil {
			uniqueName := s.inspectionNameRegistry.ReserveUniqueName(id, header.InspectionName)
			header.InspectionName = uniqueName
			header.SuggestedFileName = fmt.Sprintf("%s.khi", uniqueName)
		}
	}
	runner := NewImportedInspectionRunner(s, s.ioConfig, id, store, metadata, s.runContextOptions...)
	s.inspectionsMu.Lock()
	s.inspections[id] = runner
	s.inspectionsMu.Unlock()
	return runner
}
