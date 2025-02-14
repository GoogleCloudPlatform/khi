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
	"testing"
	"time"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type testLeaseHolder struct {
	id string
}

// Equals implements LeaseHolder.
func (t *testLeaseHolder) Equals(holder LeaseHolder) bool {
	return holder.(*testLeaseHolder).id == t.id
}

var _ LeaseHolder = (*testLeaseHolder)(nil)

func TestTouchResourceLease(t *testing.T) {
	type touch struct {
		identifier  string
		touchTiming time.Time
		holderId    string
	}
	type testCase struct {
		name               string
		touches            []touch
		resourceIdentifier string
		time               time.Time
		expectedId         string
		expectedLeaseTime  time.Time
		expectedErr        error
	}
	testCases := []testCase{
		{
			name: "after last lease in existing resource",
			touches: []touch{
				{
					"resource-foo", time.Date(2000, time.April, 1, 0, 0, 0, 0, time.UTC), "holder-bar1",
				},
				{
					"resource-foo", time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC), "holder-bar3",
				},
			},
			resourceIdentifier: "resource-foo",
			time:               time.Date(2000, time.April, 4, 0, 0, 0, 0, time.UTC),
			expectedLeaseTime:  time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC),
			expectedId:         "holder-bar3",
		},
		{
			name: "before the first lease in existing resource",
			touches: []touch{
				{
					"resource-foo", time.Date(2000, time.April, 1, 0, 0, 0, 0, time.UTC), "holder-bar1",
				},
				{
					"resource-foo", time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC), "holder-bar3",
				},
			},
			resourceIdentifier: "resource-foo",
			time:               time.Date(1999, time.April, 4, 0, 0, 0, 0, time.UTC),
			expectedLeaseTime:  time.Time{},
			expectedErr:        NoResourceLeaseHolderFoundAtTheTime,
		},
		{
			name: "middle leases in existing resource",
			touches: []touch{
				{
					"resource-foo", time.Date(2000, time.April, 1, 0, 0, 0, 0, time.UTC), "holder-bar1",
				},
				{
					"resource-foo", time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 5, 0, 0, 0, 0, time.UTC), "holder-bar3",
				},
			},
			resourceIdentifier: "resource-foo",
			time:               time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC),
			expectedLeaseTime:  time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC),
			expectedId:         "holder-bar2",
		},
		{
			name: "non existing resource",
			touches: []touch{
				{
					"resource-foo", time.Date(2000, time.April, 1, 0, 0, 0, 0, time.UTC), "holder-bar1",
				},
				{
					"resource-foo", time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 5, 0, 0, 0, 0, time.UTC), "holder-bar3",
				},
			},
			resourceIdentifier: "resource-qux",
			time:               time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC),
			expectedErr:        NoResourceFound,
		},
		{
			name: "random order update",
			touches: []touch{
				{
					"resource-foo", time.Date(2000, time.April, 1, 0, 0, 0, 0, time.UTC), "holder-bar1",
				},
				{
					"resource-foo", time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC), "holder-bar3",
				},
				{
					"resource-foo", time.Date(2000, time.April, 6, 0, 0, 0, 0, time.UTC), "holder-bar4",
				},
				{
					"resource-foo", time.Date(2000, time.April, 5, 0, 0, 0, 0, time.UTC), "holder-bar5",
				},
			},
			resourceIdentifier: "resource-foo",
			time:               time.Date(2000, time.April, 4, 0, 0, 0, 0, time.UTC),
			expectedId:         "holder-bar2",
			expectedLeaseTime:  time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "including same lease holder",
			touches: []touch{
				{
					"resource-foo", time.Date(2000, time.April, 1, 0, 0, 0, 0, time.UTC), "holder-bar1",
				},
				{
					"resource-foo", time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 3, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 4, 0, 0, 0, 0, time.UTC), "holder-bar2",
				},
				{
					"resource-foo", time.Date(2000, time.April, 6, 0, 0, 0, 0, time.UTC), "holder-bar1",
				},
			},
			resourceIdentifier: "resource-foo",
			time:               time.Date(2000, time.April, 5, 0, 0, 0, 0, time.UTC),
			expectedId:         "holder-bar2",
			expectedLeaseTime:  time.Date(2000, time.April, 2, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			history := NewResourceLeaseHistory[*testLeaseHolder]()
			for _, touch := range tc.touches {
				history.TouchResourceLease(touch.identifier, touch.touchTiming, &testLeaseHolder{
					id: touch.holderId,
				})
			}
			lease, err := history.GetResourceLeaseHolderAt(tc.resourceIdentifier, tc.time)
			if tc.expectedErr != nil {
				if !errors.Is(tc.expectedErr, err) {
					t.Errorf("unmatched error %s(expected:%s)", err.Error(), tc.expectedErr.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error %s", err)
				}
				if tc.expectedId != lease.Holder.id {
					t.Errorf("unmatched holder id %s(expected:%s)", lease.Holder.id, tc.expectedId)
				}
				if tc.expectedLeaseTime != lease.StartAt {
					t.Errorf("unmatched lease time %s(expected:%s)", lease.StartAt.String(), tc.expectedLeaseTime.String())
				}
			}
		})
	}
}
