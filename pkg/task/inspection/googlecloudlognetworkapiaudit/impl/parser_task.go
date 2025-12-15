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

package googlecloudlognetworkapiaudit_impl

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlognetworkapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlognetworkapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"gopkg.in/yaml.v3"
)

var FieldSetReaderTask = inspectiontaskbase.NewFieldSetReadTask(googlecloudlognetworkapiaudit_contract.FieldSetReaderTaskID, googlecloudlognetworkapiaudit_contract.ListLogEntriesTaskID.Ref(), []log.FieldSetReader{
	&googlecloudcommon_contract.GCPOperationAuditLogFieldSetReader{},
})

var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(googlecloudlognetworkapiaudit_contract.LogIngesterTaskID, googlecloudlognetworkapiaudit_contract.ListLogEntriesTaskID.Ref())

var LogGrouperTask = inspectiontaskbase.NewLogGrouperTask(googlecloudlognetworkapiaudit_contract.LogGrouperTaskID, googlecloudlognetworkapiaudit_contract.FieldSetReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) string {
		// Group logs by the NEG resource name.
		audit, err := log.GetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
		if err != nil {
			return "unknown"
		}
		return audit.ResourceName
	},
)

var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask[*perNEGHistoryModificationStatus](googlecloudlognetworkapiaudit_contract.LogToTimelineMapperTaskID, &networkAPILogToTimelineMapperTaskSetting{},
	inspectioncore_contract.FeatureTaskLabel(`GCE Network Logs`,
		`Gather GCE Network API logs to visualize statuses of Network Endpoint Groups(NEG)`,
		enum.LogTypeNetworkAPI,
		7000,
		true,
		googlecloudinspectiontypegroup_contract.GKEBasedClusterInspectionTypes...),
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

type networkAPILogToTimelineMapperTaskSetting struct{}

// Dependencies implements inspectiontaskbase.LogToTimelineMapper.
func (n *networkAPILogToTimelineMapperTaskSetting) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.NEGNamesDiscoveryTaskID.Ref(),
		commonlogk8sauditv2_contract.IPLeaseHistoryInventoryTaskID.Ref(),
	}
}

// GroupedLogTask implements inspectiontaskbase.LogToTimelineMapper.
func (n *networkAPILogToTimelineMapperTaskSetting) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudlognetworkapiaudit_contract.LogGrouperTaskID.Ref()
}

// LogIngesterTask implements inspectiontaskbase.LogToTimelineMapper.
func (n *networkAPILogToTimelineMapperTaskSetting) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlognetworkapiaudit_contract.LogIngesterTaskID.Ref()
}

// ProcessLogByGroup implements inspectiontaskbase.LogToTimelineMapper.
func (n *networkAPILogToTimelineMapperTaskSetting) ProcessLogByGroup(ctx context.Context, l *log.Log, cs *history.ChangeSet, builder *history.Builder, prevGroupData *perNEGHistoryModificationStatus) (*perNEGHistoryModificationStatus, error) {
	commonFieldSet := log.MustGetFieldSet(l, &log.CommonFieldSet{})
	auditFieldSet := log.MustGetFieldSet(l, &googlecloudcommon_contract.GCPAuditLogFieldSet{})
	if prevGroupData == nil {
		prevGroupData = &perNEGHistoryModificationStatus{}
	}

	negs := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.NEGNamesInventoryTaskID.Ref())
	var negResourcePath resourcepath.ResourcePath
	negName := getNegNameFromResourceName(auditFieldSet.ResourceName)

	if negResource, found := negs[negName]; found {
		negResourcePath = resourcepath.NetworkEndpointGroup(negResource.Namespace, negName)
	} else {
		negResourcePath = resourcepath.NetworkEndpointGroup("unknown", negName)
	}

	// Add operation subresource under sneg resource
	negOperationPath := auditFieldSet.OperationPath(negResourcePath)
	if auditFieldSet.ImmediateOperation() {
		cs.AddEvent(negOperationPath)
	} else {
		state := enum.RevisionStateOperationStarted
		verb := enum.RevisionVerbOperationStart
		if auditFieldSet.Ending() {
			state = enum.RevisionStateOperationFinished
			verb = enum.RevisionVerbOperationFinish
		}
		requestBody, _ := auditFieldSet.RequestString()
		cs.AddRevision(negOperationPath, &history.StagingResourceRevision{
			Body:       requestBody,
			Verb:       verb,
			State:      state,
			Requestor:  auditFieldSet.PrincipalEmail,
			ChangeTime: commonFieldSet.Timestamp,
		})
	}

	ipLeases := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.IPLeaseHistoryInventoryTaskID.Ref())
	// Add neg subresource under resources with the same IP of the endpoint
	shortMethodName := getShortMethodNameFromMethodName(auditFieldSet.MethodName)
	var negRequest *negAttachOrDetachRequest
	var verb enum.RevisionVerb
	var state enum.RevisionState
	switch shortMethodName {
	case "attachNetworkEndpoints":
		if auditFieldSet.Starting() {
			// Operation starting log only contain its request(IP data), but it should be marked as ready when the last log coming.
			var err error
			request, err := parseNEGAttachOrDetachRequest(l)
			if err != nil {
				return prevGroupData, err
			}
			prevGroupData.LastNegAttachRequest = request // Save the neg attach request in the per group status, and it will be consumed in the next ending operation log.
			break
		}
		negRequest = prevGroupData.LastNegAttachRequest
		prevGroupData.LastNegAttachRequest = nil
		verb = enum.RevisionVerbReady
		state = enum.RevisionStateConditionTrue
	case "detachNetworkEndpoints":
		if auditFieldSet.Ending() {
			break
		}
		var err error
		negRequest, err = parseNEGAttachOrDetachRequest(l)
		if err != nil {
			return prevGroupData, err
		}
		verb = enum.RevisionVerbNonReady
		state = enum.RevisionStateConditionFalse
	}
	if negRequest != nil {
		for _, endpoint := range negRequest.NetworkEndpoints {
			lease, err := ipLeases.GetResourceLeaseHolderAt(endpoint.IpAddress, commonFieldSet.Timestamp)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Failed to identify the holder of the IP %s.\n This might be because the IP holder resource wasn't updated during the log period ", endpoint.IpAddress))
				continue
			}
			holder := lease.Holder
			if holder.Kind == "pod" {
				podPath := resourcepath.Pod(holder.Namespace, holder.Name)
				negSubresourcePath := resourcepath.NetworkEndpointGroupUnderResource(podPath, holder.Namespace, negName)
				cs.AddRevision(negSubresourcePath, &history.StagingResourceRevision{
					Verb:       verb,
					State:      state,
					Requestor:  auditFieldSet.PrincipalEmail,
					ChangeTime: commonFieldSet.Timestamp,
				})
			}
		}

	}

	switch {
	case auditFieldSet.Starting():
		cs.SetLogSummary(fmt.Sprintf("%s Started", auditFieldSet.MethodName))
	case auditFieldSet.Ending():
		cs.SetLogSummary(fmt.Sprintf("%s Finished", auditFieldSet.MethodName))
	default:
		cs.SetLogSummary(auditFieldSet.MethodName)
	}
	return prevGroupData, nil
}

var _ inspectiontaskbase.LogToTimelineMapper[*perNEGHistoryModificationStatus] = (*networkAPILogToTimelineMapperTaskSetting)(nil)

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
