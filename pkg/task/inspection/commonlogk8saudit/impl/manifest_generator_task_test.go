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

package commonlogk8saudit_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/google/go-cmp/cmp"
)

type testGroupManifestGeneratorInput struct {
	verb         *pb.Verb
	requestYAML  string
	responseYAML string
}

func TestGroupManifestGenerator(t *testing.T) {
	testCases := []struct {
		desc         string
		inputs       []*testGroupManifestGeneratorInput
		resourceName string
		wantBodies   []string
	}{
		{
			desc: "update must override existing values",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbUpdate,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar`,
				},
				{
					verb: commonlogk8saudit_contract.VerbUpdate,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  labels:
    qux: quux`,
				},
			},
			wantBodies: []string{
				`apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar
`,
				`apiVersion: v1
kind: Pod
metadata:
  labels:
    qux: quux
`,
			},
		},
		{
			desc: "simple patch request",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbUpdate,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar`,
				},
				{
					verb: commonlogk8saudit_contract.VerbPatch,
					requestYAML: `metadata:
  labels:
    qux: quux`,
				},
			},
			wantBodies: []string{
				`apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar
`,
				`apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar
    qux: quux
`,
			},
		},
		{
			desc: "delete responded with deleteOptions must retain the previous merged result",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbUpdate,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar`,
				},
				{
					verb: commonlogk8saudit_contract.VerbDelete,
					responseYAML: `apiVersion: meta.k8s.io/__internal
kind: DeleteOptions
`,
				},
			},
			wantBodies: []string{`apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar
`,
				`apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar
`},
		},
		{
			desc: "response with Status must use request",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbUpdate,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar`,
				},
				{
					verb: commonlogk8saudit_contract.VerbPatch,
					responseYAML: `apiVersion: v1
kind: Status`,
					requestYAML: `metadata:
  labels:
    qux: quux`},
			},
			wantBodies: []string{
				`apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar
`,
				`apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar
    qux: quux
`,
			},
		},
		{
			desc:         "deletecollection for set of pods",
			resourceName: "test-pod",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbDelete,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    foo: bar`,
				},
				{
					verb: commonlogk8saudit_contract.VerbDeleteCollection,
					responseYAML: `apiVersion: v1
kind: PodList
items:
    - metadata:
        name: not-a-test-pod
        labels:
            foo: qux
    - metadata:
        name: test-pod
        labels:
            foo: qux`,
				},
			},
			wantBodies: []string{
				`apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    foo: bar
`,
				`apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    foo: qux
`},
		},
		{
			desc:         "deletecollection at the beginnning of logs bound to the resource",
			resourceName: "test-pod",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbDeleteCollection,
					responseYAML: `apiVersion: v1
kind: PodList
items:
    - metadata:
        name: not-a-test-pod
        labels:
            foo: qux
    - metadata:
        name: test-pod
        labels:
            foo: qux`,
				},
			},
			wantBodies: []string{ // XXXList doesn't include apiVersion or kind in its items, in the case, KHI can't create populate the apiVersion and kind fields.
				`metadata:
  name: test-pod
  labels:
    foo: qux
`,
			},
		},
		{
			desc:         "deletecollection for entire namespace",
			resourceName: "test-pod",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbDelete,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    foo: bar`,
				},
				{
					verb: commonlogk8saudit_contract.VerbDeleteCollection,
					responseYAML: `apiVersion: meta.k8s.io/__internal
kind: DeleteOptions`,
				},
			},
			wantBodies: []string{
				`apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    foo: bar
`,
				`apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    foo: bar
`},
		},
		{
			desc: "metadata level requests",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbUpdate,
				},
				{
					verb: commonlogk8saudit_contract.VerbUpdate,
				},
			},
			wantBodies: []string{
				"# Resource data is unavailable. Audit logs for this resource is recorded at metadata level.",
				"# Resource data is unavailable. Audit logs for this resource is recorded at metadata level.",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			logs := []*log.Log{}
			for _, input := range tc.inputs {
				var request, response *structured.NodeReader
				if input.requestYAML != "" {
					node, err := structured.FromYAML(input.requestYAML)
					if err != nil {
						t.Fatalf("failed to parse request YAML: %v", err)
					}
					request = structured.NewNodeReader(node)
				}
				if input.responseYAML != "" {
					node, err := structured.FromYAML(input.responseYAML)
					if err != nil {
						t.Fatalf("failed to parse response YAML: %v", err)
					}
					response = structured.NewNodeReader(node)
				}
				verb := input.verb
				logs = append(logs, log.NewLogWithFieldSetsForTest(&commonlogk8saudit_contract.K8sAuditLogFieldSet{
					ClusterName: "k8s",
					Verb:        verb,
					Request:     request,
					Response:    response,
				}))
			}

			config, err := k8s.GenerateDefaultMergeConfig()
			if err != nil {
				t.Fatalf("failed to generate default merge config:%v", config)
			}
			groupManifestGenerator := groupManifestGenerator{
				mergeConfigRegistry: config,
				resourceName:        tc.resourceName,
			}
			gotManifests := []string{}
			for _, l := range logs {
				rl, err := groupManifestGenerator.Process(t.Context(), l)
				if err != nil {
					t.Errorf("failed to generate manifest:%v", err)
				}
				gotManifests = append(gotManifests, rl.ResourceBodyYAML)

				if rl.ResourceBodyReader == nil {
					continue
				}

				yamlFromReader, err := rl.ResourceBodyReader.Serialize("", &structured.YAMLNodeSerializer{})
				if err != nil {
					t.Errorf("failed to serialize resource body to yaml\n%s", err.Error())
				}
				if diff := cmp.Diff(rl.ResourceBodyYAML, string(yamlFromReader)); diff != "" {
					t.Errorf("YAML mismatch between reader and string (-want +got):%s", diff)
				}
			}
			if diff := cmp.Diff(tc.wantBodies, gotManifests); diff != "" {
				t.Errorf("mismatch (-want +got):%s", diff)
			}
		})
	}
}
