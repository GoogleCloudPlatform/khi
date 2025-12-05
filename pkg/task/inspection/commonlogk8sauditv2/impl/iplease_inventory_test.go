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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestIPLeaseHistoryDiscoveryTassk(t *testing.T) {
	testTime := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		desc            string
		inputResource   *commonlogk8sauditv2_contract.ResourceIdentity
		inputManifest   string
		wantIdentifiers []struct {
			ip       string
			resource *commonlogk8sauditv2_contract.ResourceIdentity
		}
	}{
		{
			desc: "standard pod input",
			inputResource: &commonlogk8sauditv2_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Name:       "test-pod",
				Namespace:  "test-namespace",
			},
			inputManifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: test-namespace
status:
  phase: Running
  podIP: 10.0.0.1`,
			wantIdentifiers: []struct {
				ip       string
				resource *commonlogk8sauditv2_contract.ResourceIdentity
			}{
				{
					ip: "10.0.0.1",
					resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       "test-pod",
						Namespace:  "test-namespace",
					},
				},
			},
		},
		{
			desc: "a pod with multiple podIPs",
			inputResource: &commonlogk8sauditv2_contract.ResourceIdentity{
				APIVersion: "core/v1",
				Kind:       "pod",
				Name:       "test-pod",
				Namespace:  "test-namespace",
			},
			inputManifest: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: test-namespace
status:
  phase: Running
  podIP: 10.0.0.1
  podIPs:
    - ip: 10.0.0.2
    - ip: 10.0.0.3
`,
			wantIdentifiers: []struct {
				ip       string
				resource *commonlogk8sauditv2_contract.ResourceIdentity
			}{
				{
					ip: "10.0.0.1",
					resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       "test-pod",
						Namespace:  "test-namespace",
					},
				},
				{
					ip: "10.0.0.2",
					resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       "test-pod",
						Namespace:  "test-namespace",
					},
				},
				{
					ip: "10.0.0.3",
					resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       "test-pod",
						Namespace:  "test-namespace",
					},
				},
			},
		},
		{
			desc: "a standard endpoint slice",
			inputResource: &commonlogk8sauditv2_contract.ResourceIdentity{
				APIVersion: "discovery.k8s.io/v1",
				Kind:       "endpointslice",
				Name:       "test-endpointslice",
				Namespace:  "test-namespace",
			},
			inputManifest: `apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: test-endpointslice
  namespace: test-namespace
endpoints:
  - addresses:
    - 10.0.0.1
    - 10.0.0.2
    targetRef:
      kind: Pod
      name: test-pod
      namespace: test-namespace
  - addresses:
    - 10.0.0.3
    targetRef:
      kind: Pod
      name: test-pod2
      namespace: test-namespace`,
			wantIdentifiers: []struct {
				ip       string
				resource *commonlogk8sauditv2_contract.ResourceIdentity
			}{
				{
					ip: "10.0.0.1",
					resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       "test-pod",
						Namespace:  "test-namespace",
					},
				},
				{
					ip: "10.0.0.2",
					resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       "test-pod",
						Namespace:  "test-namespace",
					},
				},
				{
					ip: "10.0.0.3",
					resource: &commonlogk8sauditv2_contract.ResourceIdentity{
						APIVersion: "core/v1",
						Kind:       "pod",
						Name:       "test-pod2",
						Namespace:  "test-namespace",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(&log.CommonFieldSet{
				Timestamp: testTime,
			})
			yamlNode, err := structured.FromYAML(test.inputManifest)
			if err != nil {
				t.Fatal(err)
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			input := commonlogk8sauditv2_contract.ResourceManifestLogGroupMap{}
			input["test"] = &commonlogk8sauditv2_contract.ResourceManifestLogGroup{
				Resource: test.inputResource,
				Logs: []*commonlogk8sauditv2_contract.ResourceManifestLog{
					{
						Log:                l,
						ResourceBodyReader: structured.NewNodeReader(yamlNode),
					},
				},
			}
			got, _, err := inspectiontest.RunInspectionTask(ctx, IPLeaseHistoryDiscoveryTask, inspectioncore_contract.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(commonlogk8sauditv2_contract.ManifestGeneratorTaskID.Ref(), input),
			)
			if err != nil {
				t.Fatalf("RunInspectionTask failed: %v", err)
			}
			for i, wantIdentifier := range test.wantIdentifiers {
				result, err := got.GetResourceLeaseHolderAt(wantIdentifier.ip, testTime)
				if err != nil {
					t.Fatalf("GetResourceLeaseHolderAt failed: %v", err)
				}
				if diff := cmp.Diff(wantIdentifier.resource, result.Holder); diff != "" {
					t.Errorf("GetResourceLeaseHolderAt returned unexpected diff (-want +got) at %d th identifier:\n%s", i, diff)
				}
			}
		})
	}
}
