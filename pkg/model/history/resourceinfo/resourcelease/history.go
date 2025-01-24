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

package resourcelease

import (
	"errors"
	"log/slog"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
)

// Resource identifier itself couldn't be found
var NoResourceFound = errors.New("no resource identifier found")

// Resource identifier was found, but no holder found at the given time
var NoResourceLeaseHolderFoundAtTheTime = errors.New("no resource holder at the time")

type LeaseHolder interface {
	Equals(holder LeaseHolder) bool
}

type lease[H LeaseHolder] struct {
	StartAt time.Time
	Holder  H
}

// ResourceLeaseHistory is a common interface for memorizing resources used in the cluster.
// This is used for example: IPs associated to a Pod, Load balancer names associated to Ingress ..etc.
type ResourceLeaseHistory[H LeaseHolder] struct {
	leaseHolders *common.ShardingMap[[]*lease[H]]
}

func NewResourceLeaseHistory[H LeaseHolder]() *ResourceLeaseHistory[H] {
	return &ResourceLeaseHistory[H]{
		leaseHolders: common.NewShardingMap[[]*lease[H]](common.NewSuffixShardingProvider(128, 4)),
	}
}

func (r *ResourceLeaseHistory[H]) TouchResourceLease(resourceIdentifier string, holdsAtLeastAt time.Time, holder H) {
	leaseHolderMap := r.leaseHolders.AcquireShard(resourceIdentifier)
	defer r.leaseHolders.ReleaseShard(resourceIdentifier)
	lastLease, index, err := r.getResourceLeaseHolderAtWithIndex(leaseHolderMap, resourceIdentifier, holdsAtLeastAt)
	newLease := &lease[H]{
		Holder:  holder,
		StartAt: holdsAtLeastAt,
	}
	if errors.Is(NoResourceFound, err) {
		leaseHolderMap[resourceIdentifier] = []*lease[H]{
			newLease,
		}
		return
	}
	if errors.Is(NoResourceLeaseHolderFoundAtTheTime, err) {
		prevLeaseSeries := leaseHolderMap[resourceIdentifier]
		next := prevLeaseSeries[0]
		newLease := lease[H]{
			Holder:  holder,
			StartAt: holdsAtLeastAt,
		}
		current := []*lease[H]{&newLease}

		if holder.Equals(next.Holder) {
			current = append(current, prevLeaseSeries[1:]...)
		} else {
			current = append(current, prevLeaseSeries...)
		}
		leaseHolderMap[resourceIdentifier] = current
		return
	}
	if lastLease == nil {
		slog.Info("last lease was nil!")
	}
	if holder.Equals(lastLease.Holder) {
		// No change at the holder,ignore it.
		return
	}
	previous := leaseHolderMap[resourceIdentifier]
	if len(previous)-1 == index {
		leaseHolderMap[resourceIdentifier] = append(leaseHolderMap[resourceIdentifier], newLease)
		return
	}
	current := []*lease[H]{}
	current = append(current, previous[:index+1]...)
	current = append(current, newLease)
	current = append(current, previous[index+1:]...)
	leaseHolderMap[resourceIdentifier] = current
}

func (r *ResourceLeaseHistory[H]) GetResourceLeaseHolderAt(resourceIdentifier string, time time.Time) (*lease[H], error) {
	leaseHolderMap := r.leaseHolders.AcquireShard(resourceIdentifier)
	defer r.leaseHolders.ReleaseShard(resourceIdentifier)
	lease, _, err := r.getResourceLeaseHolderAtWithIndex(leaseHolderMap, resourceIdentifier, time)
	return lease, err
}

func (r *ResourceLeaseHistory[H]) getResourceLeaseHolderAtWithIndex(leaseHolderMap map[string][]*lease[H], resourceIdentifier string, time time.Time) (*lease[H], int, error) {
	if leases, found := leaseHolderMap[resourceIdentifier]; !found {
		return nil, 0, NoResourceFound
	} else {
		if time.Sub(leases[len(leases)-1].StartAt) >= 0 {
			return leases[len(leases)-1], len(leases) - 1, nil
		}
		if time.Sub(leases[0].StartAt) < 0 {
			return nil, 0, NoResourceLeaseHolderFoundAtTheTime
		}
		var min = 0
		var max = len(leases) - 1
		for {
			if max-min == 1 {
				break
			}
			mid := (max + min) / 2
			if time.Sub(leases[mid].StartAt) >= 0 {
				min = mid
			} else {
				max = mid
			}
		}
		return leases[min], min, nil
	}
}

func (r *ResourceLeaseHistory[H]) GetAllIdentifiers() []string {
	return r.leaseHolders.AllKeys()
}
