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
	"slices"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/idgenerator"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/lifecycle"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InspectionInterceptor is a function that can intercept the execution of an inspection task.
type InspectionInterceptor func(ctx context.Context, req *inspectioncore_contract.InspectionRequest, next func(context.Context) error) error

var inspectionRunnerGlobalSharedMap = typedmap.NewTypedMap()

// DefaultFeatureTaskOrder is a number used for sorting feature task when the task has no LabelKeyFeatureTaskOrder label.
var DefaultFeatureTaskOrder = 1000000

// InspectionTaskRunner manages the lifecycle of a single inspection instance.
// It handles task graph resolution, execution, and result retrieval for a given inspection type and feature set.
type InspectionTaskRunner struct {
	inspectionServer       *InspectionTaskServer
	ID                     string
	runIDGenerator         idgenerator.IDGenerator
	enabledFeatures        map[string]bool
	availableTasks         *coretask.TaskSet
	featureTasks           *coretask.TaskSet
	runner                 coretask.TaskRunner
	runnerLock             sync.Mutex
	metadata               *typedmap.ReadonlyTypedMap
	cancel                 context.CancelFunc
	inspectionSharedMap    *typedmap.TypedMap
	currentInspectionType  string
	ioconfig               *inspectioncore_contract.IOConfig
	runContextOptions      []RunContextOption
	inspectionCreationTime time.Time
	interceptors           []InspectionInterceptor
	runComplete            chan (struct{})
}

// NewInspectionRunner creates a new InspectionTaskRunner.
func NewInspectionRunner(server *InspectionTaskServer, ioConfig *inspectioncore_contract.IOConfig, id string, options ...RunContextOption) *InspectionTaskRunner {
	runner := &InspectionTaskRunner{
		inspectionCreationTime: time.Now(),
		inspectionServer:       server,
		ID:                     id,
		runIDGenerator:         idgenerator.NewPrefixIDGenerator("run-"),
		enabledFeatures:        map[string]bool{},
		availableTasks:         nil,
		featureTasks:           nil,
		runner:                 nil,
		runnerLock:             sync.Mutex{},
		metadata:               nil,
		inspectionSharedMap:    typedmap.NewTypedMap(),
		cancel:                 nil,
		currentInspectionType:  "N/A",
		ioconfig:               ioConfig,
		runContextOptions:      options,
		interceptors:           []InspectionInterceptor{},
		runComplete:            make(chan struct{}),
	}
	runner.addDefaultRunContextOptions()
	runner.interceptors = append(runner.interceptors, InspectionTaskLogger(slog.LevelDebug, slog.LevelInfo, parameters.Debug.NoColor == nil || !*parameters.Debug.NoColor))
	return runner
}

func (i *InspectionTaskRunner) addDefaultRunContextOptions() {
	// Options common for any run from this runner.
	defaultRunContextOptions := []RunContextOption{
		RunContextOptionFromValue(inspectioncore_contract.InspectionCreationTime, i.inspectionCreationTime),
		RunContextOptionFromFunc(inspectioncore_contract.InspectionTaskRunID, func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (string, error) {
			return i.runIDGenerator.Generate(), nil
		}),
		RunContextOptionFromValue(inspectioncore_contract.InspectionTaskInspectionID, i.ID),
		RunContextOptionFromValue(inspectioncore_contract.InspectionSharedMap, i.inspectionSharedMap),
		RunContextOptionFromValue(inspectioncore_contract.GlobalSharedMap, inspectionRunnerGlobalSharedMap),
		RunContextOptionFromValue(inspectioncore_contract.CurrentIOConfig, i.ioconfig),
		RunContextOptionFromFunc(inspectioncore_contract.CurrentHistoryBuilder, func(ctx context.Context, mode inspectioncore_contract.InspectionTaskModeType) (*history.Builder, error) {
			return history.NewBuilder(i.ioconfig.TemporaryFolder), nil
		}),
	}

	i.runContextOptions = append(i.runContextOptions, defaultRunContextOptions...)
}

// AddInterceptors adds interceptors to the runner.
func (i *InspectionTaskRunner) AddInterceptors(interceptors ...InspectionInterceptor) {
	i.interceptors = append(i.interceptors, interceptors...)
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
	i.availableTasks = coretask.Subset(i.inspectionServer.RootTaskSet, filter.NewContainsElementFilter(inspectioncore_contract.LabelKeyInspectionTypes, inspectionType, true))
	defaultFeatures := coretask.Subset(i.availableTasks, filter.NewEnabledFilter(inspectioncore_contract.LabelKeyInspectionDefaultFeatureFlag, false))
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
	featureSet := coretask.Subset(i.availableTasks, filter.NewEnabledFilter(inspectioncore_contract.LabelKeyInspectionFeatureFlag, false))
	features := []FeatureListItem{}
	for _, featureTask := range featureSet.GetAll() {
		label := typedmap.GetOrDefault(featureTask.Labels(), inspectioncore_contract.LabelKeyFeatureTaskTitle, fmt.Sprintf("No label Set!(%s)", featureTask.UntypedID()))
		description := typedmap.GetOrDefault(featureTask.Labels(), inspectioncore_contract.LabelKeyFeatureTaskDescription, "")
		order := typedmap.GetOrDefault(featureTask.Labels(), inspectioncore_contract.LabelKeyFeatureTaskOrder, DefaultFeatureTaskOrder)
		enabled := false
		if v, exist := i.enabledFeatures[featureTask.UntypedID().String()]; exist && v {
			enabled = true
		}
		features = append(features, FeatureListItem{
			Id:          featureTask.UntypedID().String(),
			Label:       label,
			Description: description,
			Enabled:     enabled,
			Order:       order,
		})
	}
	slices.SortFunc(features, func(a FeatureListItem, b FeatureListItem) int { return a.Order - b.Order })
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
		if !typedmap.GetOrDefault(featureTask.Labels(), inspectioncore_contract.LabelKeyInspectionFeatureFlag, false) {
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
		if !typedmap.GetOrDefault(task.Labels(), inspectioncore_contract.LabelKeyInspectionFeatureFlag, false) {
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
func (i *InspectionTaskRunner) withRunContextValues(ctx context.Context, runner coretask.TaskRunner, runMode inspectioncore_contract.InspectionTaskModeType, taskInput map[string]any) (context.Context, error) {

	opts := make([]RunContextOption, 0, len(i.runContextOptions)+2)
	opts = append(opts, i.runContextOptions...)
	// Add option values determined for this run call.
	opts = append(opts, RunContextOptionFromValue(inspectioncore_contract.TaskRunner, runner))
	opts = append(opts, RunContextOptionFromValue(inspectioncore_contract.InspectionTaskInput, taskInput))
	opts = append(opts, RunContextOptionFromValue(inspectioncore_contract.InspectionTaskMode, runMode))

	var err error
	for _, opt := range opts {
		if ctx, err = opt(ctx, runMode); err != nil {
			return nil, err
		}
	}
	return ctx, nil
}

// Run executes the inspection. It resolves the task graph, sets up the context
// and metadata, and starts the task runner asynchronously.
func (i *InspectionTaskRunner) Run(ctx context.Context, req *inspectioncore_contract.InspectionRequest) error {
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

	runner, err := coretask.NewLocalRunner(runnableTaskGraph)
	if err != nil {
		return err
	}
	i.runner = runner

	runCtx, err := i.withRunContextValues(ctx, i.runner, inspectioncore_contract.TaskModeRun, req.Values)
	if err != nil {
		return err
	}

	runMetadata := i.generateMetadataForRun(runCtx, &inspectionmetadata.HeaderMetadata{
		InspectTimeUnixSeconds: time.Now().Unix(),
		InspectionType:         currentInspectionType.Name,
		InspectionTypeIconPath: currentInspectionType.Icon,
		SuggestedFileName:      "unnamed.khi",
	}, runnableTaskGraph)

	runCtx = khictx.WithValue(runCtx, inspectioncore_contract.InspectionRunMetadata, runMetadata)

	cancelableCtx, cancel := context.WithCancel(runCtx)
	i.cancel = cancel

	i.metadata = runMetadata
	lifecycle.Default.NotifyInspectionStart(khictx.MustGetValue(runCtx, inspectioncore_contract.InspectionTaskRunID), currentInspectionType.Name)

	// Run the inspection with interceptors
	runFunc := func(ctx context.Context) error {
		err := i.runner.Run(ctx)
		if err != nil {
			return err
		}
		<-i.runner.Wait()
		_, err = i.runner.Result()
		return err
	}

	for j := len(i.interceptors) - 1; j >= 0; j-- {
		interceptor := i.interceptors[j]
		next := runFunc
		runFunc = func(ctx context.Context) error {
			return interceptor(ctx, req, next)
		}
	}

	go func() {
		defer close(i.runComplete)
		runFunc(cancelableCtx)
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

			history, found := typedmap.Get(result, typedmap.NewTypedKey[inspectioncore_contract.Store](inspectioncore_contract.SerializerTaskID.ReferenceIDString()))
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
		lifecycle.Default.NotifyInspectionEnd(khictx.MustGetValue(runCtx, inspectioncore_contract.InspectionTaskRunID), currentInspectionType.Name, status, resultSize)
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

	inspectionDataStore, found := typedmap.Get(v, typedmap.NewTypedKey[inspectioncore_contract.Store](inspectioncore_contract.SerializerTaskID.ReferenceIDString()))
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
func (i *InspectionTaskRunner) DryRun(ctx context.Context, req *inspectioncore_contract.InspectionRequest) (*InspectionDryRunResult, error) {
	runnableTaskGraph, err := i.resolveTaskGraph()
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return nil, err
	}

	runner, err := coretask.NewLocalRunner(runnableTaskGraph)
	if err != nil {
		return nil, err
	}

	runCtx, err := i.withRunContextValues(ctx, runner, inspectioncore_contract.TaskModeDryRun, req.Values)
	if err != nil {
		return nil, err
	}

	dryrunMetadata := i.generateMetadataForDryRun(runCtx, &inspectionmetadata.HeaderMetadata{}, runnableTaskGraph)

	runCtx = khictx.WithValue(runCtx, inspectioncore_contract.InspectionRunMetadata, dryrunMetadata)

	runFunc := func(ctx context.Context) error {
		err := runner.Run(ctx)
		if err != nil {
			return err
		}
		<-runner.Wait()
		_, err = runner.Result()
		return err
	}

	for j := len(i.interceptors) - 1; j >= 0; j-- {
		interceptor := i.interceptors[j]
		next := runFunc
		runFunc = func(ctx context.Context) error {
			return interceptor(ctx, req, next)
		}
	}
	err = runFunc(runCtx)
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
func (i *InspectionTaskRunner) Wait() <-chan struct{} {
	return i.runComplete
}

func (i *InspectionTaskRunner) resolveTaskGraph() (*coretask.TaskSet, error) {
	if i.featureTasks == nil || i.availableTasks == nil {
		return nil, fmt.Errorf("this runner is not ready for resolving graph")
	}
	resolver := coretask.DefaultTaskGraphResolver
	resolvedTask, err := resolver.Resolve(i.featureTasks.GetAll(), i.availableTasks.GetAll())
	if err != nil {
		return nil, err
	}
	initialTaskSet, err := coretask.NewTaskSet(resolvedTask)
	if err != nil {
		return nil, err
	}
	return initialTaskSet.ToRunnableTaskSet()
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
	typedmap.Set(writableMetadata, inspectionmetadata.LogMetadataKey, inspectionmetadata.NewLogMetadata())

	progressMeta := inspectionmetadata.NewProgress()
	progressMeta.SetTotalTaskCount(len(coretask.Subset(taskGraph, filter.NewEnabledFilter(inspectioncore_contract.LabelKeyProgressReportable, false)).GetAll()))
	typedmap.Set(writableMetadata, inspectionmetadata.ProgressMetadataKey, progressMeta)

	taskGraphStr, err := taskGraph.DumpGraphviz()
	if err != nil {
		taskGraphStr = fmt.Sprintf("failed to generate task graph %v", err.Error())
	}
	typedmap.Set(writableMetadata, inspectionmetadata.InspectionPlanMetadataKey, inspectionmetadata.NewInspectionPlanMetadata(taskGraphStr))

}

func InspectionTaskLogger(logLevelForRun slog.Level, logLevelForDryRun slog.Level, withColor bool) InspectionInterceptor {
	return func(ctx context.Context, req *inspectioncore_contract.InspectionRequest, next func(context.Context) error) error {
		logMetadata := inspectionmetadata.NewLogMetadata()
		inspectionID := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskInspectionID)
		runID := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskRunID)
		mode := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionTaskMode)
		runner := khictx.MustGetValue(ctx, inspectioncore_contract.TaskRunner)
		logLevel := logLevelForRun
		if mode == inspectioncore_contract.TaskModeDryRun {
			logLevel = logLevelForDryRun
		}

		for _, def := range runner.Tasks() {
			l := makeLogger(logLevel, logMetadata.GetTaskLogBuffer(def.UntypedID()), withColor)
			logger.RegisterTaskLogger(inspectionID, def.UntypedID(), runID, l)
		}
		err := next(ctx)
		if err != nil {
			return err
		}
		for _, task := range runner.Tasks() {
			logger.UnregisterTaskLogger(inspectionID, task.UntypedID(), runID)
		}
		return nil
	}
}

func makeLogger(minLevel slog.Level, logBuffer *bytes.Buffer, withColor bool) slog.Handler {
	logThrottleCount := 10 // Similar logs over logThrottleCount will be discarded

	return logger.NewTeeHandler(
		logger.NewThrottleFilter(logThrottleCount, logger.NewSeverityFilter(minLevel, logger.NewKHIFormatLogger(os.Stdout, withColor))),
		logger.NewThrottleFilter(logThrottleCount, logger.NewSeverityFilter(minLevel, logger.NewKHIFormatLogger(logBuffer, false))),
	)
}
