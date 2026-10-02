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

// CAISearchTargetResolver resolves the CAI project ID, search scope, and asset discovery callback for the current inspection.
// If skip is true, the fetcher immediately returns an empty snapshot slice without calling CAI.
type CAISearchTargetResolver func(ctx context.Context) (projectID string, target CAIAssetSearchTarget, skip bool, err error)

// CAIInitialRevisionMapper builds the staging spec for an asset snapshot that was active at queryStartTime.
// If skip is true, no timeline revision is staged for this log.
type CAIInitialRevisionMapper[Identity any] func(ctx context.Context, l *log.Log, identity Identity, observedTime time.Time) (spec CAIInitialSnapshotRevisionSpec, skip bool, err error)

// CAITaskSuiteConfig configures a standard 5-task Cloud Asset Inventory inspection pipeline.
type CAITaskSuiteConfig[Identity any] struct {
	// TaskIDs contains the 5 task IDs for the CAI pipeline.
	TaskIDs CAITaskIDSet

	// BindSearchTargetResolver declares the task inputs required to resolve the CAI search target
	// and returns the resolver invoked by the fetcher task in Run mode.
	BindSearchTargetResolver func(b *coretask.Binder) CAISearchTargetResolver

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

	// BindInitialRevisionMapper declares the task inputs required by the timeline mapper
	// and returns the callback that builds the initial snapshot revision spec for an active asset.
	BindInitialRevisionMapper func(b *coretask.Binder) CAIInitialRevisionMapper[Identity]
}

// CAITaskSuite bundles the 5 tasks constituting a Cloud Asset Inventory inspection pipeline.
type CAITaskSuite struct {
	FetcherTask        coretask.DefinedTask[[]*CAIAssetSnapshot]
	RawLogTask         coretask.DefinedTask[[]*log.Log]
	LogGrouperTask     coretask.DefinedTask[inspectiontaskbase.LogGroupMap]
	LogIngesterTask    coretask.DefinedTask[struct{}]
	TimelineMapperTask coretask.DefinedTask[struct{}]
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

// DefineCAITaskSuite constructs a CAITaskSuite from the given configuration.
func DefineCAITaskSuite[Identity any](cfg CAITaskSuiteConfig[Identity]) *CAITaskSuite {
	return &CAITaskSuite{
		FetcherTask:        defineCAIFetcherTask(cfg),
		RawLogTask:         defineCAIRawLogTask(cfg),
		LogGrouperTask:     defineCAILogGrouperTask(cfg),
		LogIngesterTask:    defineCAILogIngesterTask(cfg),
		TimelineMapperTask: defineCAITimelineMapperTask(cfg),
	}
}

func defineCAIFetcherTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.DefinedTask[[]*CAIAssetSnapshot] {
	return inspectiontaskbase.DefineInspectionTask(
		cfg.TaskIDs.Fetcher,
		func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[[]*CAIAssetSnapshot] {
			factory := coretask.Use(b, APIClientFactoryTaskID.Ref())
			injector := coretask.Use(b, APIClientCallOptionsInjectorTaskID.Ref())
			startTime := coretask.Use(b, InputStartTimeTaskID.Ref())
			endTime := coretask.Use(b, InputEndTimeTaskID.Ref())
			resolveSearchTarget := cfg.BindSearchTargetResolver(b)

			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*CAIAssetSnapshot, error) {
				if taskMode == inspectioncore.TaskModeDryRun {
					return []*CAIAssetSnapshot{}, nil
				}
				projectID, target, skip, err := resolveSearchTarget(ctx)
				if err != nil {
					return nil, err
				}
				if skip {
					return []*CAIAssetSnapshot{}, nil
				}

				fetcher := NewCAIFetcher(factory.Get(ctx), injector.Get(ctx), projectID)
				snapshots, fetchErr := FetchCAIAssetSnapshots(ctx, fetcher, target.Scope, startTime.Get(ctx), endTime.Get(ctx), target.Discover)
				if fetchErr != nil {
					slog.WarnContext(ctx, "failed to fetch resource snapshots from CAI", "error", fetchErr)
					return []*CAIAssetSnapshot{}, nil
				}
				return snapshots, nil
			}
		},
	)
}

func defineCAIRawLogTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.DefinedTask[[]*log.Log] {
	return inspectiontaskbase.DefineInspectionTask(
		cfg.TaskIDs.RawLog,
		func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[[]*log.Log] {
			snapshots := coretask.Use(b, cfg.TaskIDs.Fetcher.Ref())

			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) ([]*log.Log, error) {
				if taskMode == inspectioncore.TaskModeDryRun {
					return []*log.Log{}, nil
				}
				rawSnapshots := snapshots.Get(ctx)
				idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)

				logs := make([]*log.Log, len(rawSnapshots))
				err := progress.ForEach(ctx, rawSnapshots, func(i int, s *CAIAssetSnapshot) error {
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
			}
		},
	)
}

func defineCAILogGrouperTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.DefinedTask[inspectiontaskbase.LogGroupMap] {
	return inspectiontaskbase.DefineLogGrouperTask(
		cfg.TaskIDs.LogGrouper,
		cfg.TaskIDs.RawLog.Ref(),
		func(_ *coretask.Binder) inspectiontaskbase.LogGrouperFunc {
			return func(_ context.Context, l *log.Log) string {
				identity, ok := cfg.ExtractIdentity(l.NodeReader)
				if !ok {
					return "unknown"
				}
				key := cfg.IdentityGroupKey(identity)
				if key == "" {
					return "unknown"
				}
				return key
			}
		},
	)
}

func defineCAILogIngesterTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.DefinedTask[struct{}] {
	return inspectiontaskbase.DefineLogIngesterTask(
		cfg.TaskIDs.LogIngester,
		cfg.TaskIDs.RawLog.Ref(),
		func(_ *coretask.Binder) inspectiontaskbase.LogIngesterFunc {
			return func(_ context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
				return processCAILog(l, cfg.ExtractIdentity, cfg.FormatLogSummary)
			}
		},
	)
}

// processCAILog populates the metadata of a CAI resource snapshot log into a LogChangeSet.
func processCAILog[Identity any](
	l *log.Log,
	extractIdentity func(reader *structured.NodeReader) (Identity, bool),
	formatLogSummary func(identity Identity) string,
) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetTimestamp(l.Timestamp)
	cs.SetLogType(LogTypeCAIResourceSnapshot)
	cs.SetSeverity(inspectioncore.SeverityInfo)

	if identity, ok := extractIdentity(l.NodeReader); ok {
		cs.SetSummary(formatLogSummary(identity))
	} else {
		cs.SetSummary("CAI resource snapshot: unknown")
	}

	return cs, nil
}

func defineCAITimelineMapperTask[Identity any](cfg CAITaskSuiteConfig[Identity]) coretask.DefinedTask[struct{}] {
	return inspectiontaskbase.DefineLogToTimelineMapperTask(
		cfg.TaskIDs.TimelineMapper,
		inspectiontaskbase.TimelineMapperInputs{
			LogIngester: cfg.TaskIDs.LogIngester.Ref(),
			GroupedLogs: cfg.TaskIDs.LogGrouper.Ref(),
		},
		func(b *coretask.Binder) inspectiontaskbase.TimelineMapper[struct{}] {
			return &caiTimelineMapper[Identity]{
				startTime:          coretask.Use(b, InputStartTimeTaskID.Ref()),
				extractIdentity:    cfg.ExtractIdentity,
				mapInitialRevision: cfg.BindInitialRevisionMapper(b),
			}
		},
	)
}

type caiTimelineMapper[Identity any] struct {
	inspectiontaskbase.StatelessMapperBase
	startTime          coretask.Input[time.Time]
	extractIdentity    func(reader *structured.NodeReader) (Identity, bool)
	mapInitialRevision CAIInitialRevisionMapper[Identity]
}

var _ inspectiontaskbase.TimelineMapper[struct{}] = (*caiTimelineMapper[any])(nil)

// ProcessLogByGroup implements inspectiontaskbase.TimelineMapper.
func (m *caiTimelineMapper[Identity]) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	cs, err := mapCAILogToTimeline(ctx, l, m.startTime.Get(ctx), m.extractIdentity, m.mapInitialRevision)
	return cs, struct{}{}, err
}

// mapCAILogToTimeline processes a CAI snapshot log entry and stages a timeline revision if the asset was active at queryStartTime.
func mapCAILogToTimeline[Identity any](
	ctx context.Context,
	l *log.Log,
	queryStartTime time.Time,
	extractIdentity func(reader *structured.NodeReader) (Identity, bool),
	mapInitialRevision CAIInitialRevisionMapper[Identity],
) (*khifilev6.TimelineChangeSet, error) {
	assetWindowStartTime, assetWindowEndTime, isDeleted := ExtractCAITimeWindow(l.NodeReader)
	if isDeleted {
		return nil, nil
	}

	if !IsCAIAssetActiveAt(assetWindowStartTime, assetWindowEndTime, queryStartTime) {
		return nil, nil
	}

	identity, ok := extractIdentity(l.NodeReader)
	if !ok {
		return nil, nil
	}

	observedTime := assetWindowStartTime
	if observedTime.IsZero() {
		observedTime = queryStartTime
	}

	spec, skip, err := mapInitialRevision(ctx, l, identity, observedTime)
	if err != nil || skip {
		return nil, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	StageCAIInitialSnapshotRevisions(cs, spec)
	return cs, nil
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

// DefineCAIInitialResourceStateProviderTask defines an inspection task that provides initial resource states
// from CAI snapshots active at queryStartTime.
func DefineCAIInitialResourceStateProviderTask[Identity any, Provider any](
	taskID taskid.TaskImplementationID[Provider],
	suiteTaskIDs CAITaskIDSet,
	extractIdentity func(reader *structured.NodeReader) (Identity, bool),
	identityKey func(Identity) string,
	extractBody func(reader *structured.NodeReader) structured.Node,
	buildProvider func(activeStates []CAIActiveAssetState[Identity]) Provider,
) coretask.DefinedTask[Provider] {
	return inspectiontaskbase.DefineInspectionTask(
		taskID,
		func(b *coretask.Binder) inspectiontaskbase.InspectionTaskFunc[Provider] {
			logs := coretask.Use(b, suiteTaskIDs.RawLog.Ref())
			coretask.After(b, suiteTaskIDs.TimelineMapper.Ref())
			queryStartTime := coretask.Use(b, InputStartTimeTaskID.Ref())

			return func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (Provider, error) {
				if taskMode == inspectioncore.TaskModeDryRun {
					return buildProvider(nil), nil
				}
				states := ExtractCAIActiveAssetStates(ctx, logs.Get(ctx), queryStartTime.Get(ctx), extractIdentity, identityKey, extractBody)
				return buildProvider(states), nil
			}
		},
		coretask.WithSelectionPriority(1000),
	)
}
