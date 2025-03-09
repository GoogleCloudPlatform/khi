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

package resourceinfo

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
)

const (
	DiffNew     EndppointSliceEndpointDiffOperation = "NEW"
	DiffChanged EndppointSliceEndpointDiffOperation = "CHANGED"
	DiffDeleted EndppointSliceEndpointDiffOperation = "DELETED"
)

type EndppointSliceEndpointDiffOperation = string

type EndpointSliceEndpointDiff struct {
	Operation EndppointSliceEndpointDiffOperation
	Current   *model.EndpointSliceEndpoint
	Previous  *model.EndpointSliceEndpoint
	TargetRef *model.K8sTargetRef
}

type EndpointSliceInfo struct {
	endpointSlices map[string][]*model.EndpointSlice
	ipLeases       *resourcelease.ResourceLeaseHistory[*resourcelease.K8sResourceLeaseHolder]
}

func newEndpointSliceInfo(ipLeases *resourcelease.ResourceLeaseHistory[*resourcelease.K8sResourceLeaseHolder]) *EndpointSliceInfo {
	return &EndpointSliceInfo{endpointSlices: make(map[string][]*model.EndpointSlice), ipLeases: ipLeases}
}

// ReadEndpointSlice parse given EndpointSlice manifest and returns the uid of endpoint slice
func (e *EndpointSliceInfo) ReadEndpointSlice(reader *structure.Reader, time time.Time) (string, error) {
	var endpointSlice model.EndpointSlice
	err := reader.ReadReflect("", &endpointSlice)
	if err != nil {
		return "", err
	}
	for _, endpoint := range endpointSlice.Endpoints {
		if endpoint.TargetRef != nil {
			target := endpoint.TargetRef
			for _, address := range endpoint.Addresses {
				e.ipLeases.TouchResourceLease(address, time, resourcelease.NewK8sResourceLeaseHolder(target.Kind, target.Namespace, target.Name))
			}
		}
	}
	if endpointSlice.Metadata == nil {
		var deletePrecondition model.K8sDeleteRequest
		err := reader.ReadReflect("", &deletePrecondition)
		if err != nil {
			return "", fmt.Errorf("failed to get metadata of endpoint slice")
		}
		uid := deletePrecondition.Preconditions.Uid
		resources := e.endpointSlices[uid]
		e.endpointSlices[uid] = append(resources, nil)
		return uid, nil
	}
	if _, found := e.endpointSlices[endpointSlice.Metadata.UID]; !found {
		e.endpointSlices[endpointSlice.Metadata.UID] = make([]*model.EndpointSlice, 0)
	}
	resources := e.endpointSlices[endpointSlice.Metadata.UID]
	e.endpointSlices[endpointSlice.Metadata.UID] = append(resources, &endpointSlice)
	return endpointSlice.Metadata.UID, nil
}

func (e *EndpointSliceInfo) getLastPair(uid string) (*model.EndpointSlice, *model.EndpointSlice, error) {
	if resources, found := e.endpointSlices[uid]; found {
		if len(resources) == 1 {
			return nil, resources[0], nil
		} else {
			return resources[len(resources)-2], resources[len(resources)-1], nil
		}
	}
	return nil, nil, fmt.Errorf("specified endpoint slice couldn't be found")
}

func (e *EndpointSliceInfo) GetLastDiffs(uid string) ([]*EndpointSliceEndpointDiff, error) {
	previous, current, err := e.getLastPair(uid)
	if err != nil {
		return nil, err
	}
	result := []*EndpointSliceEndpointDiff{}

	switch {
	case previous == nil:
		if len(current.Endpoints) == 0 {
			return []*EndpointSliceEndpointDiff{}, nil
		}
		for _, endpoint := range current.Endpoints {
			result = append(result, &EndpointSliceEndpointDiff{
				Operation: DiffDeleted,
				Previous:  nil,
				Current:   endpoint,
				TargetRef: endpoint.TargetRef,
			})
		}
	case current == nil:
		for _, endpoint := range previous.Endpoints {
			result = append(result, &EndpointSliceEndpointDiff{
				Operation: DiffDeleted,
				Previous:  endpoint,
				Current:   nil,
				TargetRef: endpoint.TargetRef,
			})
		}
	default:
		previousEndpointsMap := map[string]*model.EndpointSliceEndpoint{}
		currentEndpointsMap := map[string]*model.EndpointSliceEndpoint{}
		if previous.Endpoints != nil {
			for _, endpoint := range previous.Endpoints {
				if endpoint == nil || endpoint.TargetRef == nil {
					slog.Warn(fmt.Sprintf("endpoint (%s) has no uid in the target ref", strings.Join(endpoint.Addresses, ",")))
					continue
				}
				previousEndpointsMap[endpoint.TargetRef.Uid] = endpoint
			}
		}
		if current.Endpoints != nil {
			for _, endpoint := range current.Endpoints {
				if endpoint == nil || endpoint.TargetRef == nil {
					slog.Warn(fmt.Sprintf("endpoint (%s) has no uid in the target ref", strings.Join(endpoint.Addresses, ",")))
					continue
				}
				currentEndpointsMap[endpoint.TargetRef.Uid] = endpoint
			}
		}
		for key, previous := range previousEndpointsMap {
			if _, found := currentEndpointsMap[key]; !found {
				result = append(result, &EndpointSliceEndpointDiff{
					Operation: DiffDeleted,
					Previous:  previous,
					Current:   nil,
					TargetRef: previous.TargetRef,
				})
			}
		}
		for key, current := range currentEndpointsMap {
			if previous, found := previousEndpointsMap[key]; !found {
				result = append(result, &EndpointSliceEndpointDiff{
					Operation: DiffNew,
					Previous:  nil,
					Current:   current,
					TargetRef: current.TargetRef,
				})
			} else if !previous.Conditions.SameWith(current.Conditions) {
				result = append(result, &EndpointSliceEndpointDiff{
					Operation: DiffChanged,
					Previous:  previous,
					Current:   current,
					TargetRef: current.TargetRef,
				})
			}
		}
	}
	return result, nil
}
