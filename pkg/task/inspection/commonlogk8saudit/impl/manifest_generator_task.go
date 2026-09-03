// Copyright 2025 Google LLC
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

package commonlogk8saudit_impl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progressutil"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"golang.org/x/sync/errgroup"
)

var (
	pathAPIVersion   = structured.CompileFieldPath("apiVersion")
	pathKind         = structured.CompileFieldPath("kind")
	pathItems        = structured.CompileFieldPath("items")
	pathMetadataName = structured.CompileFieldPath("metadata.name")
)

// ManifestGeneratorTask is the task to generate manifest from k8s audit logs.
var ManifestGeneratorTask = inspectiontaskbase.NewProgressReportableInspectionTask(commonlogk8saudit_contract.ManifestGeneratorTaskID, []taskid.UntypedTaskReference{
	commonlogk8saudit_contract.ChangeTargetGrouperTaskID.Ref(),
	commonlogk8saudit_contract.K8sResourceMergeConfigTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (commonlogk8saudit_contract.ResourceManifestLogGroupMap, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return map[string]*commonlogk8saudit_contract.ResourceManifestLogGroup{}, nil
	}

	logGroups := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ChangeTargetGrouperTaskID.Ref())
	mergeConfigRegistry := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.K8sResourceMergeConfigTaskID.Ref())
	result := commonlogk8saudit_contract.ResourceManifestLogGroupMap{}
	resultLock := sync.Mutex{}

	doneGroupCount := atomic.Int32{}
	updator := progressutil.NewProgressUpdator(progress, time.Second, func(tp *inspectionmetadata.TaskProgressMetadata) {
		current := doneGroupCount.Load()
		total := len(logGroups)
		if total > 0 {
			tp.Percentage = float32(current) / float32(total)
		} else {
			tp.Percentage = 1.0
		}
		tp.Message = fmt.Sprintf("%d/%d", current, total)
	})
	updator.Start(ctx)
	defer updator.Done()

	grp, childCtx := errgroup.WithContext(ctx)
	grp.SetLimit(runtime.GOMAXPROCS(0))

	for path, group := range logGroups {
		grp.Go(func() error {
			defer doneGroupCount.Add(1)
			resourceLogs := []*commonlogk8saudit_contract.ResourceManifestLog{}
			generator := groupManifestGenerator{
				mergeConfigRegistry: mergeConfigRegistry,
				resourceName:        group.Resource.Name,
			}
			for _, l := range group.Logs {
				select {
				case <-childCtx.Done():
					return context.Canceled
				default:
					r, err := generator.Process(childCtx, l)
					if err != nil {
						return err
					}
					resourceLogs = append(resourceLogs, r)
				}
			}
			resultLock.Lock()
			defer resultLock.Unlock()
			result[path] = &commonlogk8saudit_contract.ResourceManifestLogGroup{
				Resource: group.Resource,
				Logs:     resourceLogs,
			}
			return nil
		})
	}

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	return result, nil
})

type groupManifestGenerator struct {
	// prevRevisionReader is the reader for the previous revision.
	prevRevisionReader *structured.NodeReader
	// mergeConfigRegistry is the registry for merge config.
	mergeConfigRegistry *k8s.K8sManifestMergeConfigRegistry
	// resourceName is the name of the resource.
	resourceName string
}

// Process processes the log to generate manifest.
func (g *groupManifestGenerator) Process(ctx context.Context, l *log.Log) (*commonlogk8saudit_contract.ResourceManifestLog, error) {
	if g.prevRevisionReader == nil {
		g.prevRevisionReader = structured.NewNodeReader(structured.NewEmptyMapNode())
	}
	fieldSet, _ := commonlogk8saudit_contract.ExtractK8sAuditLog(ctx, l.NodeReader)
	if fieldSet.IsDryRun {
		return &commonlogk8saudit_contract.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: g.prevRevisionReader,
		}, nil
	}
	if fieldSet.IsTruncated {
		g.prevRevisionReader = nil
		return &commonlogk8saudit_contract.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: nil,
		}, nil
	}
	currentBodyReader := fieldSet.Response
	partial := false
	if currentBodyReader == nil {
		currentBodyReader = fieldSet.Request
		partial = true
	} else {
		apiVersion := currentBodyReader.ReadStringOrDefault(pathAPIVersion, "")
		kind := currentBodyReader.ReadStringOrDefault(pathKind, "")
		if apiVersion == "v1" && kind == "Status" {
			currentBodyReader = fieldSet.Request
			partial = true
		}
	}
	// request or response may contain its proto type as @type. Removing it because its not a k8s field.
	if currentBodyReader != nil {
		currentBodyReader = structured.NewNodeReader(structured.NewFieldFilterNode(currentBodyReader.Node, []string{"@type"}))
	}
	if currentBodyReader == nil {
		return &commonlogk8saudit_contract.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: nil,
		}, nil
	}

	if fieldSet.Verb == commonlogk8saudit_contract.VerbDeleteCollection {
		items, err := currentBodyReader.GetReader(pathItems)
		if err != nil {
			return &commonlogk8saudit_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
		found := false
		items.Children()(func(key structured.NodeChildrenKey, item structured.NodeReader) bool {
			name := item.ReadStringOrDefault(pathMetadataName, "")
			if name == g.resourceName {
				found = true
				bodyReader, err := constructResourceBodyFromListItem(&item, g.prevRevisionReader)
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to construct resource body from list item: %v", err))
				} else {
					currentBodyReader = bodyReader
				}
				return false
			}
			return true
		})
		if !found {
			return &commonlogk8saudit_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: nil,
			}, nil
		}
	}

	if fieldSet.Verb == commonlogk8saudit_contract.VerbPatch && partial {
		mergeConfigResolver := g.mergeConfigRegistry.Get(fieldSet.APIVersion, commonlogk8saudit_contract.GetSingularKindName(fieldSet.PluralKind))
		mergedNode, err := structured.MergeNode(g.prevRevisionReader.Node, currentBodyReader.Node, structured.MergeConfiguration{
			MergeMapOrderStrategy:    &structured.DefaultMergeMapOrderStrategy{},
			ArrayMergeConfigResolver: mergeConfigResolver,
		})
		var mergedNodeReader *structured.NodeReader
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("failed to merge resource body\n%s", err.Error()))
			return &commonlogk8saudit_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		} else {
			lazyMergedNode, err := structured.NewLazyJSONNode(mergedNode)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("failed to convert merged node to lazy JSON node: %v", err))
				mergedNodeReader = structured.NewNodeReader(structured.WithKeyOrder(mergedNode, k8s.K8sManifestKeyOrder...))
			} else {
				mergedNodeReader = structured.NewNodeReader(structured.WithKeyOrder(lazyMergedNode, k8s.K8sManifestKeyOrder...))
			}
			g.prevRevisionReader = mergedNodeReader
			return &commonlogk8saudit_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
	} else {
		apiVersion := currentBodyReader.ReadStringOrDefault(pathAPIVersion, "")
		kind := currentBodyReader.ReadStringOrDefault(pathKind, "")
		if apiVersion == "meta.k8s.io/__internal" && kind == "DeleteOptions" {
			return &commonlogk8saudit_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
		g.prevRevisionReader = currentBodyReader
		return &commonlogk8saudit_contract.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: g.prevRevisionReader,
		}, nil
	}
}

// constructResourceBodyFromListItem constructs a complete resource manifest NodeReader from a list item,
// injecting apiVersion and kind from the previous revision if they are not present in the item.
func constructResourceBodyFromListItem(item *structured.NodeReader, prevRevision *structured.NodeReader) (*structured.NodeReader, error) {
	if item == nil {
		return nil, fmt.Errorf("item reader cannot be nil")
	}

	var prevAPIVersion, prevKind string
	if prevRevision != nil {
		prevAPIVersion = prevRevision.ReadStringOrDefault(pathAPIVersion, "")
		prevKind = prevRevision.ReadStringOrDefault(pathKind, "")
	}

	rawJSON, err := item.Serialize(structured.EmptyFieldPath, &structured.JSONNodeSerializer{})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize resource body to json: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	hasField := false
	if prevAPIVersion != "" {
		buf.WriteString(`"apiVersion":`)
		b, err := json.Marshal(prevAPIVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal apiVersion: %w", err)
		}
		buf.Write(b)
		hasField = true
	}
	if prevKind != "" {
		if hasField {
			buf.WriteByte(',')
		}
		buf.WriteString(`"kind":`)
		b, err := json.Marshal(prevKind)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal kind: %w", err)
		}
		buf.Write(b)
		hasField = true
	}
	trimmed := bytes.TrimSpace(rawJSON)
	if len(trimmed) >= 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		inner := bytes.TrimSpace(trimmed[1 : len(trimmed)-1])
		if len(inner) > 0 {
			if hasField {
				buf.WriteByte(',')
			}
			buf.Write(inner)
		}
	}
	buf.WriteByte('}')
	lazyNode := structured.NewLazyJSONNodeFromBytes(buf.Bytes())
	return structured.NewNodeReader(structured.WithKeyOrder(lazyNode, k8s.K8sManifestKeyOrder...)), nil
}
