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

package gcpcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"google.golang.org/protobuf/encoding/protojson"
)

// SnapshotToCAIRawLog converts a CAIAssetSnapshot to a raw Log entity, optionally preprocessing the unmarshaled JSON map.
func SnapshotToCAIRawLog(idGen *id.Generator, s *CAIAssetSnapshot, preprocessRawMap func(map[string]any)) (*log.Log, error) {
	jsonBytes, err := protojson.Marshal(s.TemporalAsset)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal temporal asset to JSON: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal temporal asset JSON: %w", err)
	}
	if preprocessRawMap != nil {
		preprocessRawMap(m)
	}
	node, nodeErr := structured.FromGoValue(m, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if nodeErr != nil {
		return nil, fmt.Errorf("failed to convert temporal asset map to structured node: %w", nodeErr)
	}

	reader := structured.NewNodeReader(node)
	return log.NewLogWithTimestamp(idGen, reader, s.StartTime()), nil
}

// CAIAssetSearchTarget defines the search scope and discovery function for querying Cloud Asset Inventory.
type CAIAssetSearchTarget struct {
	// Scope is the CAI search scope and history parent, e.g., "projects/my-project".
	Scope string
	// Discover returns the full CAI asset names whose history should be fetched.
	Discover func(ctx context.Context, fetcher CAIFetcher) ([]string, error)
}

// CAITaskSuiteConfig configures a standard 5-task Cloud Asset Inventory inspection pipeline.
type CAITaskSuiteConfig[Identity any] struct {
	// TaskIDs contains the 5 task IDs for the CAI pipeline.
	TaskIDs CAITaskIDSet

	// FetcherDependencies specifies additional task dependencies required by the Fetcher task.
	// APIClientFactoryTaskID, APIClientCallOptionsInjectorTaskID, InputStartTimeTaskID, and InputEndTimeTaskID
	// are automatically added.
	FetcherDependencies []coretask.Dependency

	// ResolveSearchTarget returns the CAI project ID, search scope, and asset discovery callback.
	// If skip is true, the fetcher immediately returns an empty snapshot slice without calling CAI.
	ResolveSearchTarget func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (projectID string, target CAIAssetSearchTarget, skip bool, err error)

	// PreprocessRawMap is an optional hook called on the unmarshaled JSON map of a TemporalAsset
	// before converting it into a structured.Node.
	PreprocessRawMap func(m map[string]any)

	// ExtractIdentity extracts a domain-specific Identity value from a log's NodeReader.
	// Returning ok=false indicates an unrecognized or incomplete log that should be grouped under "unknown"
	// and skipped during timeline mapping.
	ExtractIdentity func(reader *structured.NodeReader) (identity Identity, ok bool)

	// IdentityGroupKey returns a unique string key for grouping logs of the same resource.
	IdentityGroupKey func(identity Identity) string

	// FormatLogSummary returns the human-readable summary string for LogChangeSet.
	FormatLogSummary func(identity Identity) string

	// MapperDependencies specifies additional task dependencies required by the TimelineMapper task.
	// InputStartTimeTaskID is automatically added.
	MapperDependencies []coretask.Dependency

	// MapInitialRevision returns the staging spec for an asset snapshot that was active at queryStartTime.
	// If skip is true, no timeline revision is staged for this log.
	MapInitialRevision func(ctx context.Context, l *log.Log, identity Identity, observedTime time.Time) (spec CAIInitialSnapshotRevisionSpec, skip bool, err error)
}

// CAITaskSuite bundles the 5 tasks constituting a Cloud Asset Inventory inspection pipeline.
type CAITaskSuite struct {
	FetcherTask        coretask.Task[[]*CAIAssetSnapshot]
	RawLogTask         coretask.Task[[]*log.Log]
	LogGrouperTask     coretask.Task[inspectiontaskbase.LogGroupMap]
	LogIngesterTask    coretask.Task[struct{}]
	TimelineMapperTask coretask.Task[struct{}]
}

// Tasks returns all 5 tasks in the suite.
func (s *CAITaskSuite) Tasks() []coretask.UntypedTask {
	return []coretask.UntypedTask{
		s.FetcherTask,
		s.RawLogTask,
		s.LogGrouperTask,
		s.LogIngesterTask,
		s.TimelineMapperTask,
	}
}

// Register registers all 5 pipeline tasks into the given task registry.
func (s *CAITaskSuite) Register(registry coretask.TaskRegistry) error {
	return coretask.RegisterTasks(registry, s.Tasks()...)
}

// NewCAITaskSuite constructs a CAITaskSuite from the given configuration.
func NewCAITaskSuite[Identity any](cfg CAITaskSuiteConfig[Identity]) *CAITaskSuite {
	ingester := &caiLogIngester[Identity]{
		rawLogRef:        cfg.TaskIDs.RawLog.Ref(),
		extractIdentity:  cfg.ExtractIdentity,
		formatLogSummary: cfg.FormatLogSummary,
	}

	mapperDeps := append(
		[]coretask.Dependency{
			InputStartTimeTaskID.Ref(),
		},
		cfg.MapperDependencies...,
	)

	mapper := &caiTimelineMapper[Identity]{
		logIngesterRef:     cfg.TaskIDs.LogIngester.Ref(),
		groupedLogRef:      cfg.TaskIDs.LogGrouper.Ref(),
		dependencies:       mapperDeps,
		extractIdentity:    cfg.ExtractIdentity,
		mapInitialRevision: cfg.MapInitialRevision,
	}

	return &CAITaskSuite{
		FetcherTask:        newCAIFetcherTask(cfg),
		RawLogTask:         newCAIRawLogTask(cfg),
		LogGrouperTask:     newCAILogGrouperTask(cfg),
		LogIngesterTask:    inspectiontaskbase.NewLogIngesterTask(cfg.TaskIDs.LogIngester, ingester),
		TimelineMapperTask: inspectiontaskbase.NewLogToTimelineMapperTask(cfg.TaskIDs.TimelineMapper, mapper),
	}
}

func newCAIFetcherTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.Task[[]*CAIAssetSnapshot] {
	fetcherDeps := append(
		[]coretask.Dependency{
			APIClientFactoryTaskID.Ref(),
			APIClientCallOptionsInjectorTaskID.Ref(),
			InputStartTimeTaskID.Ref(),
			InputEndTimeTaskID.Ref(),
		},
		cfg.FetcherDependencies...,
	)

	return inspectiontaskbase.NewInspectionTask(
		cfg.TaskIDs.Fetcher,
		fetcherDeps,
		func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*CAIAssetSnapshot, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return []*CAIAssetSnapshot{}, nil
			}
			projectID, target, skip, err := cfg.ResolveSearchTarget(ctx, taskMode)
			if err != nil {
				return nil, err
			}
			if skip {
				return []*CAIAssetSnapshot{}, nil
			}

			factory := coretask.GetTaskResult(ctx, APIClientFactoryTaskID.Ref())
			injector := coretask.GetTaskResult(ctx, APIClientCallOptionsInjectorTaskID.Ref())
			startTime := coretask.GetTaskResult(ctx, InputStartTimeTaskID.Ref())
			endTime := coretask.GetTaskResult(ctx, InputEndTimeTaskID.Ref())

			fetcher := NewCAIFetcher(factory, injector, projectID)
			snapshots, fetchErr := FetchCAIAssetSnapshots(ctx, fetcher, target.Scope, startTime, endTime, target.Discover)
			if fetchErr != nil {
				slog.WarnContext(ctx, "failed to fetch resource snapshots from CAI", "error", fetchErr)
				return []*CAIAssetSnapshot{}, nil
			}
			return snapshots, nil
		},
	)
}

func newCAIRawLogTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.Task[[]*log.Log] {
	return inspectiontaskbase.NewInspectionTask(
		cfg.TaskIDs.RawLog,
		[]coretask.Dependency{
			cfg.TaskIDs.Fetcher.Ref(),
		},
		func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*log.Log, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return []*log.Log{}, nil
			}
			snapshots := coretask.GetTaskResult(ctx, cfg.TaskIDs.Fetcher.Ref())
			idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)

			logs := make([]*log.Log, len(snapshots))
			err := progress.ForEach(ctx, snapshots, func(i int, s *CAIAssetSnapshot) error {
				l, err := SnapshotToCAIRawLog(idGen, s, cfg.PreprocessRawMap)
				if err != nil {
					return err
				}
				logs[i] = l
				return nil
			}, progress.WithUnit("snapshots"))
			if err != nil {
				return nil, err
			}
			return logs, nil
		},
	)
}

func newCAILogGrouperTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.Task[inspectiontaskbase.LogGroupMap] {
	return inspectiontaskbase.NewLogGrouperTask(
		cfg.TaskIDs.LogGrouper,
		cfg.TaskIDs.RawLog.Ref(),
		func(ctx context.Context, l *log.Log) string {
			identity, ok := cfg.ExtractIdentity(l.NodeReader)
			if !ok {
				return "unknown"
			}
			key := cfg.IdentityGroupKey(identity)
			if key == "" {
				return "unknown"
			}
			return key
		},
	)
}

type caiLogIngester[Identity any] struct {
	rawLogRef        taskid.TaskReference[[]*log.Log]
	extractIdentity  func(reader *structured.NodeReader) (Identity, bool)
	formatLogSummary func(identity Identity) string
}

var _ inspectiontaskbase.LogIngester = (*caiLogIngester[any])(nil)

// RawLogTask returns the task reference providing raw CAI logs.
func (i *caiLogIngester[Identity]) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return i.rawLogRef
}

// Dependencies returns additional task dependencies for log ingestion.
func (i *caiLogIngester[Identity]) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// ProcessLog populates the metadata into LogChangeSet.
func (i *caiLogIngester[Identity]) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetTimestamp(l.Timestamp)
	cs.SetLogType(LogTypeCAIResourceSnapshot)
	cs.SetSeverity(inspectioncore.SeverityInfo)

	if identity, ok := i.extractIdentity(l.NodeReader); ok {
		cs.SetSummary(i.formatLogSummary(identity))
	} else {
		cs.SetSummary("CAI resource snapshot: unknown")
	}

	return cs, nil
}

type caiTimelineMapper[Identity any] struct {
	inspectiontaskbase.StatelessMapperBase
	logIngesterRef     taskid.TaskReference[struct{}]
	groupedLogRef      taskid.TaskReference[inspectiontaskbase.LogGroupMap]
	dependencies       []coretask.Dependency
	extractIdentity    func(reader *structured.NodeReader) (Identity, bool)
	mapInitialRevision func(ctx context.Context, l *log.Log, identity Identity, observedTime time.Time) (CAIInitialSnapshotRevisionSpec, bool, error)
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*caiTimelineMapper[any])(nil)

// LogIngesterTask returns the prerequisite log ingester task reference.
func (m *caiTimelineMapper[Identity]) LogIngesterTask() taskid.TaskReference[struct{}] {
	return m.logIngesterRef
}

// GroupedLogTask returns the reference to the task providing grouped CAI logs.
func (m *caiTimelineMapper[Identity]) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return m.groupedLogRef
}

// Dependencies returns additional task dependencies for timeline mapping.
func (m *caiTimelineMapper[Identity]) Dependencies() []coretask.Dependency {
	return m.dependencies
}

// ProcessLogByGroup processes a log entry and stages a timeline revision for existing resources.
func (m *caiTimelineMapper[Identity]) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	assetWindowStartTime, assetWindowEndTime, isDeleted := ExtractCAITimeWindow(l.NodeReader)
	if isDeleted {
		return nil, struct{}{}, nil
	}

	queryStartTime := coretask.GetTaskResult(ctx, InputStartTimeTaskID.Ref())
	if !IsCAIAssetActiveAt(assetWindowStartTime, assetWindowEndTime, queryStartTime) {
		return nil, struct{}{}, nil
	}

	identity, ok := m.extractIdentity(l.NodeReader)
	if !ok {
		return nil, struct{}{}, nil
	}

	observedTime := assetWindowStartTime
	if observedTime.IsZero() {
		observedTime = queryStartTime
	}

	spec, skip, err := m.mapInitialRevision(ctx, l, identity, observedTime)
	if err != nil || skip {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	StageCAIInitialSnapshotRevisions(cs, spec)
	return cs, struct{}{}, nil
}

// CAIActiveAssetState holds the parsed identity and resource body for an asset active at queryStartTime.
type CAIActiveAssetState[Identity any] struct {
	Identity     Identity
	ResourceBody structured.Node
	ObservedTime time.Time
}

func parseActiveAssetState[Identity any](
	l *log.Log,
	queryStartTime time.Time,
	extractIdentity func(reader *structured.NodeReader) (Identity, bool),
	extractBody func(reader *structured.NodeReader) structured.Node,
) (CAIActiveAssetState[Identity], bool) {
	assetWindowStartTime, assetWindowEndTime, isDeleted := ExtractCAITimeWindow(l.NodeReader)
	if isDeleted || !IsCAIAssetActiveAt(assetWindowStartTime, assetWindowEndTime, queryStartTime) {
		return CAIActiveAssetState[Identity]{}, false
	}
	identity, ok := extractIdentity(l.NodeReader)
	if !ok {
		return CAIActiveAssetState[Identity]{}, false
	}
	body := extractBody(l.NodeReader)
	if body == nil {
		return CAIActiveAssetState[Identity]{}, false
	}

	observedTime := assetWindowStartTime
	if observedTime.IsZero() {
		observedTime = queryStartTime
	}

	return CAIActiveAssetState[Identity]{
		Identity:     identity,
		ResourceBody: body,
		ObservedTime: observedTime,
	}, true
}

// ExtractCAIActiveAssetStates filters CAI snapshot logs to those active at queryStartTime,
// deduplicating by identityKey so that the most recently observed active snapshot wins.
func ExtractCAIActiveAssetStates[Identity any](
	ctx context.Context,
	logs []*log.Log,
	queryStartTime time.Time,
	extractIdentity func(reader *structured.NodeReader) (Identity, bool),
	identityKey func(Identity) string,
	extractBody func(reader *structured.NodeReader) structured.Node,
) []CAIActiveAssetState[Identity] {
	var results []CAIActiveAssetState[Identity]
	indexByKey := make(map[string]int)

	_ = progress.ForEach(ctx, logs, func(_ int, l *log.Log) error {
		state, ok := parseActiveAssetState(l, queryStartTime, extractIdentity, extractBody)
		if !ok {
			return nil
		}

		key := identityKey(state.Identity)
		if idx, found := indexByKey[key]; found {
			if results[idx].ObservedTime.After(state.ObservedTime) {
				return nil
			}
			results[idx] = state
			return nil
		}

		indexByKey[key] = len(results)
		results = append(results, state)
		return nil
	}, progress.WithUnit("logs"))

	return results
}

// NewCAIInitialResourceStateProviderTask creates an inspection task that provides initial resource states
// from CAI snapshots active at queryStartTime, and attaches a SubsequentTaskRefs label pointing to the suite's TimelineMapper.
func NewCAIInitialResourceStateProviderTask[Identity any, Provider any](
	taskID taskid.TaskImplementationID[Provider],
	suiteTaskIDs CAITaskIDSet,
	extractIdentity func(reader *structured.NodeReader) (Identity, bool),
	identityKey func(Identity) string,
	extractBody func(reader *structured.NodeReader) structured.Node,
	buildProvider func(activeStates []CAIActiveAssetState[Identity]) Provider,
) coretask.Task[Provider] {
	return inspectiontaskbase.NewInspectionTask(
		taskID,
		[]coretask.Dependency{
			suiteTaskIDs.RawLog.Ref(),
			InputStartTimeTaskID.Ref(),
		},
		func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (Provider, error) {
			if taskMode == inspectioncore.TaskModeDryRun {
				return buildProvider(nil), nil
			}
			logs := coretask.GetTaskResult(ctx, suiteTaskIDs.RawLog.Ref())
			queryStartTime := coretask.GetTaskResult(ctx, InputStartTimeTaskID.Ref())
			states := ExtractCAIActiveAssetStates(ctx, logs, queryStartTime, extractIdentity, identityKey, extractBody)
			return buildProvider(states), nil
		},
		coretask.WithSelectionPriority(1000),
		coretask.NewSubsequentTaskRefsTaskLabel(suiteTaskIDs.TimelineMapper.Ref()),
	)
}
