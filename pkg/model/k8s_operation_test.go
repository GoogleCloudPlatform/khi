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

package model

import (
	"fmt"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestToSingularKindName(t *testing.T) {
	testCases := []struct {
		plural   string
		singular string
	}{
		{
			plural:   "pods",
			singular: "pod",
		},
		{
			plural:   "services",
			singular: "service",
		},
		{
			plural:   "ingresses",
			singular: "ingress",
		},
		{
			plural:   "clusterdnses",
			singular: "clusterdns",
		},
		{
			plural:   "csinodetopologies",
			singular: "csinodetopology",
		},
		{
			plural:   "entitlementidentities",
			singular: "entitlementidentity",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("plural:%s", tc.plural), func(t *testing.T) {
			o := KubernetesObjectOperation{PluralKind: tc.plural}

			if tc.singular != o.GetSingularKindName() {
				t.Errorf("got %q, want %q", o.GetSingularKindName(), tc.singular)
			}
		})
	}
}
