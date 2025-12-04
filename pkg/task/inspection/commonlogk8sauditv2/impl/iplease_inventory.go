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
	"log/slog"
	"strings"

	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var IPLeaseHistoryInventoryTask = commonlogk8sauditv2_contract.IPLeaseHistoryInventoryBuilder.InventoryTask(&ipLeaseHistoryInventoryMergeStrategy{})

type ipLeaseHistoryInventoryMergeStrategy struct{}

// Merge implements inspectiontaskbase.InventoryMergerStrategy.
func (i *ipLeaseHistoryInventoryMergeStrategy) Merge(results []commonlogk8sauditv2_contract.IPLeaseHistory) (commonlogk8sauditv2_contract.IPLeaseHistory, error) {
	return resourcelease.MergeResourceLeaseHistories(results...), nil
}

var _ inspectiontaskbase.InventoryMergerStrategy[commonlogk8sauditv2_contract.IPLeaseHistory] = (*ipLeaseHistoryInventoryMergeStrategy)(nil)

var IPLeaseHistoryDiscoveryTask = commonlogk8sauditv2_contract.IPLeaseHistoryInventoryBuilder.DiscoveryTask(
	commonlogk8sauditv2_contract.IPLeaseHistoryDiscoveryTaskID,
	[]taskid.UntypedTaskReference{commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref()},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) (commonlogk8sauditv2_contract.IPLeaseHistory, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return nil, nil
		}
		resourceLogs := coretask.GetTaskResult(ctx, commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref())
		leaseHistory := resourcelease.NewResourceLeaseHistory[*commonlogk8sauditv2_contract.ResourceIdentity]()
		for _, group := range resourceLogs {
			if group.Resource.Type() != commonlogk8sauditv2_contract.Resource {
				continue
			}
			switch {
			case group.Resource.APIVersion == "core/v1" && group.Resource.Kind == "pod":
				processPodResource(group, leaseHistory)
			case group.Resource.APIVersion == "discovery.k8s.io/v1" && group.Resource.Kind == "endpointslice":
				processEndpointSliceResource(ctx, group, leaseHistory)
			}
		}
		return leaseHistory, nil
	},
)

func processPodResource(group *commonlogk8sauditv2_contract.ResourceManifestLogGroup, leaseHistory commonlogk8sauditv2_contract.IPLeaseHistory) {
	for _, l := range group.Logs {
		if l.ResourceBodyReader == nil {
			continue
		}
		commonFieldSet := log.MustGetFieldSet(l.Log, &log.CommonFieldSet{})
		if l.ResourceBodyReader.ReadStringOrDefault("status.phase", "") != "Running" {
			continue
		}
		ips := map[string]struct{}{}
		podIPsReader, err := l.ResourceBodyReader.GetReader("status.podIPs")
		if err == nil {
			for _, podIPReader := range podIPsReader.Children() {
				ip, err := podIPReader.ReadString("ip")
				if err != nil {
					continue
				}
				ips[ip] = struct{}{}
			}
		}
		podMainIP, err := l.ResourceBodyReader.ReadString("status.podIP")
		if err == nil {
			ips[podMainIP] = struct{}{}
		}
		for ip := range ips {
			leaseHistory.TouchResourceLease(ip, commonFieldSet.Timestamp, group.Resource)
		}
	}
}

func processEndpointSliceResource(ctx context.Context, group *commonlogk8sauditv2_contract.ResourceManifestLogGroup, leaseHistory commonlogk8sauditv2_contract.IPLeaseHistory) {
	for _, l := range group.Logs {
		if l.ResourceBodyReader == nil {
			continue
		}
		commonFieldSet := log.MustGetFieldSet(l.Log, &log.CommonFieldSet{})
		endpointsReader, err := l.ResourceBodyReader.GetReader("endpoints")
		if err != nil {
			continue
		}
		for _, endpointReader := range endpointsReader.Children() {
			ips := []string{}
			addressesReader, err := endpointReader.GetReader("addresses")
			if err != nil {
				continue
			}
			for _, addressValue := range addressesReader.Children() {
				ip, err := addressValue.ReadString("")
				if err != nil {
					continue
				}
				ips = append(ips, ip)
			}

			// The apiVersion field in targetRef seems to be missing in endpoint manifest for most of the cases.
			// Current implementation infers the apiVersion from its kind.

			targetRefReader, err := endpointReader.GetReader("targetRef")
			if err != nil {
				continue
			}
			kind, err := targetRefReader.ReadString("kind")
			if err != nil {
				continue
			}
			kind = strings.ToLower(kind)
			name, err := targetRefReader.ReadString("name")
			if err != nil {
				continue
			}
			namespace, err := targetRefReader.ReadString("namespace")
			if err != nil {
				continue
			}
			for _, ip := range ips {
				switch kind {
				case "pod":
					leaseHistory.TouchResourceLease(ip, commonFieldSet.Timestamp, &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       name,
						Namespace:  namespace,
					})
				default:
					slog.WarnContext(ctx, "unsupported kind for IP lease history discovery task", "kind", kind)
				}
			}
		}
	}
}
