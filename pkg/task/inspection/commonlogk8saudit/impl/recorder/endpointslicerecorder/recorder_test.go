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

package endpointslicerecorder

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	goyaml "gopkg.in/yaml.v3"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestParseSingleEndpoint(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name              string
		endpoint          *model.EndpointSliceEndpoint
		prevEndpointSlice *model.EndpointSlice
		expected          *singleEndpointParseResult
	}{
		{
			name: "Endpoint for Pod, condition changed to Ready",
			endpoint: &model.EndpointSliceEndpoint{
				Conditions: &model.EndpointSliceEndpointConditions{
					Ready:       true,
					Serving:     true,
					Terminating: false,
				},
				TargetRef: &model.K8sTargetRef{
					Kind:      "Pod",
					Name:      "pod-name",
					Namespace: "pod-namespace",
				},
			},
			prevEndpointSlice: &model.EndpointSlice{},
			expected: &singleEndpointParseResult{
				isEndpointForPod:   true,
				hasCoditionChanged: true,
				state:              enum.RevisionStateEndpointReady,
				verb:               enum.RevisionVerbReady,
			},
		},
		{
			name: "Same condition from previous",
			endpoint: &model.EndpointSliceEndpoint{
				Conditions: &model.EndpointSliceEndpointConditions{
					Ready:       true,
					Serving:     true,
					Terminating: false,
				},
				TargetRef: &model.K8sTargetRef{
					Kind:      "Pod",
					Name:      "pod-name",
					Namespace: "pod-namespace",
				},
			},
			prevEndpointSlice: &model.EndpointSlice{
				Endpoints: []*model.EndpointSliceEndpoint{
					{
						Conditions: &model.EndpointSliceEndpointConditions{
							Ready:       true,
							Serving:     true,
							Terminating: false,
						},
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
				},
			},
			expected: &singleEndpointParseResult{
				isEndpointForPod:   true,
				hasCoditionChanged: false,
				state:              enum.RevisionStateConditionUnknown,
				verb:               enum.RevisionVerbUnknown,
			},
		},
		{
			name: "Endpoint for Pod, condition changed to Terminating",
			endpoint: &model.EndpointSliceEndpoint{
				Conditions: &model.EndpointSliceEndpointConditions{
					Ready:       false,
					Serving:     true,
					Terminating: true,
				},
				TargetRef: &model.K8sTargetRef{
					Kind:      "Pod",
					Name:      "pod-name",
					Namespace: "pod-namespace",
				},
			},
			prevEndpointSlice: &model.EndpointSlice{},
			expected: &singleEndpointParseResult{
				isEndpointForPod:   true,
				hasCoditionChanged: true,
				state:              enum.RevisionStateEndpointTerminating,
				verb:               enum.RevisionVerbTerminating,
			},
		},
		{
			name: "Endpoint for Pod, condition changed to Unready",
			endpoint: &model.EndpointSliceEndpoint{
				Conditions: &model.EndpointSliceEndpointConditions{
					Ready:       false,
					Serving:     false,
					Terminating: false,
				},
				TargetRef: &model.K8sTargetRef{
					Kind:      "Pod",
					Name:      "pod-name",
					Namespace: "pod-namespace",
				},
			},
			prevEndpointSlice: &model.EndpointSlice{
				Endpoints: []*model.EndpointSliceEndpoint{
					{
						Conditions: &model.EndpointSliceEndpointConditions{
							Ready:       true,
							Serving:     true,
							Terminating: true,
						},
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
				},
			},
			expected: &singleEndpointParseResult{
				isEndpointForPod:   true,
				hasCoditionChanged: true,
				state:              enum.RevisionStateEndpointUnready,
				verb:               enum.RevisionVerbNonReady,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expected.manifest == "" {
				marshalled, err := goyaml.Marshal(tc.endpoint)
				if err != nil {
					t.Fatalf("failed to marshal endpoint: %v", err)
				}
				tc.expected.manifest = string(marshalled)
			}

			result, err := parseSingleEndpoint(ctx, tc.endpoint, tc.prevEndpointSlice)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expected, result, cmp.AllowUnexported(singleEndpointParseResult{}), cmpopts.IgnoreFields(singleEndpointParseResult{}, "manifest")); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}
		})
	}
}

func TestParseEndpointsOfEndpointSlice(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		endpointSlice  *model.EndpointSlice
		expected       *endpointsParseResult
		expectedErrMsg string
	}{
		{
			name: "only a ready endpoint",
			endpointSlice: &model.EndpointSlice{
				Endpoints: []*model.EndpointSliceEndpoint{
					{
						Conditions: &model.EndpointSliceEndpointConditions{
							Ready:       true,
							Serving:     true,
							Terminating: false,
						},
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
				},
			},
			expected: &endpointsParseResult{
				state:            enum.RevisionStateEndpointReady,
				verb:             enum.RevisionVerbReady,
				hasConditionInfo: true,
			},
		},
		{
			name: "ready and terminating endpoint",
			endpointSlice: &model.EndpointSlice{
				Endpoints: []*model.EndpointSliceEndpoint{
					{
						Conditions: &model.EndpointSliceEndpointConditions{
							Ready:       false,
							Serving:     true,
							Terminating: true,
						},
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
					{
						Conditions: &model.EndpointSliceEndpointConditions{
							Ready:       true,
							Serving:     true,
							Terminating: false,
						},
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
				},
			},
			expected: &endpointsParseResult{
				state:            enum.RevisionStateEndpointReady,
				verb:             enum.RevisionVerbReady,
				hasConditionInfo: true,
			},
		}, {
			name: "only terminating endpoints",
			endpointSlice: &model.EndpointSlice{
				Endpoints: []*model.EndpointSliceEndpoint{
					{
						Conditions: &model.EndpointSliceEndpointConditions{
							Ready:       false,
							Serving:     true,
							Terminating: true,
						},
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
					{
						Conditions: &model.EndpointSliceEndpointConditions{
							Ready:       false,
							Serving:     true,
							Terminating: true,
						},
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
				},
			},
			expected: &endpointsParseResult{
				state:            enum.RevisionStateEndpointTerminating,
				verb:             enum.RevisionVerbTerminating,
				hasConditionInfo: true,
			},
		},
		{
			name: "for endpoints without condition",
			endpointSlice: &model.EndpointSlice{
				Endpoints: []*model.EndpointSliceEndpoint{
					{
						Conditions: nil,
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
					{
						Conditions: nil,
						TargetRef: &model.K8sTargetRef{
							Kind:      "Pod",
							Name:      "pod-name",
							Namespace: "pod-namespace",
						},
					},
				},
			},
			expected: &endpointsParseResult{
				state:            enum.RevisionStateEndpointUnready,
				verb:             enum.RevisionVerbUnknown,
				hasConditionInfo: false,
			},
		},
		{
			name: "without any endpoints",
			endpointSlice: &model.EndpointSlice{
				Endpoints: []*model.EndpointSliceEndpoint{},
			},
			expected: &endpointsParseResult{
				state:            enum.RevisionStateEndpointUnready,
				verb:             enum.RevisionVerbNonReady,
				hasConditionInfo: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expected.manifest == "" {
				marshalled, err := goyaml.Marshal(tc.endpointSlice)
				if err != nil {
					t.Fatalf("failed to marshal endpointSlice: %v", err)
				}
				tc.expected.manifest = string(marshalled)
			}

			result, err := parseEndpointsOfEndpointSlice(ctx, tc.endpointSlice)
			if tc.expectedErrMsg != "" {
				if err == nil || err.Error() != tc.expectedErrMsg {
					t.Fatalf("expected error message: %q, got: %v", tc.expectedErrMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expected, result, cmp.AllowUnexported(endpointsParseResult{}), cmpopts.IgnoreFields(endpointsParseResult{}, "manifest")); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}

		})
	}
}
