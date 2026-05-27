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

package googlecloudlogk8scontrolplane_impl

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
)

func kindToKLogFieldPair(apiVersion string, kind string, klogField string, isNamespaced bool) *googlecloudlogk8scontrolplane_contract.KindToKLogFieldPairData {
	return &googlecloudlogk8scontrolplane_contract.KindToKLogFieldPairData{
		APIVersion:   apiVersion,
		KindName:     kind,
		KLogField:    klogField,
		IsNamespaced: isNamespaced,
	}
}

var ControllerManagerFilterTask = inspectiontaskbase.NewLogFilterTask(
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogFilterTaskID,
	googlecloudlogk8scontrolplane_contract.CommonFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
		if err != nil {
			return false
		}
		return componentFieldSet.ComponentParserType() == googlecloudlogk8scontrolplane_contract.ComponentParserTypeControllerManager
	},
)

var ControllerManagerLogFieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlogk8scontrolplane_contract.ControllerManagerLogFieldSetReaderTaskID,
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogFilterTaskID.Ref(),
	[]log.FieldSetReader{
		&googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSetReader{},
		&googlecloudlogk8scontrolplane_contract.K8sControllerManagerComponentFieldSetReader{
			WellKnownSourceLocationToControllerMap: map[string]string{
				"namespace_controller.go":      "namespace-controller",
				"resource_quota_controller.go": "resourcequota-controller",
				"requestheader_controller.go":  "requestheader-controller",
				"pv_protection_controller.go":  "persistentvolume-protection-controller",
			},
			WellKnownKindToKLogFieldPairs: []*googlecloudlogk8scontrolplane_contract.KindToKLogFieldPairData{
				kindToKLogFieldPair("apps/v1", "deployment", "deployment", true),
				kindToKLogFieldPair("apps/v1", "replicaset", "replicaSet", true),
				kindToKLogFieldPair("apps/v1", "statefulset", "statefulSet", true),
				kindToKLogFieldPair("apps/v1", "daemonset", "daemonSet", true),
				kindToKLogFieldPair("batch/v1", "cronjob", "cronjob", true),
				kindToKLogFieldPair("batch/v1", "job", "job", true),
				kindToKLogFieldPair("policy/v1", "poddisruptionbudget", "podDisruptionBudget", true),
				kindToKLogFieldPair("certificates.k8s.io/v1", "certificatesigningrequest", "csr", false),
				kindToKLogFieldPair("core/v1", "persistentvolumeclaim", "PVC", true),
				kindToKLogFieldPair("core/v1", "persistentvolume", "volumeName", false),
				kindToKLogFieldPair("core/v1", "service", "service", true),
				kindToKLogFieldPair("core/v1", "node", "node", false),
				kindToKLogFieldPair("core/v1", "pod", "pod", true),
				kindToKLogFieldPair("core/v1", "namespace", "namespace", false),
			},
			KLogParser: logutil.NewKLogTextParser(false),
		},
	},
)

var ControllerManagerGrouperTask = inspectiontaskbase.NewLogGrouperTask(
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogGrouperTaskID,
	googlecloudlogk8scontrolplane_contract.ControllerManagerLogFieldSetReaderTaskID.Ref(),
	func(ctx context.Context, log *log.Log) string {
		return "" // No grouping needed
	},
)

var ControllerManagerLogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2[struct{}](googlecloudlogk8scontrolplane_contract.ControllerManagerLogToTimelineMapperTaskID, &ControllerManagerTimelineMapper{})

// ControllerManagerTimelineMapper maps controller manager logs to timeline paths.
type ControllerManagerTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
	uidPrefixTokenCandidates []rune
}

// Dependencies implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *ControllerManagerTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *ControllerManagerTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlogk8scontrolplane_contract.ControllerManagerLogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *ControllerManagerTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapperV2.
func (o *ControllerManagerTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	finder := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref())
	componentFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	commonMainMessage, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}
	controllerManagerFieldSet, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControllerManagerComponentFieldSet{})
	if err != nil {
		return nil, struct{}{}, err
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	resources := patternfinder.FindAllWithStarterRunes(commonMainMessage.Message, finder, false, o.uidPrefixTokenCandidates...)
	writtenResourcePaths := map[uint32]struct{}{}

	compTimeline := resolveControllerManagerTimelinePath(ctx, componentFieldSet.ClusterName, controllerManagerFieldSet.Controller)
	cs.AddEvent(compTimeline)

	for _, resourcePath := range controllerManagerFieldSet.AssociatedResources {
		if tPath, err := resolveTimelinePathFromRawString(ctx, componentFieldSet.ClusterName, resourcePath.Path); err == nil {
			cs.AddEvent(tPath)
			writtenResourcePaths[tPath.ID] = struct{}{}
		}
	}

	for _, resource := range resources {
		path := resource.Value.ResourcePathString()
		if tPath, err := resolveTimelinePathFromRawString(ctx, componentFieldSet.ClusterName, path); err == nil {
			if _, ok := writtenResourcePaths[tPath.ID]; ok {
				continue
			}
			cs.AddEvent(tPath)
			writtenResourcePaths[tPath.ID] = struct{}{}
		}
	}

	return cs, struct{}{}, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[struct{}] = (*ControllerManagerTimelineMapper)(nil)

// resolveTimelinePathFromRawString converts a resource path string formatted as APIVersion#Kind#Namespace#Name into a *TimelinePath.
func resolveTimelinePathFromRawString(ctx context.Context, clusterName string, pathStr string) (*khifilev6.TimelinePath, error) {
	parts := strings.Split(pathStr, "#")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid resource path string: %s", pathStr)
	}
	apiVersion := parts[0]
	if !strings.Contains(apiVersion, "/") {
		if apiVersion == "v1" {
			apiVersion = "core/v1"
		} else {
			apiVersion = "core/" + apiVersion
		}
	}
	kind := strings.ToLower(parts[1])
	namespace := parts[2]
	name := parts[3]

	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterName)
	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, apiVersion)
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, kind)

	var resourceTimeline *khifilev6.TimelinePath
	if namespace == "cluster-scope" {
		resourceTimeline = commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, kindTimeline, name)
	} else {
		namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, namespace)
		resourceTimeline = commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, name)
	}

	if len(parts) >= 5 {
		subresource := parts[4]
		return commonlogk8saudit_contract.MustK8sSubresourceTimeline(ctx, resourceTimeline, subresource), nil
	}

	return resourceTimeline, nil
}

// resolveControllerManagerTimelinePath resolves a controller manager component timeline path.
func resolveControllerManagerTimelinePath(ctx context.Context, clusterName string, controllerName string) *khifilev6.TimelinePath {
	if controllerName == "" {
		return MustControlPlaneComponentTimeline(ctx, clusterName, "controller-manager")
	}
	return MustControlPlaneComponentTimeline(ctx, clusterName, fmt.Sprintf("%s(controller-manager)", controllerName))
}
