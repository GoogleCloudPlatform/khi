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

package k8saudit_impl

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
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"golang.org/x/sync/errgroup"
)

var (
	pathAPIVersion        = structured.CompileFieldPath("apiVersion")
	pathKind              = structured.CompileFieldPath("kind")
	pathItems             = structured.CompileFieldPath("items")
	pathMetadataName      = structured.CompileFieldPath("metadata.name")
	pathMetadataNamespace = structured.CompileFieldPath("metadata.namespace")
)

// ManifestGeneratorTask is the task to generate manifest from k8s audit logs.
var ManifestGeneratorTask = inspectiontaskbase.NewProgressReportableInspectionTask(k8saudit.ManifestGeneratorTaskID, []coretask.Dependency{
	k8saudit.ChangeTargetGrouperTaskID.Ref(),
	k8saudit.K8sResourceMergeConfigTaskID.Ref(),
	k8saudit.K8sAuditLogExtractorRef.Ref(coretask.FromActiveGraph),
	k8saudit.InitialResourceStateProviderRef,
}, func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (k8saudit.ResourceManifestLogGroupMap, error) {
	if taskMode == inspectioncore.TaskModeDryRun {
		return map[string]*k8saudit.ResourceManifestLogGroup{}, nil
	}

	logGroups := coretask.GetTaskResult(ctx, k8saudit.ChangeTargetGrouperTaskID.Ref())
	mergeConfigRegistry := coretask.GetTaskResult(ctx, k8saudit.K8sResourceMergeConfigTaskID.Ref())
	initialStateProvider := coretask.GetTaskResult(ctx, k8saudit.InitialResourceStateProviderRef)
	result := k8saudit.ResourceManifestLogGroupMap{}
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
	blockStore := structured.NewDefaultLazyJSONBlockStore()

	for path, group := range logGroups {
		grp.Go(func() error {
			defer doneGroupCount.Add(1)
			resourceLogs := []*k8saudit.ResourceManifestLog{}
			generator := groupManifestGenerator{
				mergeConfigRegistry: mergeConfigRegistry,
				resourceName:        group.Resource.Name,
				blockStore:          blockStore,
			}
			// Merging the first partial patch onto the state observed before the logs keeps the rendered
			// manifest complete instead of showing only the patched fields.
			if initialBody, found := initialStateProvider.InitialResourceState(group.Resource); found {
				generator.prevRevisionReader = initialBody
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
			result[path] = &k8saudit.ResourceManifestLogGroup{
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
	// blockStore is the store for compressing manifest nodes.
	blockStore *structured.LazyJSONBlockStore
}

// Process processes the log to generate manifest.
func (g *groupManifestGenerator) Process(ctx context.Context, l *log.Log) (*k8saudit.ResourceManifestLog, error) {
	fieldSet, _ := k8saudit.ExtractK8sAuditLog(ctx, l.NodeReader)
	if fieldSet.IsDryRun {
		return &k8saudit.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: g.prevRevisionReader,
		}, nil
	}
	if fieldSet.IsTruncated {
		if g.prevRevisionReader != nil {
			g.prevRevisionReader = extractResourceIdentity(g.blockStore, g.prevRevisionReader)
		}
		return &k8saudit.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: g.prevRevisionReader,
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
		return &k8saudit.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: nil,
		}, nil
	}

	if fieldSet.Verb == k8saudit.VerbDeleteCollection {
		items, err := currentBodyReader.GetReader(pathItems)
		if err != nil {
			return &k8saudit.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
		found := false
		items.Children()(func(key structured.NodeChildrenKey, item structured.NodeReader) bool {
			name := item.ReadStringOrDefault(pathMetadataName, "")
			if name == g.resourceName {
				found = true
				bodyReader, err := constructResourceBodyFromListItem(g.blockStore, &item, g.prevRevisionReader)
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
			return &k8saudit.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: nil,
			}, nil
		}
	}

	if fieldSet.Verb == k8saudit.VerbPatch && partial {
		if g.prevRevisionReader == nil {
			g.prevRevisionReader = structured.NewNodeReader(structured.NewEmptyMapNode())
		}
		mergeConfigResolver := g.mergeConfigRegistry.Get(fieldSet.APIVersion, k8saudit.GetSingularKindName(fieldSet.PluralKind))
		mergedNode, err := structured.MergeNode(g.prevRevisionReader.Node, currentBodyReader.Node, structured.MergeConfiguration{
			MergeMapOrderStrategy:    &structured.DefaultMergeMapOrderStrategy{},
			ArrayMergeConfigResolver: mergeConfigResolver,
		})
		var mergedNodeReader *structured.NodeReader
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("failed to merge resource body\n%s", err.Error()))
			return &k8saudit.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		} else {
			lazyMergedNode, err := structured.NewLazyJSONNode(g.blockStore, mergedNode)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("failed to convert merged node to lazy JSON node: %v", err))
				mergedNodeReader = structured.NewNodeReader(structured.WithKeyOrder(mergedNode, k8s.K8sManifestKeyOrder...))
			} else {
				mergedNodeReader = structured.NewNodeReader(structured.WithKeyOrder(lazyMergedNode, k8s.K8sManifestKeyOrder...))
			}
			g.prevRevisionReader = mergedNodeReader
			return &k8saudit.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
	} else {
		apiVersion := currentBodyReader.ReadStringOrDefault(pathAPIVersion, "")
		kind := currentBodyReader.ReadStringOrDefault(pathKind, "")
		if apiVersion == "meta.k8s.io/__internal" && kind == "DeleteOptions" {
			return &k8saudit.ResourceManifestLog{
				Log:                l,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
		g.prevRevisionReader = currentBodyReader
		return &k8saudit.ResourceManifestLog{
			Log:                l,
			ResourceBodyReader: g.prevRevisionReader,
		}, nil
	}
}

// constructResourceBodyFromListItem constructs a complete resource manifest NodeReader from a list item,
// injecting apiVersion and kind from the previous revision if they are not present in the item.
func constructResourceBodyFromListItem(store *structured.LazyJSONBlockStore, item *structured.NodeReader, prevRevision *structured.NodeReader) (*structured.NodeReader, error) {
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
	lazyNode := structured.NewLazyJSONNodeFromBytes(store, buf.Bytes())
	return structured.NewNodeReader(structured.WithKeyOrder(lazyNode, k8s.K8sManifestKeyOrder...)), nil
}

// extractResourceIdentity extracts immutable resource identity metadata from the previous revision.
// It preserves apiVersion, kind, metadata.name, metadata.namespace, metadata.uid, and metadata.creationTimestamp,
// returning a NodeReader containing only those fields. If prevRevision contains no identity fields,
// it returns an empty map NodeReader.
func extractResourceIdentity(store *structured.LazyJSONBlockStore, prevRevision *structured.NodeReader) *structured.NodeReader {
	apiVersion := prevRevision.ReadStringOrDefault(pathAPIVersion, "")
	kind := prevRevision.ReadStringOrDefault(pathKind, "")
	name := prevRevision.ReadStringOrDefault(pathMetadataName, "")
	namespace := prevRevision.ReadStringOrDefault(pathMetadataNamespace, "")
	uid := prevRevision.ReadStringOrDefault(pathMetadataUID, "")
	creationTimestamp := prevRevision.ReadStringOrDefault(pathMetadataCreationTimestamp, "")

	if apiVersion == "" && kind == "" && name == "" && namespace == "" && uid == "" && creationTimestamp == "" {
		return structured.NewNodeReader(structured.NewEmptyMapNode())
	}

	metadata := make(map[string]any)
	if name != "" {
		metadata["name"] = name
	}
	if namespace != "" {
		metadata["namespace"] = namespace
	}
	if uid != "" {
		metadata["uid"] = uid
	}
	if creationTimestamp != "" {
		metadata["creationTimestamp"] = creationTimestamp
	}

	root := make(map[string]any)
	if apiVersion != "" {
		root["apiVersion"] = apiVersion
	}
	if kind != "" {
		root["kind"] = kind
	}
	if len(metadata) > 0 {
		root["metadata"] = metadata
	}

	jsonBytes, err := json.Marshal(root)
	if err != nil {
		return structured.NewNodeReader(structured.NewEmptyMapNode())
	}

	lazyNode := structured.NewLazyJSONNodeFromBytes(store, jsonBytes)
	return structured.NewNodeReader(structured.WithKeyOrder(lazyNode, k8s.K8sManifestKeyOrder...))
}
