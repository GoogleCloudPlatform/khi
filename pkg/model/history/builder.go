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

package history

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/binarychunk"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo"
	"golang.org/x/sync/errgroup"
)

type BuilderLogWalker = func(logIndex int, l *log.LogEntity) *ChangeSet

// Builder builds History from ChangeSet obtained from parsers.
type Builder struct {
	history                *History
	historyLock            sync.Mutex
	binaryChunk            *binarychunk.Builder
	timelinemap            *common.ShardingMap[*ResourceTimeline]
	timelineBuilders       *common.ShardingMap[*TimelineBuilder]
	logIdToSerializableLog *common.ShardingMap[*SerializableLog]
	historyResourceCache   *common.ShardingMap[*Resource]
	sorter                 *ResourceSorter
	ClusterResource        *resourceinfo.Cluster
}

func NewBuilder(ioConfig *ioconfig.IOConfig) *Builder {
	return &Builder{
		history:                NewHistory(),
		historyLock:            sync.Mutex{},
		binaryChunk:            binarychunk.NewBuilder(binarychunk.NewFileSystemGzipCompressor(ioConfig.TemporaryFolder), ioConfig.TemporaryFolder),
		timelinemap:            common.NewShardingMap[*ResourceTimeline](common.NewSuffixShardingProvider(128, 4)),
		timelineBuilders:       common.NewShardingMap[*TimelineBuilder](common.NewSuffixShardingProvider(128, 4)),
		logIdToSerializableLog: common.NewShardingMap[*SerializableLog](common.NewSuffixShardingProvider(128, 4)),
		historyResourceCache:   common.NewShardingMap[*Resource](common.NewSuffixShardingProvider(128, 4)),
		ClusterResource:        resourceinfo.NewClusterResourceInfo(),
		sorter: NewResourceSorter(
			&FirstRevisionTimeSortStrategy{
				TargetRelationship: enum.RelationshipPodBinding,
			},
			&FirstRevisionTimeSortStrategy{
				TargetRelationship: enum.RelationshipOwnerReference,
			},
			&FirstRevisionTimeSortStrategy{
				TargetRelationship: enum.RelationshipOperation,
			},
			NewNameSortStrategy(0, []string{"@Cluster", "core/v1", "apps/v1"}),
			NewNameSortStrategy(1, []string{"node", "pod", "service", "deployment", "replicaset", "daemonset", "cronjob", "job"}),
			NewNameSortStrategy(2, []string{"kube-system"}),
			NewNameSortStrategy(3, []string{}),
			NewNameSortStrategy(4, []string{}),
			&UnreachableSortStrategy{},
		),
	}
}

// Ensure specified resource path exists hierachicaly. Add resource history in middle or last when missing resource history was found on the path.
// (This method will do something similar to `mkdir -p`.)
func (builder *Builder) ensureResourcePath(resourcePath string) *Resource {
	// Get resource from cache
	resources := builder.historyResourceCache.AcquireShard(resourcePath)
	if resource, found := resources[resourcePath]; found {
		builder.historyResourceCache.ReleaseShard(resourcePath)
		return resource
	}
	builder.historyResourceCache.ReleaseShard(resourcePath)

	resourcePathFragment := strings.Split(resourcePath, "#")
	currentResourceContainer := &builder.history.Resources
	var currentResource *Resource
	currentPath := ""
	for _, fragment := range resourcePathFragment {
		if currentPath != "" {
			currentPath += "#"
		}
		currentPath += fragment
		resources := builder.historyResourceCache.AcquireShard(currentPath)
		if resource, found := resources[currentPath]; found {
			builder.historyResourceCache.ReleaseShard(currentPath)
			currentResource = resource
			currentResourceContainer = &resource.Children
			continue
		} else {
			nr := Resource{
				ResourceName:     fragment,
				Timeline:         "",
				Children:         []*Resource{},
				Relationship:     enum.RelationshipChild,
				FullResourcePath: currentPath,
			}
			*currentResourceContainer = append(*currentResourceContainer, &nr)
			currentResource = &nr
			currentResourceContainer = &nr.Children
			resources[currentPath] = currentResource
			builder.historyResourceCache.ReleaseShard(currentPath)
		}
	}
	return currentResource
}

func (builder *Builder) setLogSummary(logId string, summary string) error {
	serializableLogs := builder.logIdToSerializableLog.AcquireShardReadonly(logId)
	defer builder.logIdToSerializableLog.ReleaseShardReadonly(logId)
	if sl, exist := serializableLogs[logId]; exist {
		if sl.Summary != nil {
			slog.Warn(fmt.Sprintf("log: %s has its summary already. Ignoreing", logId))
			return nil
		}
		summaryRef, err := builder.binaryChunk.Write([]byte(summary))
		if err != nil {
			return err
		}

		sl.Summary = summaryRef
	}
	return fmt.Errorf("no log found %s", logId)
}

func (builder *Builder) setLogAnnotations(logId string, annotations []LogAnnotation) error {
	serializableLogs := builder.logIdToSerializableLog.AcquireShardReadonly(logId)
	defer builder.logIdToSerializableLog.ReleaseShardReadonly(logId)
	if sl, exist := serializableLogs[logId]; exist {
		slices.SortFunc(annotations, func(a LogAnnotation, b LogAnnotation) int {
			return a.Priority() - b.Priority()
		})
		for _, annotation := range annotations {
			result, err := annotation.Serialize(builder.binaryChunk)
			if err != nil {
				return err
			}
			sl.Annotations = append(sl.Annotations, result)
		}
	}
	return fmt.Errorf("no log found %s", logId)
}

func (builder *Builder) setLogSeverity(logId string, severity enum.Severity) error {
	serializableLogs := builder.logIdToSerializableLog.AcquireShardReadonly(logId)
	defer builder.logIdToSerializableLog.ReleaseShardReadonly(logId)
	if sl, exist := serializableLogs[logId]; exist {
		sl.Severity = severity
	}
	return fmt.Errorf("no log found %s", logId)
}

func (builder *Builder) GetTimelineBuilder(resourcePath string) *TimelineBuilder {
	resource := builder.ensureResourcePath(resourcePath)
	// When specified resource has no associated timeline
	if resource.Timeline == "" {
		tid := builder.generateTimelineID()
		timelineMap := builder.timelinemap.AcquireShard(tid)
		timeline := newTimeline(tid)
		resource.Timeline = tid
		builder.history.Timelines = append(builder.history.Timelines, timeline)
		timelineMap[tid] = timeline
		builder.timelinemap.ReleaseShard(tid)
	}
	// When the timeline builder was already created, then return it
	timelineBuilderMap := builder.timelineBuilders.AcquireShard(resource.Timeline)
	defer builder.timelineBuilders.ReleaseShard(resource.Timeline)
	if timelineBuilder, exist := timelineBuilderMap[resource.Timeline]; exist {
		return timelineBuilder
	}

	// If not exist, then create it.
	timelineMap := builder.timelinemap.AcquireShard(resource.Timeline)
	defer builder.timelinemap.ReleaseShard(resource.Timeline)
	timeline := timelineMap[resource.Timeline]
	tb := newTimelineBuilder(builder, timeline)
	timelineBuilderMap[resource.Timeline] = tb
	return tb
}

// GetChildResources returns the list of ResourceTimeline filtered with the prefix of resource path.
func (builder *Builder) GetChildResources(parentResourcePath string) []*Resource {
	currentList := builder.history.Resources
	searchPaths := strings.Split(parentResourcePath, "#")
	for i := 0; i < len(searchPaths); i++ {
		nextFind := searchPaths[i]
		found := false
		for _, resource := range currentList {
			if resource.ResourceName == nextFind {
				currentList = resource.Children
				found = true
				break
			}
		}
		if !found {
			currentList = make([]*Resource, 0)
		}
	}
	return currentList
}

// GetLog returns a copy of SerializableLog. Returns an error when the specified logId wasn't found from the list of consumed logs.
func (builder *Builder) GetLog(logId string) (*SerializableLog, error) {
	serializableLogs := builder.logIdToSerializableLog.AcquireShardReadonly(logId)
	defer builder.logIdToSerializableLog.ReleaseShardReadonly(logId)
	if serializedLog, found := serializableLogs[logId]; found {
		return serializedLog, nil
	}
	return nil, fmt.Errorf("log %s was not found", logId)
}

func (builder *Builder) addTimelineAlias(sourcePath string, destPath string) {
	builder.GetTimelineBuilder(sourcePath) // Make sure timeline element related to the resource is already generated
	copySource := builder.ensureResourcePath(sourcePath)
	copyTo := builder.ensureResourcePath(destPath)
	copyTo.Timeline = copySource.Timeline
}

func (builder *Builder) rewriteRelationship(path string, relationship enum.ParentRelationship) error {
	resource := builder.ensureResourcePath(path)
	if resource.Relationship != relationship && resource.Relationship != enum.RelationshipChild {
		return fmt.Errorf("failed to rewrite the parentRelationship of %s. It was already rewritten to %d", path, resource.Relationship)
	}
	resource.Relationship = relationship
	return nil
}

// PrepareParseLogs will prepare this builder to be ready to handle parsing logs by groups.
func (builder *Builder) PrepareParseLogs(ctx context.Context, entireLogs []*log.LogEntity, onLogPorcessed func()) error {
	parallelism := 16
	errGrp := errgroup.Group{}

	for i := 0; i < parallelism; i++ {
		shard := i
		errGrp.Go(func() error {
			logs := []*SerializableLog{}
			for logIndex := shard; logIndex < len(entireLogs); logIndex += parallelism {
				onLogPorcessed()
				log := entireLogs[logIndex]
				logId := log.ID()
				serializableLogs := builder.logIdToSerializableLog.AcquireShard(logId)
				if _, found := serializableLogs[logId]; found {
					builder.logIdToSerializableLog.ReleaseShard(logId)
					slog.WarnContext(ctx, fmt.Sprintf("duplicated consumed log %s", logId))
					continue
				}
				yaml := log.LogBody()
				bodyRef, err := builder.binaryChunk.Write([]byte(yaml))
				if err != nil {
					builder.logIdToSerializableLog.ReleaseShard(logId)
					return err
				}
				severity, err := log.Severity()
				if err != nil {
					severity = enum.SeverityUnknown
				}
				sl := &SerializableLog{
					ID:          logId,
					DisplayId:   log.GetStringOrDefault("insertId", "unknown"),
					Body:        bodyRef,
					Timestamp:   log.Timestamp(),
					Type:        log.LogType,
					Severity:    severity,
					Annotations: make([]any, 0),
				}
				logs = append(logs, sl)
				serializableLogs[sl.ID] = sl
				builder.logIdToSerializableLog.ReleaseShard(logId)
			}
			builder.historyLock.Lock()
			defer builder.historyLock.Unlock()
			builder.history.Logs = append(builder.history.Logs, logs...)
			return nil
		})
	}
	return errGrp.Wait()
}

func (builder *Builder) ParseLogsByGroups(ctx context.Context, groupedLogs []*log.LogEntity, logWalker BuilderLogWalker) error {
	for i, l := range groupedLogs {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
			cs := logWalker(i, l)
			if cs != nil {
				cp, err := cs.FlushToHistory(builder)
				for _, path := range cp {
					tb := builder.GetTimelineBuilder(path)
					tb.Sort()
				}
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (builder *Builder) sortData() error {
	sortedResources, err := builder.sorter.SortAll(builder, builder.history.Resources)
	if err != nil {
		return err
	}
	builder.history.Resources = sortedResources
	sort.Slice(builder.history.Logs, func(i, j int) bool {
		return builder.history.Logs[i].Timestamp.Sub(builder.history.Logs[j].Timestamp) <= 0
	})
	return nil
}

// Finalize flushes the binary chunk data and serialized metadata to the given io.Writer. Returns the written data size in bytes and error.
func (builder *Builder) Finalize(ctx context.Context, serializedMetadata map[string]interface{}, writer io.Writer, progress *progress.TaskProgress) (int, error) {
	fileSize := 0
	progress.Update(0, "Sorting log entries")
	progress.MarkIndeterminate()
	builder.history.Metadata = serializedMetadata
	err := builder.sortData()
	if err != nil {
		return 0, err
	}
	jsonString, err := json.Marshal(builder.history)
	if err != nil {
		return 0, err
	}
	jsonBytes := []byte(jsonString)

	if writtenSize, err := writer.Write([]byte("KHI")); err != nil {
		return 0, err
	} else {
		fileSize += writtenSize
	}

	metaFieldJsonSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(metaFieldJsonSize, uint32(len(jsonBytes)))
	if writtenSize, err := writer.Write(metaFieldJsonSize); err != nil {
		return 0, err
	} else {
		fileSize += writtenSize
	}

	if writtenSize, err := writer.Write(jsonBytes); err != nil {
		return 0, err
	} else {
		fileSize += writtenSize
	}

	if writtenSize, err := builder.binaryChunk.Build(ctx, writer, progress); err != nil {
		return 0, err
	} else {
		fileSize += writtenSize
	}
	return fileSize, nil
}

func (builder *Builder) generateTimelineID() string {
	const idLength = 7
	charset := "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"
	id := make([]byte, idLength)
	for i := 0; i < len(id); i++ {
		id[i] = charset[rand.Intn(len(charset))]
	}
	return string(id)
}
