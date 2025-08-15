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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/lifecycle"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
	inspection_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/impl"
)

var inspectionRunnerGlobalSharedMap = typedmap.NewTypedMap()

// InspectionTaskRunner manages the lifecycle of a single inspection instance.
// It handles task graph resolution, execution, and result retrieval for a given inspection type and feature set.
type InspectionTaskRunner struct {
	inspectionServer      *InspectionTaskServer
	ID                    string
	runIDGenerator        idgenerator.IDGenerator
	enabledFeatures       map[string]bool
	availableTasks        *coretask.TaskSet
	featureTasks          *coretask.TaskSet
	requiredTasks         *coretask.TaskSet
	runner                coretask.TaskRunner
	runnerLock            sync.Mutex
	metadata              *typedmap.ReadonlyTypedMap
	cancel                context.CancelFunc
	inspectionSharedMap   *typedmap.TypedMap
	currentInspectionType string
	ioconfig              *inspection_contract.IOConfig
}

// NewInspectionRunner creates a new InspectionTaskRunner.
func NewInspectionRunner(server *InspectionTaskServer, ioConfig *inspection_contract.IOConfig, id string) *InspectionTaskRunner {
	return &InspectionTaskRunner{
		inspectionServer:      server,
		ID:                    id,
		runIDGenerator:        idgenerator.NewPrefixIDGenerator("run-"),
		enabledFeatures:       map[string]bool{},
		availableTasks:        nil,
		featureTasks:          nil,
		requiredTasks:         nil,
		runner:                nil,
		runnerLock:            sync.Mutex{},
		metadata:              nil,
		inspectionSharedMap:   typedmap.NewTypedMap(),
		cancel:                nil,
		currentInspectionType: "N/A",
		ioconfig:              ioConfig,
	}
}

// Started returns true if the inspection has been started.
func (i *InspectionTaskRunner) Started() bool {
	return i.runner != nil
}

// SetInspectionType sets the type of inspection and initializes the available tasks.
// It filters the root task set from the server to get tasks relevant to the specified inspectionType.
func (i *InspectionTaskRunner) SetInspectionType(inspectionType string) error {
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
	i.availableTasks = coretask.Subset(i.inspectionServer.RootTaskSet, filter.NewContainsElementFilter(inspection_contract.LabelKeyInspectionTypes, inspectionType, true))
	defaultFeatures := coretask.Subset(i.availableTasks, filter.NewEnabledFilter(inspection_contract.LabelKeyInspectionDefaultFeatureFlag, false))
	i.requiredTasks = coretask.Subset(i.availableTasks, filter.NewEnabledFilter(inspection_contract.LabelKeyInspectionRequiredFlag, false))
	defaultFeatureIds := []string{}
	for _, featureTask := range defaultFeatures.GetAll() {
		defaultFeatureIds = append(defaultFeatureIds, featureTask.UntypedID().String())
	}
	i.currentInspectionType = inspectionType
	return i.SetFeatureList(defaultFeatureIds)
}

// FeatureList returns the list of available features for the current inspection type.
func (i *InspectionTaskRunner) FeatureList() ([]FeatureListItem, error) {
	if i.availableTasks == nil {
		return nil, fmt.Errorf("inspection type is not yet initialized")
	}
	featureSet := coretask.Subset(i.availableTasks, filter.NewEnabledFilter(inspection_contract.LabelKeyInspectionFeatureFlag, false))
	features := []FeatureListItem{}
	for _, featureTask := range featureSet.GetAll() {
		label := typedmap.GetOrDefault(featureTask.Labels(), inspection_contract.LabelKeyFeatureTaskTitle, fmt.Sprintf("No label Set!(%s)", featureTask.UntypedID()))
		description := typedmap.GetOrDefault(featureTask.Labels(), inspection_contract.LabelKeyFeatureTaskDescription, "")
		enabled := false
		if v, exist := i.enabledFeatures[featureTask.UntypedID().String()]; exist && v {
			enabled = true
		}
		features = append(features, FeatureListItem{
			Id:          featureTask.UntypedID().String(),
			Label:       label,
			Description: description,
			Enabled:     enabled,
		})
	}
	return features, nil
}

// SetFeatureList sets the list of enabled features for the inspection.
func (i *InspectionTaskRunner) SetFeatureList(featureList []string) error {
	featureTasks := []coretask.UntypedTask{}
	for _, featureId := range featureList {
		featureTask, err := i.availableTasks.Get(featureId)
		if err != nil {
			return err
		}
		if !typedmap.GetOrDefault(featureTask.Labels(), inspection_contract.LabelKeyInspectionFeatureFlag, false) {
			return fmt.Errorf("task `%s` is not marked as a feature but requested to be included in the feature set of an inspection", featureTask.UntypedID())
		}
		featureTasks = append(featureTasks, featureTask)
	}
	featureTaskSet, err := coretask.NewTaskSet(featureTasks)
	if err != nil {
		return err
	}
	i.enabledFeatures = map[string]bool{}
	for _, feature := range featureList {
		i.enabledFeatures[feature] = true
	}
	i.featureTasks = featureTaskSet
	return nil
}

// UpdateFeatureMap updates the enabled features based on the provided map.
// The input map contains feature IDs and a boolean indicating if they should be enabled.
func (i *InspectionTaskRunner) UpdateFeatureMap(featureMap map[string]bool) error {
	for featureId := range featureMap {
		task, err := i.availableTasks.Get(featureId)
		if err != nil {
			return err
		}
		if !typedmap.GetOrDefault(task.Labels(), inspection_contract.LabelKeyInspectionFeatureFlag, false) {
			return fmt.Errorf("task `%s` is not marked as a feature but requested to be included in the feature set of an inspection", task.UntypedID())
		}
		if featureMap[featureId] {
			i.featureTasks.Add(task)
		} else {
			i.featureTasks.Remove(featureId)
		}
		i.enabledFeatures[featureId] = featureMap[featureId]
	}
	return nil
}

// withRunContextValues returns a context with the value specific to a single run of task.
func (i *InspectionTaskRunner) withRunContextValues(ctx context.Context, runMode inspection_contract.InspectionTaskModeType, taskInput map[string]any) (context.Context, error) {
	rid := i.runIDGenerator.Generate()
	runCtx := khictx.WithValue(ctx, inspection_contract.InspectionTaskRunID, rid)
	runCtx = khictx.WithValue(runCtx, inspection_contract.InspectionTaskInspectionID, i.ID)
	runCtx = khictx.WithValue(runCtx, inspection_contract.InspectionSharedMap, i.inspectionSharedMap)
	runCtx = khictx.WithValue(runCtx, inspection_contract.GlobalSharedMap, inspectionRunnerGlobalSharedMap)
	runCtx = khictx.WithValue(runCtx, inspection_contract.InspectionTaskInput, taskInput)
	runCtx = khictx.WithValue(runCtx, inspection_contract.InspectionTaskMode, runMode)
	runCtx = khictx.WithValue(runCtx, inspection_contract.CurrentIOConfig, i.ioconfig)
	runCtx = khictx.WithValue(runCtx, inspection_contract.CurrentHistoryBuilder, history.NewBuilder(i.ioconfig.TemporaryFolder))

	return runCtx, nil
}

// Run executes the inspection. It resolves the task graph, sets up the context
// and metadata, and starts the task runner asynchronously.
func (i *InspectionTaskRunner) Run(ctx context.Context, req *inspection_contract.InspectionRequest) error {
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

	runCtx, err := i.withRunContextValues(ctx, inspection_contract.TaskModeRun, req.Values)
	if err != nil {
		return err
	}

	runMetadata := i.generateMetadataForRun(runCtx, &inspectionmetadata.HeaderMetadata{
		InspectTimeUnixSeconds: time.Now().Unix(),
		InspectionType:         currentInspectionType.Name,
		InspectionTypeIconPath: currentInspectionType.Icon,
		SuggestedFileName:      "unnamed.khi",
	}, runnableTaskGraph)

	runCtx = khictx.WithValue(runCtx, inspection_contract.InspectionRunMetadata, runMetadata)

	cancelableCtx, cancel := context.WithCancel(runCtx)
	i.cancel = cancel

	runner, err := coretask.NewLocalRunner(runnableTaskGraph)
	if err != nil {
		i.cleanupAfterAnyRun(runCtx, runnableTaskGraph)
		return err
	}
	i.runner = runner

	i.metadata = runMetadata
	lifecycle.Default.NotifyInspectionStart(khictx.MustGetValue(runCtx, inspection_contract.InspectionTaskRunID), currentInspectionType.Name)

	err = i.runner.Run(cancelableCtx)
	if err != nil {
		i.cleanupAfterAnyRun(runCtx, runnableTaskGraph)
		return err
	}
	go func() {
		defer i.cleanupAfterAnyRun(runCtx, runnableTaskGraph)
		<-i.runner.Wait()
		progress, found := typedmap.Get(i.metadata, inspectionmetadata.ProgressMetadataKey)
		if !found {
			slog.ErrorContext(runCtx, "progress metadata was not found")
		}
		status := ""
		resultSize := 0
		if result, err := i.runner.Result(); err != nil {
			if errors.Is(cancelableCtx.Err(), context.Canceled) {
				progress.MarkCancelled()
				status = "cancel"
			} else {
				progress.MarkError()
				status = "error"
			}
			slog.WarnContext(runCtx, fmt.Sprintf("task %s was finished with an error\n%s", i.ID, err))
		} else {
			progress.MarkDone()
			status = "done"

			history, found := typedmap.Get(result, typedmap.NewTypedKey[inspection_contract.Store](inspection_contract.SerializerTaskID.ReferenceIDString()))
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
		lifecycle.Default.NotifyInspectionEnd(khictx.MustGetValue(runCtx, inspection_contract.InspectionTaskRunID), currentInspectionType.Name, status, resultSize)
	}()
	return nil
}

// Result returns the final result of a completed inspection.
// It extracts the inspection data store and serializable metadata from the task runner's result.
func (i *InspectionTaskRunner) Result() (*InspectionRunResult, error) {
	if i.runner == nil {
		return nil, fmt.Errorf("this task is not yet started")
	}

	v, err := i.runner.Result()
	if err != nil {
		return nil, err
	}

	inspectionDataStore, found := typedmap.Get(v, typedmap.NewTypedKey[inspection_contract.Store](inspection_contract.SerializerTaskID.ReferenceIDString()))
	if !found {
		return nil, fmt.Errorf("failed to get the serializer result")
	}

	md, err := inspectionmetadata.GetSerializableSubsetMapFromMetadataSet(i.metadata, filter.NewEnabledFilter(inspectionmetadata.LabelKeyIncludedInRunResultFlag, false))
	if err != nil {
		return nil, err
	}
	return &InspectionRunResult{
		Metadata:    md,
		ResultStore: inspectionDataStore,
	}, nil
}

// Metadata returns the serializable metadata of the current run.
func (i *InspectionTaskRunner) Metadata() (map[string]any, error) {
	if i.runner == nil {
		return nil, fmt.Errorf("this task is not yet started")
	}
	md, err := inspectionmetadata.GetSerializableSubsetMapFromMetadataSet(i.metadata, filter.NewEnabledFilter(inspectionmetadata.LabelKeyIncludedInRunResultFlag, false))
	if err != nil {
		return nil, err
	}
	return md, nil
}

// DryRun performs a dry run of the inspection.
// It resolves the task graph and runs it in dry-run mode to collect metadata without executing tasks.
func (i *InspectionTaskRunner) DryRun(ctx context.Context, req *inspection_contract.InspectionRequest) (*InspectionDryRunResult, error) {
	runnableTaskGraph, err := i.resolveTaskGraph()
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return nil, err
	}

	runner, err := coretask.NewLocalRunner(runnableTaskGraph)
	if err != nil {
		return nil, err
	}

	runCtx, err := i.withRunContextValues(ctx, inspection_contract.TaskModeDryRun, req.Values)
	if err != nil {
		return nil, err
	}
	defer i.cleanupAfterAnyRun(runCtx, runnableTaskGraph)

	dryrunMetadata := i.generateMetadataForDryRun(runCtx, &inspectionmetadata.HeaderMetadata{}, runnableTaskGraph)

	runCtx = khictx.WithValue(runCtx, inspection_contract.InspectionRunMetadata, dryrunMetadata)

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
	md, err := inspectionmetadata.GetSerializableSubsetMapFromMetadataSet(dryrunMetadata, filter.NewEnabledFilter(inspectionmetadata.LabelKeyIncludedInDryRunResultFlag, false))
	if err != nil {
		return nil, err
	}
	return &InspectionDryRunResult{
		Metadata: md,
	}, nil
}

func (i *InspectionTaskRunner) makeLoggers(ctx context.Context, minLevel slog.Level, tasks []coretask.UntypedTask) *inspectionmetadata.LogMetadata {
	logMetadata := inspectionmetadata.NewLogMetadata()
	for _, def := range tasks {
		inspectionID := i.ID
		runID := khictx.MustGetValue(ctx, inspection_contract.InspectionTaskRunID)
		taskID := def.UntypedID()
		logger.RegisterTaskLogger(inspectionID, taskID, runID, i.makeLogger(minLevel, logMetadata.GetTaskLogBuffer(taskID)))
	}
	return logMetadata
}

func (i *InspectionTaskRunner) makeLogger(minLevel slog.Level, logBuffer *bytes.Buffer) slog.Handler {
	stdoutWithColor := true
	if parameters.Debug.NoColor != nil && *parameters.Debug.NoColor {
		stdoutWithColor = false
	}
	logThrottleCount := 10 // Similar logs over logThrottleCount will be discarded

	return logger.NewTeeHandler(
		logger.NewThrottleFilter(logThrottleCount, logger.NewSeverityFilter(minLevel, logger.NewKHIFormatLogger(os.Stdout, stdoutWithColor))),
		logger.NewThrottleFilter(logThrottleCount, logger.NewSeverityFilter(minLevel, logger.NewKHIFormatLogger(logBuffer, false))),
	)
}

// GetCurrentMetadata returns the metadata map for the current inspection run.
func (i *InspectionTaskRunner) GetCurrentMetadata() (*typedmap.ReadonlyTypedMap, error) {
	if i.metadata == nil {
		return nil, fmt.Errorf("this task hasn't been started")
	}
	return i.metadata, nil
}

// Cancel requests the cancellation of a running inspection.
func (i *InspectionTaskRunner) Cancel() error {
	if i.cancel == nil {
		return fmt.Errorf("this task is not yet started")
	}
	if _, err := i.Result(); err == nil {
		return fmt.Errorf("task %s is already finished", i.ID)
	}
	i.cancel()
	return nil
}

// Wait returns a channel that is closed when the inspection finishes.
func (i *InspectionTaskRunner) Wait() <-chan interface{} {
	return i.runner.Wait()
}

func (i *InspectionTaskRunner) resolveTaskGraph() (*coretask.TaskSet, error) {
	if i.featureTasks == nil || i.availableTasks == nil {
		return nil, fmt.Errorf("this runner is not ready for resolving graph")
	}
	usedTasks := []coretask.UntypedTask{}
	usedTasks = append(usedTasks, i.featureTasks.GetAll()...)
	usedTasks = append(usedTasks, i.requiredTasks.GetAll()...)
	initialTaskSet, err := coretask.NewTaskSet(usedTasks)
	if err != nil {
		return nil, err
	}
	set, err := initialTaskSet.ResolveTask(i.availableTasks)
	if err != nil {
		return nil, err
	}

	wrapped, err := set.WrapGraph(taskid.NewDefaultImplementationID[any](inspection_contract.InspectionMainSubgraphName), []taskid.UntypedTaskReference{})
	if err != nil {
		return nil, err
	}

	// Add required pre process or post process for the subgraph
	err = wrapped.Add(inspection_impl.SerializeTask)
	if err != nil {
		return nil, err
	}

	return wrapped.ResolveTask(i.availableTasks)
}

func (i *InspectionTaskRunner) generateMetadataForDryRun(ctx context.Context, initHeader *inspectionmetadata.HeaderMetadata, taskGraph *coretask.TaskSet) *typedmap.ReadonlyTypedMap {
	writableMetadata := typedmap.NewTypedMap()
	i.addCommonMetadata(ctx, writableMetadata, initHeader, taskGraph)
	return writableMetadata.AsReadonly()
}

func (i *InspectionTaskRunner) generateMetadataForRun(ctx context.Context, initHeader *inspectionmetadata.HeaderMetadata, taskGraph *coretask.TaskSet) *typedmap.ReadonlyTypedMap {
	writableMetadata := typedmap.NewTypedMap()
	i.addCommonMetadata(ctx, writableMetadata, initHeader, taskGraph)
	return writableMetadata.AsReadonly()
}

func (i *InspectionTaskRunner) addCommonMetadata(ctx context.Context, writableMetadata *typedmap.TypedMap, initHeader *inspectionmetadata.HeaderMetadata, taskGraph *coretask.TaskSet) {
	typedmap.Set(writableMetadata, inspectionmetadata.HeaderMetadataKey, initHeader)
	typedmap.Set(writableMetadata, inspectionmetadata.ErrorMessageSetMetadataKey, inspectionmetadata.NewErrorMessageSetMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.FormFieldSetMetadataKey, inspectionmetadata.NewFormFieldSetMetadata())
	typedmap.Set(writableMetadata, inspectionmetadata.QueryMetadataKey, inspectionmetadata.NewQueryMetadata())

	progressMeta := inspectionmetadata.NewProgress()
	progressMeta.SetTotalTaskCount(len(coretask.Subset(taskGraph, filter.NewEnabledFilter(inspection_contract.LabelKeyProgressReportable, false)).GetAll()))
	typedmap.Set(writableMetadata, inspectionmetadata.ProgressMetadataKey, progressMeta)

	taskGraphStr, err := taskGraph.DumpGraphviz()
	if err != nil {
		taskGraphStr = fmt.Sprintf("failed to generate task graph %v", err.Error())
	}
	typedmap.Set(writableMetadata, inspectionmetadata.InspectionPlanMetadataKey, inspectionmetadata.NewInspectionPlanMetadata(taskGraphStr))

	logMetadata := i.makeLoggers(ctx, getLogLevel(), taskGraph.GetAll())
	typedmap.Set(writableMetadata, inspectionmetadata.LogMetadataKey, logMetadata)
}

func (i *InspectionTaskRunner) cleanupAfterAnyRun(ctx context.Context, taskGraph *coretask.TaskSet) {
	// Clean up loggers registered for all tasks
	tasks := taskGraph.GetAll()
	for _, task := range tasks {
		inspectionID := i.ID
		runID := khictx.MustGetValue(ctx, inspection_contract.InspectionTaskRunID)
		logger.UnregisterTaskLogger(inspectionID, task.UntypedID(), runID)
	}
}

func getLogLevel() slog.Level {
	if parameters.Debug.Verbose != nil && *parameters.Debug.Verbose {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
