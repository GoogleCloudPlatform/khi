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

package componentparser

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func getControlPlaneComponentLogForTesting(message string) string {
	return fmt.Sprintf(`insertId: xxxxxxxx
jsonPayload:
  message: %s
labels:
  gke.googleapis.com/log_type: system
logName: projects/project-foo/logs/gkemulticloud.googleapis.com%%2Fkube-controller-manager
receiveTimestamp: "2024-08-09T15:19:38.137546263Z"
resource:
  labels:
    cluster_name: awsClusters/cluster-foo
    component_location: ap-northeast-1a
    component_name: controller-manager
    location: asia-southeast1
    project_id: project-foo
  type: k8s_control_plane_component
severity: INFO
timestamp: "2024-08-09T15:19:37.511717902Z"`, message)
}

func TestEventLogToResourcePath(t *testing.T) {
	testCases := []struct {
		testName      string
		inputLog      string
		expectedPath  string
		expectedError bool
	}{
		{
			testName:      "Normal case",
			inputLog:      getControlPlaneComponentLogForTesting(`Event occurred" object="namespace-foo/name-bar" fieldPath="" kind="Job" apiVersion="batch/v1" type="Normal" reason="Completed" message="Job completed"`),
			expectedPath:  "batch/v1#job#namespace-foo#name-bar",
			expectedError: false,
		},
		{
			testName:      "Normal case with double quote at the beginning",
			inputLog:      getControlPlaneComponentLogForTesting("\"\\\"Event occurred\\\" object=\\\"namespace-foo/name-bar\\\" fieldPath=\\\"\\\" kind=\\\"Job\\\" apiVersion=\\\"batch/v1\\\" type=\\\"Normal\\\" reason=\\\"Completed\\\" message=\\\"Job completed\\\"\""),
			expectedPath:  "batch/v1#job#namespace-foo#name-bar",
			expectedError: false,
		},
		{
			testName:      "Cluster scope resource",
			inputLog:      getControlPlaneComponentLogForTesting(`Event occurred" object="name-bar" fieldPath="" kind="Node" apiVersion="v1" type="Normal" reason="Completed" message="Job completed"`),
			expectedPath:  "core/v1#node#cluster-scope#name-bar",
			expectedError: false,
		},
		{
			testName:      "Missing object",
			inputLog:      getControlPlaneComponentLogForTesting(`Event occurred" fieldPath="" kind="Job" apiVersion="batch/v1" type="Normal" reason="Completed" message="Job completed"`),
			expectedPath:  "",
			expectedError: true,
		},

		{
			testName:      "Missing kind",
			inputLog:      getControlPlaneComponentLogForTesting(`Event occurred" object="namespace-foo/name-bar"  fieldPath=""  apiVersion="batch/v1" type="Normal" reason="Completed" message="Job completed"`),
			expectedPath:  "",
			expectedError: true,
		},
		{
			testName:      "Missing apiVersion",
			inputLog:      getControlPlaneComponentLogForTesting(`Event occurred" object="namespace-foo/name-bar"fieldPath="" kind="Job"  type="Normal" reason="Completed" message="Job completed"`),
			expectedPath:  "",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			parser := &ControllerManagerComponentParser{}
			l := testlog.MustLogFromYAML(tc.inputLog, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
			path, err := parser.eventLogToResourcePath(l)
			if tc.expectedError {
				if err == nil {
					t.Errorf("expected an error but no error returned")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if diff := cmp.Diff(tc.expectedPath, path.Path); diff != "" {
				t.Errorf("the result path is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", tc.inputLog, path.Path, tc.expectedPath)
			}
		})
	}
}

func TestControllerLogToResourcePath(t *testing.T) {
	testCases := []struct {
		testName      string
		inputLog      string
		expectedPaths []string
		expectedError bool
	}{
		{
			testName:      "Deployment resources",
			inputLog:      getControlPlaneComponentLogForTesting("'\"Error syncing deployment\" deployment=\"kube-system/konnectivity-agent\" err=\"Operation cannot be fulfilled on deployments.apps \\\"konnectivity-agent\\\": the object has been modified; please apply your changes to the latest version and try again\"'"),
			expectedPaths: []string{"apps/v1#deployment#kube-system#konnectivity-agent"},
			expectedError: false,
		},
		{
			testName:      "ReplicaSet resources",
			inputLog:      getControlPlaneComponentLogForTesting("'\"Too few replicas\" replicaSet=\"kube-system/konnectivity-agent-6d4c7f658c\"s need=1 creating=1'"),
			expectedPaths: []string{"apps/v1#replicaset#kube-system#konnectivity-agent-6d4c7f658c"},
			expectedError: false,
		},
		{
			testName:      "Node resources",
			inputLog:      getControlPlaneComponentLogForTesting("'\"attacherDetacher.DetachVolume started\" node=\"gke-gke-basic-1-default-94a087a3-gt98\" volumeName=\"kubernetes.io/csi/pd.csi.storage.gke.io^projects/khi-testing/zones/us-central1-a/disks/pvc-42ecd5dc-07c0-4b93-88c7-c60bc789c96d\"'"),
			expectedPaths: []string{"core/v1#node#cluster-scope#gke-gke-basic-1-default-94a087a3-gt98"},
			expectedError: false,
		},
		{
			testName:      "Namespace resource",
			inputLog:      getControlPlaneComponentLogForTesting("'\"Namespace has been deleted\" namespace=\"1-1-probes\"'"),
			expectedPaths: []string{"core/v1#namespace#cluster-scope#1-1-probes"},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			parser := &ControllerManagerComponentParser{}
			l := testlog.MustLogFromYAML(tc.inputLog, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
			paths, err := parser.controllerLogToResourcePath(l)
			if tc.expectedError {
				if err == nil {
					t.Errorf("expected an error but no error returned")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
			rawPaths := []string{}
			for _, path := range paths {
				rawPaths = append(rawPaths, path.Path)
			}
			if diff := cmp.Diff(tc.expectedPaths, rawPaths); diff != "" {
				t.Errorf("different resource path(-want, +got): %v", diff)
			}
		})
	}
}

func TestKindLogToResourcePath(t *testing.T) {
	testCases := []struct {
		testName      string
		inputLog      string
		expectedPath  string
		expectedError bool
	}{
		{
			testName:      "Basic klog containing kind",
			inputLog:      getControlPlaneComponentLogForTesting(`'"Finished syncing" kind="ReplicaSet" key="1-2-deployment-update/nginx-deployment-non-surge-5644b9c44d" duration="16.358421ms"'`),
			expectedPath:  "apps/v1#replicaset#1-2-deployment-update#nginx-deployment-non-surge-5644b9c44d",
			expectedError: false,
		},
		{
			testName:      "Basic klog containing kind without namespace",
			inputLog:      getControlPlaneComponentLogForTesting(`'"Finished syncing" kind="Node" key="node-foo" duration="16.358421ms"'`),
			expectedPath:  "core/v1#node#cluster-scope#node-foo",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			parser := &ControllerManagerComponentParser{}
			l := testlog.MustLogFromYAML(tc.inputLog, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
			path, err := parser.kindLogToResourcePath(context.Background(), l)
			if tc.expectedError {
				if err == nil {
					t.Errorf("expected an error but no error returned")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if diff := cmp.Diff(tc.expectedPath, path.Path); diff != "" {
				t.Errorf("different resource path (-want,+got):%v", diff)
			}
		})
	}
}
