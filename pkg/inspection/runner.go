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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	inspection_task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/inspection/contextkey"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/inspectiondata"
	inspection_task_interface "github.com/GoogleCloudPlatform/khi/pkg/inspection/interface"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata"
	error_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/error"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/plan"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/query"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/serializer"
	"github.com/GoogleCloudPlatform/khi/pkg/lifecycle"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	task_contextkey "github.com/GoogleCloudPlatform/khi/pkg/task/contextkey"
	task_interface "github.com/GoogleCloudPlatform/khi/pkg/task/inteface"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

var inspectionRunnerGlobalSharedMap = typedmap.NewTypedMap()

type InspectionRunner struct {
	inspectionServer      *InspectionTaskServer
	ID                    string
	enabledFeatures       map[string]bool
	availableDefinitions  *task.DefinitionSet
	featuresDefinitions   *task.DefinitionSet
	requiredDefinitions   *task.DefinitionSet
	runner                task_interface.TaskRunner
	runnerLock            sync.Mutex
	metadata              *typedmap.ReadonlyTypedMap
	cancel                context.CancelFunc
	inspectionSharedMap   *typedmap.TypedMap
	currentInspectionType string
}

func NewInspectionRunner(server *InspectionTaskServer) *InspectionRunner {
	return &InspectionRunner{
		inspectionServer:      server,
		ID:                    generateRandomString(),
		enabledFeatures:       map[string]bool{},
		availableDefinitions:  nil,
		featuresDefinitions:   nil,
		requiredDefinitions:   nil,
		runner:                nil,
		runnerLock:            sync.Mutex{},
		metadata:              nil,
		inspectionSharedMap:   typedmap.NewTypedMap(),
		cancel:                nil,
		currentInspectionType: "N/A",
	}
}

func (i *InspectionRunner) Started() bool {
	return i.runner != nil
}

func (i *InspectionRunner) SetInspectionType(inspectionType string) error {
	typeFound := false
	for _, inspection := range i.inspectionServer.inspectionTypes {
		if inspection.Id == inspectionType {
			typeFound = true
			break
		}
	}
	if !typeFound {
		return fmt.Errorf("inspection type %s was not found", inspectionType)
	}
	i.availableDefinitions = task.Subset(i.inspectionServer.RootTaskSet, filter.NewContainsElementFilter(inspection_task.LabelKeyInspectionTypes, inspectionType, true))
	defaultFeatures := task.Subset(i.availableDefinitions, filter.NewEnabledFilter(inspection_task.LabelKeyInspectionDefaultFeatureFlag, false))
	i.requiredDefinitions = task.Subset(i.availableDefinitions, filter.NewEnabledFilter(inspection_task.LabelKeyInspectionRequiredFlag, false))
	defaultFeatureIds := []string{}
	for _, featureTask := range defaultFeatures.GetAll() {
		defaultFeatureIds = append(defaultFeatureIds, featureTask.UntypedID().String())
	}
	i.currentInspectionType = inspectionType
	return i.SetFeatureList(defaultFeatureIds)
}

func (i *InspectionRunner) FeatureList() ([]FeatureListItem, error) {
	if i.availableDefinitions == nil {
		return nil, fmt.Errorf("inspection type is not yet initialized")
	}
	featureSet := task.Subset(i.availableDefinitions, filter.NewEnabledFilter(inspection_task.LabelKeyInspectionFeatureFlag, false))
	features := []FeatureListItem{}
	for _, definition := range featureSet.GetAll() {
		label := typedmap.GetOrDefault(definition.Labels(), inspection_task.LabelKeyFeatureTaskTitle, fmt.Sprintf("No label Set!(%s)", definition.UntypedID()))
		description := typedmap.GetOrDefault(definition.Labels(), inspection_task.LabelKeyFeatureTaskDescription, "")
		enabled := false
		if v, exist := i.enabledFeatures[definition.UntypedID().String()]; exist && v {
			enabled = true
		}
		features = append(features, FeatureListItem{
			Id:          definition.UntypedID().String(),
			Label:       label,
			Description: description,
			Enabled:     enabled,
		})
	}
	return features, nil
}

func (i *InspectionRunner) SetFeatureList(featureList []string) error {
	featureDefinitions := []task.UntypedDefinition{}
	for _, featureId := range featureList {
		definition, err := i.availableDefinitions.Get(featureId)
		if err != nil {
			return err
		}
		if !typedmap.GetOrDefault(definition.Labels(), inspection_task.LabelKeyInspectionFeatureFlag, false) {
			return fmt.Errorf("task `%s` is not marked as a feature but requested to be included in the feature set of an inspection", definition.UntypedID())
		}
		featureDefinitions = append(featureDefinitions, definition)
	}
	featureDefinitionSet, err := task.NewSet(featureDefinitions)
	if err != nil {
		return err
	}
	i.enabledFeatures = map[string]bool{}
	for _, feature := range featureList {
		i.enabledFeatures[feature] = true
	}
	i.featuresDefinitions = featureDefinitionSet
	return nil
}

// withRunContextValues returns a context with the value specific to a single run of task.
func (i *InspectionRunner) withRunContextValues(ctx context.Context, runMode inspection_task_interface.InspectionTaskMode, taskInput map[string]any) context.Context {
	rid := generateRandomString()
	runCtx := khictx.WithValue(ctx, inspection_task_contextkey.InspectionTaskRunID, rid)
	runCtx = khictx.WithValue(runCtx, inspection_task_contextkey.InspectionTaskInspectionID, i.ID)
	runCtx = khictx.WithValue(runCtx, inspection_task_contextkey.InspectionSharedMap, i.inspectionSharedMap)
	runCtx = khictx.WithValue(runCtx, inspection_task_contextkey.GlobalSharedMap, inspectionRunnerGlobalSharedMap)
	runCtx = khictx.WithValue(runCtx, inspection_task_contextkey.InspectionTaskInput, taskInput)
	return khictx.WithValue(runCtx, inspection_task_contextkey.InspectionTaskMode, runMode)
}

func (i *InspectionRunner) Run(ctx context.Context, req *inspection_task.InspectionRequest) error {
	defer i.runnerLock.Unlock()
	i.runnerLock.Lock()
	if i.runner != nil {
		return fmt.Errorf("this task is already started")
	}
	currentInspectionType := i.inspectionServer.GetInspectionType(i.currentInspectionType)
	runnableTaskGraph, err := i.resolveTaskGraph()
	if err != nil {
		return err
	}

	runCtx := i.withRunContextValues(ctx, inspection_task_interface.TaskModeRun, req.Values)

	runMetadata := i.generateMetadataForRun(runCtx, &header.Header{
		InspectTimeUnixSeconds: time.Now().Unix(),
		InspectionType:         currentInspectionType.Name,
		InspectionTypeIconPath: currentInspectionType.Icon,
	}, runnableTaskGraph)

	runCtx = khictx.WithValue(runCtx, inspection_task_contextkey.InspectionRunMetadata, runMetadata)

	cancelableCtx, cancel := context.WithCancel(runCtx)
	i.cancel = cancel

	runner, err := task.NewLocalRunner(runnableTaskGraph)
	if err != nil {
		return err
	}
	i.runner = runner

	i.metadata = runMetadata
	lifecycle.Default.NotifyInspectionStart(khictx.MustGetValue(runCtx, inspection_task_contextkey.InspectionTaskRunID), currentInspectionType.Name)

	err = i.runner.Run(cancelableCtx)
	if err != nil {
		return err
	}
	go func() {
		<-i.runner.Wait()
		progress, found := typedmap.Get(i.metadata, progress.ProgressMetadataKey)
		if !found {
			slog.ErrorContext(runCtx, "progress metadata was not found")
		}
		status := ""
		resultSize := 0
		if result, err := i.runner.Result(); err != nil {
			if errors.Is(cancelableCtx.Err(), context.Canceled) {
				progress.Cancel()
				status = "cancel"
			} else {
				progress.Error()
				status = "error"
			}
			slog.WarnContext(runCtx, fmt.Sprintf("task %s was finished with an error\n%s", i.ID, err))
		} else {
			progress.Done()
			status = "done"

			history, found := typedmap.Get(result, typedmap.NewTypedKey[inspectiondata.Store](serializer.SerializerTaskID.ReferenceIDString()))
			if !found {
				slog.ErrorContext(runCtx, fmt.Sprintf("Failed to get generated history after the completion\n%s", err))
			}
			if history == nil {
				slog.ErrorContext(runCtx, "Failed to get the serializer result. Result is nil!")
			} else {
				resultSize, err = history.GetInspectionResultSizeInBytes()
				if err != nil {
					slog.ErrorContext(runCtx, fmt.Sprintf("Failed to get the serialized result size\n%s", err))
				}
			}
		}
		lifecycle.Default.NotifyInspectionEnd(khictx.MustGetValue(runCtx, inspection_task_contextkey.InspectionTaskRunID), currentInspectionType.Name, status, resultSize)
	}()
	return nil
}

func (i *InspectionRunner) Result() (*InspectionRunResult, error) {
	if i.runner == nil {
		return nil, fmt.Errorf("this task is not yet started")
	}

	v, err := i.runner.Result()
	if err != nil {
		return nil, err
	}

	inspectionDataStore, found := typedmap.Get(v, typedmap.NewTypedKey[inspectiondata.Store](serializer.SerializerTaskID.ReferenceIDString()))
	if !found {
		return nil, fmt.Errorf("failed to get the serializer result")
	}

	md, err := metadata.GetSerializableSubsetMapFromMetadataSet(i.metadata, filter.NewEnabledFilter(metadata.LabelKeyIncludedInRunResultFlag, false))
	if err != nil {
		return nil, err
	}
	return &InspectionRunResult{
		Metadata:    md,
		ResultStore: inspectionDataStore,
	}, nil
}

func (i *InspectionRunner) Metadata() (map[string]any, error) {
	if i.runner == nil {
		return nil, fmt.Errorf("this task is not yet started")
	}
	md, err := metadata.GetSerializableSubsetMapFromMetadataSet(i.metadata, filter.NewEnabledFilter(metadata.LabelKeyIncludedInRunResultFlag, false))
	if err != nil {
		return nil, err
	}
	return md, nil
}

func (i *InspectionRunner) DryRun(ctx context.Context, req *inspection_task.InspectionRequest) (*InspectionDryRunResult, error) {
	runnableTaskGraph, err := i.resolveTaskGraph()
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return nil, err
	}

	runner, err := task.NewLocalRunner(runnableTaskGraph)
	if err != nil {
		return nil, err
	}

	runCtx := i.withRunContextValues(ctx, inspection_task_interface.TaskModeDryRun, req.Values)

	dryrunMetadata := i.generateMetadataForDryRun(runCtx, &header.Header{}, runnableTaskGraph)

	runCtx = khictx.WithValue(runCtx, inspection_task_contextkey.InspectionRunMetadata, dryrunMetadata)

	err = runner.Run(runCtx)
	if err != nil {
		return nil, err
	}
	<-runner.Wait()
	_, err = runner.Result()
	if err != nil {
		slog.ErrorContext(runCtx, err.Error())
		return nil, err
	}
	md, err := metadata.GetSerializableSubsetMapFromMetadataSet(dryrunMetadata, filter.NewEnabledFilter(metadata.LabelKeyIncludedInDryRunResultFlag, false))
	if err != nil {
		return nil, err
	}
	return &InspectionDryRunResult{
		Metadata: md,
	}, nil
}

func (i *InspectionRunner) MakeLoggers(ctx context.Context, minLevel slog.Level, m *typedmap.ReadonlyTypedMap, definitions []task.UntypedDefinition) *logger.Logger {
	logger := logger.NewLogger()
	for _, def := range definitions {
		taskCtx := khictx.WithValue(ctx, task_contextkey.TaskImplementationIDContextKey, def.UntypedID())
		logger.MakeTaskLogger(taskCtx, minLevel)
	}
	return logger
}
func (i *InspectionRunner) GetCurrentMetadata() (*typedmap.ReadonlyTypedMap, error) {
	if i.metadata == nil {
		return nil, fmt.Errorf("this task hasn't been started")
	}
	return i.metadata, nil
}

func (i *InspectionRunner) Cancel() error {
	if i.cancel == nil {
		return fmt.Errorf("this task is not yet started")
	}
	if _, err := i.Result(); err == nil {
		return fmt.Errorf("task %s is already finished", i.ID)
	}
	i.cancel()
	return nil
}

func (i *InspectionRunner) Wait() <-chan interface{} {
	return i.runner.Wait()
}

func (i *InspectionRunner) resolveTaskGraph() (*task.DefinitionSet, error) {
	if i.featuresDefinitions == nil || i.availableDefinitions == nil {
		return nil, fmt.Errorf("this runner is not ready for resolving graph")
	}
	usedTaskDefinitions := []task.UntypedDefinition{}
	usedTaskDefinitions = append(usedTaskDefinitions, i.featuresDefinitions.GetAll()...)
	usedTaskDefinitions = append(usedTaskDefinitions, i.requiredDefinitions.GetAll()...)
	initialTaskSet, err := task.NewSet(usedTaskDefinitions)
	if err != nil {
		return nil, err
	}
	set, err := initialTaskSet.ResolveTask(i.availableDefinitions)
	if err != nil {
		return nil, err
	}

	wrapped, err := set.WrapGraph(taskid.NewDefaultImplementationID[any](inspection_task.InspectionMainSubgraphName), []taskid.UntypedTaskReference{})
	if err != nil {
		return nil, err
	}

	// Add required pre process or post process for the subgraph
	err = wrapped.Add(serializer.SerializeTask)
	if err != nil {
		return nil, err
	}

	return wrapped.ResolveTask(i.availableDefinitions)
}

func (i *InspectionRunner) generateMetadataForDryRun(ctx context.Context, initHeader *header.Header, taskGraph *task.DefinitionSet) *typedmap.ReadonlyTypedMap {
	writableMetadata := typedmap.NewTypedMap()
	i.addCommonMetadata(ctx, writableMetadata, initHeader, taskGraph)
	return writableMetadata.AsReadonly()
}

func (i *InspectionRunner) generateMetadataForRun(ctx context.Context, initHeader *header.Header, taskGraph *task.DefinitionSet) *typedmap.ReadonlyTypedMap {
	writableMetadata := typedmap.NewTypedMap()
	i.addCommonMetadata(ctx, writableMetadata, initHeader, taskGraph)
	return writableMetadata.AsReadonly()
}

func (i *InspectionRunner) addCommonMetadata(ctx context.Context, writableMetadata *typedmap.TypedMap, initHeader *header.Header, taskGraph *task.DefinitionSet) {
	typedmap.Set(writableMetadata, header.HeaderMetadataKey, initHeader)
	typedmap.Set(writableMetadata, error_metadata.ErrorMessageSetMetadataKey, error_metadata.NewErrorMessageSet())
	typedmap.Set(writableMetadata, form.FormFieldSetMetadataKey, form.NewFormFieldSet())
	typedmap.Set(writableMetadata, query.QueryMetadataKey, query.NewQueryMetadata())

	progressMeta := progress.NewProgress()
	progressMeta.SetTotalTaskCount(len(task.Subset(taskGraph, filter.NewEnabledFilter(inspection_task.LabelKeyProgressReportable, false)).GetAll()))
	typedmap.Set(writableMetadata, progress.ProgressMetadataKey, progressMeta)

	taskGraphStr, err := taskGraph.DumpGraphviz()
	if err != nil {
		taskGraphStr = fmt.Sprintf("failed to generate task graph %v", err.Error())
	}
	typedmap.Set(writableMetadata, plan.InspectionPlanMetadataKey, plan.NewInspectionPlan(taskGraphStr))

	i.MakeLoggers(ctx, getLogLevel(), writableMetadata.AsReadonly(), taskGraph.GetAll())
}

func generateRandomString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomid := make([]rune, 16)
	for i := range randomid {
		randomid[i] = letters[rand.Intn(len(letters))]
	}
	return string(randomid)
}

func getLogLevel() slog.Level {
	if parameters.Debug.Verbose != nil && *parameters.Debug.Verbose {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
