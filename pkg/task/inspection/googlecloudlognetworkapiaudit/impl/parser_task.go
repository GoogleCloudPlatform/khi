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

package googlecloudlognetworkapiaudit_impl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlognetworkapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlognetworkapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"gopkg.in/yaml.v3"
)

// FieldSetReaderTask is a task that reads fieldsets needed for GCE network API logs.
var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlognetworkapiaudit_contract.FieldSetReaderTaskID, googlecloudlognetworkapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
	&googlecloudcommon_contract.GCPDefaultSeverityFieldSetReader{},
})

type networkAPILogIngester struct{}

// RawLogTask returns the task reference that provides the raw logs to ingest.
func (i *networkAPILogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlognetworkapiaudit_contract.FieldSetReaderTaskID.Ref()
}

// Dependencies returns additional task dependencies of the ingester.
func (i *networkAPILogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog parses raw log entry and manually populates the LogChangeSet.
func (i *networkAPILogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}

	cs.SetLogType(googlecloudlognetworkapiaudit_contract.LogTypeNetworkAPI)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	if auditFieldSet, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{}); err == nil {
		switch {
		case auditFieldSet.Starting():
			cs.SetSummary(fmt.Sprintf("%s Started", auditFieldSet.MethodName))
		case auditFieldSet.Ending():
			cs.SetSummary(fmt.Sprintf("%s Finished", auditFieldSet.MethodName))
		default:
			cs.SetSummary(auditFieldSet.MethodName)
		}
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*networkAPILogIngester)(nil)

// LogIngesterTask is the task id to finalize the logs to be included in the final output.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(googlecloudlognetworkapiaudit_contract.LogIngesterTaskID, &networkAPILogIngester{})

// LogGrouperTask groups logs by the NEG resource name.
var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlognetworkapiaudit_contract.LogGrouperTaskID, googlecloudlognetworkapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return audit.ResourceName
	},
)

type negAttachOrDetachRequestEndpoint struct {
	Instance  string `yaml:"instance"`
	IpAddress string `yaml:"ipAddress"`
	Port      string `yaml:"port"`
}

type negAttachOrDetachRequest struct {
	NetworkEndpoints []*negAttachOrDetachRequestEndpoint `yaml:"networkEndpoints"`
}

type perNEGHistoryModificationStatus struct {
	LastNegAttachRequest *negAttachOrDetachRequest
}

type networkAPITimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[*perNEGHistoryModificationStatus]
}

// LogIngesterTask is the task reference that provides the ingested logs.
func (m *networkAPITimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlognetworkapiaudit_contract.LogIngesterTaskID.Ref()
}

// Dependencies are the additional references used in timeline mapper.
func (m *networkAPITimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
		googlecloudk8scommon_contract.NEGNamesDiscoveryTaskID.Ref(),
		commonlogk8saudit_contract.IPLeaseHistoryInventoryTaskID.Ref(),
		googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref(),
	}
}

// GroupedLogTask returns a reference to the task that provides the grouped logs.
func (m *networkAPITimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlognetworkapiaudit_contract.LogGrouperTaskID.Ref()
}

// ProcessLogByGroup maps the NEG audit log to resource timelines as state revisions.
func (m *networkAPITimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData *perNEGHistoryModificationStatus) (*khifilev6.TimelineChangeSet, *perNEGHistoryModificationStatus, error) {
	commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
	auditFieldSet := log.MustGetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if prevGroupData == nil {
		prevGroupData = &perNEGHistoryModificationStatus{}
	}

	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())
	negs := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.NEGNamesInventoryTaskID.Ref())
	var negResourcePath *khifilev6.TimelinePath
	negName := getNegNameFromResourceName(auditFieldSet.ResourceName)

	if negResource, found := negs[negName]; found {
		negResourcePath = googlecloudlognetworkapiaudit_contract.MustNEGTimeline(ctx, clusterIdentity.ClusterName, negResource.Namespace, negName)
	} else {
		negResourcePath = googlecloudlognetworkapiaudit_contract.MustNEGTimeline(ctx, clusterIdentity.ClusterName, "unknown", negName)
	}

	cs := khifilev6.NewTimelineChangeSet(l)

	// Add operation subresource under neg resource.
	var negOperationPath *khifilev6.TimelinePath
	if auditFieldSet.ImmediateOperation() {
		negOperationPath = negResourcePath
	} else {
		negOperationPath = googlecloudlognetworkapiaudit_contract.MustNEGOperationTimeline(ctx, negResourcePath, auditFieldSet.MethodName, auditFieldSet.OperationID)
	}

	if auditFieldSet.ImmediateOperation() {
		cs.AddEvent(negOperationPath)
	} else {
		var state *pb.RevisionState
		var verb *pb.Verb
		if auditFieldSet.Ending() {
			state = googlecloudcommon_contract.RevisionStateOperationFinished
			verb = googlecloudcommon_contract.VerbOperationFinish
		} else {
			state = googlecloudcommon_contract.RevisionStateOperationStarted
			verb = googlecloudcommon_contract.VerbOperationStart
		}
		requestBody, _ := auditFieldSet.RequestString()
		var bodyNode structured.Node
		if requestBody != "" {
			if node, err := structured.FromYAML(requestBody); err == nil {
				bodyNode = node
			}
		}
		cs.AddRevision(negOperationPath, &khifilev6.StagingRevision{
			ChangedTime:  commonFieldSet.Timestamp,
			ResourceBody: bodyNode,
			VerbType:     verb,
			StateType:    state,
			Principal:    auditFieldSet.PrincipalEmail,
		})
	}

	ipLeases := coretask.GetTaskResult(ctx, commonlogk8saudit_contract.IPLeaseHistoryInventoryTaskID.Ref())
	// Add neg subresource under resources with the same IP of the endpoint.
	shortMethodName := getShortMethodNameFromMethodName(auditFieldSet.MethodName)
	var negRequest *negAttachOrDetachRequest
	var verb *pb.Verb
	var state *pb.RevisionState
	switch shortMethodName {
	case "attachNetworkEndpoints":
		if auditFieldSet.Starting() {
			// Operation starting log only contain its request(IP data), but it should be marked as ready when the last log coming.
			var err error
			request, err := parseNEGAttachOrDetachRequest(l)
			if err != nil {
				return nil, prevGroupData, err
			}
			prevGroupData.LastNegAttachRequest = request // Save the neg attach request in the per group status, and it will be consumed in the next ending operation log.
			break
		}
		negRequest = prevGroupData.LastNegAttachRequest
		prevGroupData.LastNegAttachRequest = nil
		verb = commonlogk8saudit_contract.VerbReady
		state = commonlogk8saudit_contract.RevisionStateConditionTrue
	case "detachNetworkEndpoints":
		if auditFieldSet.Ending() {
			break
		}
		var err error
		negRequest, err = parseNEGAttachOrDetachRequest(l)
		if err != nil {
			return nil, prevGroupData, err
		}
		verb = commonlogk8saudit_contract.VerbNonReady
		state = commonlogk8saudit_contract.RevisionStateConditionFalse
	}

	if negRequest != nil {
		negToBS := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref())
		for _, endpoint := range negRequest.NetworkEndpoints {
			lease, err := ipLeases.GetResourceLeaseHolderAt(endpoint.IpAddress, commonFieldSet.Timestamp)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to identify the holder of the IP %s.\n This might be because the IP holder resource wasn't updated during the log period ", endpoint.IpAddress))
				continue
			}
			holder := lease.Holder
			if holder.Kind == "pod" {
				clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, clusterIdentity.ClusterName)
				apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
				kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "pod")
				nsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, holder.Namespace)
				podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, nsPath, holder.Name)

				negSubresourcePath := googlecloudlognetworkapiaudit_contract.MustNEGUnderResourceTimeline(ctx, podPath, negName)
				cs.AddRevision(negSubresourcePath, &khifilev6.StagingRevision{
					ChangedTime: commonFieldSet.Timestamp,
					VerbType:    verb,
					StateType:   state,
					Principal:   auditFieldSet.PrincipalEmail,
				})

				if bsName, found := negToBS[negName]; found {
					// BackendService is usually global in the context of gsmrsvd backends.
					bsPath := googlecloudlognetworkapiaudit_contract.MustGCPResourceTimeline(ctx, clusterIdentity.ProjectID, "backendServices", bsName)
					negSubresourcePath := googlecloudlognetworkapiaudit_contract.MustNEGUnderResourceTimeline(ctx, bsPath, holder.Name)
					cs.AddRevision(negSubresourcePath, &khifilev6.StagingRevision{
						ChangedTime: commonFieldSet.Timestamp,
						VerbType:    verb,
						StateType:   state,
						Principal:   auditFieldSet.PrincipalEmail,
					})
				}
			}
		}
	}

	return cs, prevGroupData, nil
}

var _ inspectiontaskbase.LogToTimelineMapperV2[*perNEGHistoryModificationStatus] = (*networkAPITimelineMapper)(nil)

// LogToTimelineMapperTask registers the mapper to resolve network status in timeline.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTaskV2(googlecloudlognetworkapiaudit_contract.LogToTimelineMapperTaskID, &networkAPITimelineMapper{},
	inspectioncore_contract.FeatureTaskLabelV2(`GCE Network Logs`,
		`Gather GCE Network API logs to visualize statuses of Network Endpoint Groups(NEG)`,
		7000,
		true,
	),
)

func getNegNameFromResourceName(resourceName string) string {
	lastSlashIndex := strings.LastIndex(resourceName, "/")
	if lastSlashIndex == -1 {
		return resourceName
	}
	return resourceName[lastSlashIndex+1:]
}

func getShortMethodNameFromMethodName(methodName string) string {
	lastDotIndex := strings.LastIndex(methodName, ".")
	if lastDotIndex == -1 {
		return methodName
	}
	return methodName[lastDotIndex+1:]
}

func parseNEGAttachOrDetachRequest(l *log.Log) (*negAttachOrDetachRequest, error) {
	auditFieldSet := log.MustGetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	requestBody, err := auditFieldSet.RequestString()
	if err != nil {
		return nil, err
	}
	var negRequest negAttachOrDetachRequest
	err = yaml.Unmarshal([]byte(requestBody), &negRequest)
	if err != nil {
		return nil, err
	}
	return &negRequest, nil
}
