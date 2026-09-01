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
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

type testGroupManifestGeneratorInput struct {
	verb         *pb.Verb
	requestYAML  string
	responseYAML string
	isDryRun     bool
	isTruncated  bool
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
			desc: "truncated log clears previous revision and does not merge old state in subsequent patch",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
					responseYAML: `apiVersion: v1
kind: Pod
metadata:
  labels:
    foo: bar`,
				},
				{
					verb:        commonlogk8saudit_contract.VerbUpdate,
					isTruncated: true,
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
				"",
				`metadata:
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
				"",
				"",
			},
		},
		{
			desc: "dry run log should not override existing values",
			inputs: []*testGroupManifestGeneratorInput{
				{
					verb: commonlogk8saudit_contract.VerbCreate,
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
					isDryRun: true,
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
`,
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
				logs = append(logs, testlog.NewMockLog(commonlogk8saudit_contract.K8sAuditLogFieldSet{
					ClusterName: "k8s",
					Verb:        verb,
					Request:     request,
					Response:    response,
					IsDryRun:    input.isDryRun,
					IsTruncated: input.isTruncated,
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
				if rl.ResourceBodyReader == nil {
					gotManifests = append(gotManifests, "")
					continue
				}

				yamlFromReader, err := rl.ResourceBodyReader.Serialize(structured.EmptyFieldPath, &structured.YAMLNodeSerializer{})
				if err != nil {
					t.Errorf("failed to serialize resource body to yaml\n%s", err.Error())
				}
				gotManifests = append(gotManifests, string(yamlFromReader))
			}
			if diff := cmp.Diff(tc.wantBodies, gotManifests); diff != "" {
				t.Errorf("mismatch (-want +got):%s", diff)
			}
		})
	}
}

func TestConstructResourceBodyFromListItem(t *testing.T) {
	testCases := []struct {
		name         string
		itemYAML     string
		prevYAML     string
		wantYAML     string
		wantAPIVer   string
		wantKind     string
		wantMetaName string
		wantErr      bool
	}{
		{
			name: "injects apiVersion and kind from prevRevision",
			itemYAML: `metadata:
  name: test-pod
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:latest`,
			prevYAML: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod`,
			wantYAML: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
spec:
  containers:
    - name: nginx
      image: nginx:latest
`,
			wantAPIVer:   "v1",
			wantKind:     "Pod",
			wantMetaName: "test-pod",
		},
		{
			name: "prevRevision is empty without apiVersion or kind",
			itemYAML: `metadata:
  name: test-cm
data:
  key: value`,
			prevYAML: ``,
			wantYAML: `metadata:
  name: test-cm
data:
  key: value
`,
			wantAPIVer:   "",
			wantKind:     "",
			wantMetaName: "test-cm",
		},
		{
			name: "prevRevision has only apiVersion",
			itemYAML: `metadata:
  name: test-res`,
			prevYAML: `apiVersion: apps/v1`,
			wantYAML: `apiVersion: apps/v1
metadata:
  name: test-res
`,
			wantAPIVer:   "apps/v1",
			wantKind:     "",
			wantMetaName: "test-res",
		},
		{
			name: "prevRevision has only kind",
			itemYAML: `metadata:
  name: test-res`,
			prevYAML: `kind: Deployment`,
			wantYAML: `kind: Deployment
metadata:
  name: test-res
`,
			wantAPIVer:   "",
			wantKind:     "Deployment",
			wantMetaName: "test-res",
		},
		{
			name: "nested array and mapping fields preserved in key order",
			itemYAML: `status:
  phase: Running
metadata:
  name: complex-pod
spec:
  nodeName: node-1`,
			prevYAML: `apiVersion: v1
kind: Pod`,
			wantYAML: `apiVersion: v1
kind: Pod
metadata:
  name: complex-pod
spec:
  nodeName: node-1
status:
  phase: Running
`,
			wantAPIVer:   "v1",
			wantKind:     "Pod",
			wantMetaName: "complex-pod",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			itemNode, err := structured.FromYAML(tc.itemYAML)
			if err != nil {
				t.Fatalf("failed to parse itemYAML: %v", err)
			}
			itemReader := structured.NewNodeReader(itemNode)

			var prevReader *structured.NodeReader
			if tc.prevYAML != "" {
				prevNode, err := structured.FromYAML(tc.prevYAML)
				if err != nil {
					t.Fatalf("failed to parse prevYAML: %v", err)
				}
				prevReader = structured.NewNodeReader(prevNode)
			}

			gotReader, err := constructResourceBodyFromListItem(itemReader, prevReader)
			if (err != nil) != tc.wantErr {
				t.Fatalf("constructResourceBodyFromListItem() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if got := gotReader.ReadStringOrDefault(pathAPIVersion, ""); got != tc.wantAPIVer {
				t.Errorf("apiVersion mismatch (-want +got):\n%s", cmp.Diff(tc.wantAPIVer, got))
			}
			if got := gotReader.ReadStringOrDefault(pathKind, ""); got != tc.wantKind {
				t.Errorf("kind mismatch (-want +got):\n%s", cmp.Diff(tc.wantKind, got))
			}
			if got := gotReader.ReadStringOrDefault(pathMetadataName, ""); got != tc.wantMetaName {
				t.Errorf("metadata.name mismatch (-want +got):\n%s", cmp.Diff(tc.wantMetaName, got))
			}

			yamlBytes, err := gotReader.Serialize(structured.EmptyFieldPath, &structured.YAMLNodeSerializer{})
			if err != nil {
				t.Fatalf("Serialize() failed: %v", err)
			}
			if diff := cmp.Diff(tc.wantYAML, string(yamlBytes)); diff != "" {
				t.Errorf("YAML serialization mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
