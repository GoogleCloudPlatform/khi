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

package commonlogk8sauditv2_impl

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"golang.org/x/sync/errgroup"
)

var bodyPlaceholderForMetadataLevelAuditLog = "# Resource data is unavailable. Audit logs for this resource is recorded at metadata level."

// ManifestGeneratorTask is the task to generate manifest from k8s audit logs.
var ManifestGeneratorTask = inspectiontaskbase.NewProgressReportableInspectionTask(commonlogk8sauditv2_contract.ManifestGeneratorTaskID, []taskid.UntypedTaskReference{
	commonlogk8sauditv2_contract.ChangeTargetGrouperTaskID.Ref(),
	googlecloudk8scommon_contract.K8sResourceMergeConfigTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (commonlogk8sauditv2_contract.ResourceManifestLogGroupMap, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return map[string]*commonlogk8sauditv2_contract.ResourceManifestLogGroup{}, nil
	}

	logGroups := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ChangeTargetGrouperTaskID.Ref())
	mergeConfigRegistry := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.K8sResourceMergeConfigTaskID.Ref())
	result := commonlogk8sauditv2_contract.ResourceManifestLogGroupMap{}
	resultLock := sync.Mutex{}

	grp, childCtx := errgroup.WithContext(ctx)
	grp.SetLimit(runtime.GOMAXPROCS(0))

	for path, group := range logGroups {
		path := path
		group := group
		grp.Go(func() error {
			resourceLogs := []*commonlogk8sauditv2_contract.ResourceManifestLog{}
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
			result[path] = &commonlogk8sauditv2_contract.ResourceManifestLogGroup{
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
	// prevRevisionBody is the body of the previous revision.
	prevRevisionBody string
	// resourceName is the name of the resource.
	resourceName string
}

// Process processes the log to generate manifest.
func (g *groupManifestGenerator) Process(ctx context.Context, l *log.Log) (*commonlogk8sauditv2_contract.ResourceManifestLog, error) {
	if g.prevRevisionReader == nil {
		g.prevRevisionReader = structured.NewNodeReader(structured.NewEmptyMapNode())
	}
	fieldSet := log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{})
	currentBodyReader := fieldSet.Response
	partial := false
	if currentBodyReader == nil {
		currentBodyReader = fieldSet.Request
		partial = true
	} else {
		apiVersion := currentBodyReader.ReadStringOrDefault("apiVersion", "")
		kind := currentBodyReader.ReadStringOrDefault("kind", "")
		if apiVersion == "v1" && kind == "Status" {
			currentBodyReader = fieldSet.Request
			partial = true
		}
	}

	if currentBodyReader == nil {
		return &commonlogk8sauditv2_contract.ResourceManifestLog{
			Log:                l,
			ResourceBodyYAML:   bodyPlaceholderForMetadataLevelAuditLog,
			ResourceBodyReader: nil,
		}, nil
	}

	if fieldSet.K8sOperation.Verb == enum.RevisionVerbDeleteCollection {
		items, err := currentBodyReader.GetReader("items")
		if err != nil {
			return &commonlogk8sauditv2_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyYAML:   g.prevRevisionBody,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
		found := false
		for _, item := range items.Children() {
			name := item.ReadStringOrDefault("metadata.name", "")
			if name == g.resourceName {
				found = true
				// XXList omits apiVersion and kind in its item. Generate a reader with the field.
				rawYAML, err := item.Serialize("", &structured.YAMLNodeSerializer{})
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to serialize resource body to yaml\n%s", err.Error()))
				}
				var prevAPIVersion, prevKind string
				if g.prevRevisionReader != nil {
					prevAPIVersion = g.prevRevisionReader.ReadStringOrDefault("apiVersion", "")
					prevKind = g.prevRevisionReader.ReadStringOrDefault("kind", "")
				}
				fullYAMLStr := string(rawYAML)
				if prevAPIVersion != "" && prevKind != "" {
					fullYAMLStr = fmt.Sprintf(`apiVersion: %s
kind: %s
%s`, prevAPIVersion, prevKind, fullYAMLStr)
				}

				rootNode, err := structured.FromYAML(fullYAMLStr)
				if err != nil {
					slog.WarnContext(ctx, fmt.Sprintf("failed to serialize resource body to yaml after constructing full yaml\n%s", err.Error()))
				} else {
					currentBodyReader = structured.NewNodeReader(rootNode)
				}
				break
			}
		}
		if !found {
			return &commonlogk8sauditv2_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyYAML:   bodyPlaceholderForMetadataLevelAuditLog,
				ResourceBodyReader: nil,
			}, nil
		}
	}

	currentRevisionBodyRaw, err := currentBodyReader.Serialize("", &structured.YAMLNodeSerializer{})
	if err != nil {
		slog.WarnContext(ctx, fmt.Sprintf("failed to serialize resource body to yaml\n%s", err.Error()))
	}
	currentRevisionBody := string(currentRevisionBodyRaw)
	currentRevisionBody = removeAtType(currentRevisionBody)

	if fieldSet.K8sOperation.Verb == enum.RevisionVerbPatch && partial {
		op := fieldSet.K8sOperation
		mergeConfigResolver := g.mergeConfigRegistry.Get(op.APIVersion, op.GetSingularKindName())
		mergedNode, err := structured.MergeNode(g.prevRevisionReader.Node, currentBodyReader.Node, structured.MergeConfiguration{
			MergeMapOrderStrategy:    &structured.DefaultMergeMapOrderStrategy{},
			ArrayMergeConfigResolver: mergeConfigResolver,
		})
		var mergedNodeReader *structured.NodeReader
		var mergedYAML string
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("failed to merge resource body\n%s", err.Error()))
			return &commonlogk8sauditv2_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyYAML:   g.prevRevisionBody,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		} else {
			mergedNodeReader = structured.NewNodeReader(mergedNode)
			mergedYAMLRaw, err := mergedNodeReader.Serialize("", &structured.YAMLNodeSerializer{})
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("failed to read the merged resource body\n%s", err.Error()))
			}
			mergedYAML = removeAtType(string(mergedYAMLRaw))
			g.prevRevisionBody = mergedYAML
			g.prevRevisionReader = mergedNodeReader
			return &commonlogk8sauditv2_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyYAML:   g.prevRevisionBody,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
	} else {
		apiVersion := currentBodyReader.ReadStringOrDefault("apiVersion", "")
		kind := currentBodyReader.ReadStringOrDefault("kind", "")
		if apiVersion == "meta.k8s.io/__internal" && kind == "DeleteOptions" {
			return &commonlogk8sauditv2_contract.ResourceManifestLog{
				Log:                l,
				ResourceBodyYAML:   g.prevRevisionBody,
				ResourceBodyReader: g.prevRevisionReader,
			}, nil
		}
		g.prevRevisionBody = currentRevisionBody
		g.prevRevisionReader = currentBodyReader
		return &commonlogk8sauditv2_contract.ResourceManifestLog{
			Log:                l,
			ResourceBodyYAML:   g.prevRevisionBody,
			ResourceBodyReader: g.prevRevisionReader,
		}, nil
	}
}

// removeAtType removes @type in response or request payload.
func removeAtType(yamlString string) string {
	lines := strings.Split(yamlString, "\n")
	var result []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "'@type'") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}
