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

package containerstatusrecorder

import (
	"context"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection/ioconfig"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/adapter"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/fieldextractor"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	log_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	corev1 "k8s.io/api/core/v1"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestRecordChangeSetForLog(t *testing.T) {
	os.Setenv("TZ", "UTC")
	testutil.InitTestIO()
	testCases := []struct {
		name          string
		resourcePath  string
		logPaths      []string
		manifestPaths []string
		asserters     [][]testchangeset.ChangeSetAsserter
	}{
		{
			name:         "standard pods conatiners to be running to be ready",
			resourcePath: "core/v1#pods#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4",
			logPaths: []string{
				"test/logs/k8s_audit/container_recorder_test_1_1.yaml",
				"test/logs/k8s_audit/container_recorder_test_1_2.yaml",
				"test/logs/k8s_audit/container_recorder_test_1_3.yaml",
			},
			manifestPaths: []string{
				"test/logs/k8s_audit/container_recorder_test_1_1_manifest.yaml",
				"test/logs/k8s_audit/container_recorder_test_1_2_manifest.yaml",
				"test/logs/k8s_audit/container_recorder_test_1_3_manifest.yaml",
			},
			asserters: [][]testchangeset.ChangeSetAsserter{
				{
					&testchangeset.MatchResourcePathSet{
						WantResourcePaths: []string{},
					},
				},
				{
					&testchangeset.MatchResourcePathSet{
						WantResourcePaths: []string{
							"core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#late-startup",
							"core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#nginx",
						},
					},
					&testchangeset.HasRevision{
						ResourcePath: "core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#late-startup",
						WantRevision: history.StagingResourceRevision{
							ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:00:00Z"),
							State:      enum.RevisionStateContainerWaiting,
							Verb:       enum.RevisionVerbContainerWaiting,
							Body: `name: late-startup
state:
  waiting:
    reason: ContainerCreating
    message: ""
  running: null
  terminated: null
lastterminationstate:
  waiting: null
  running: null
  terminated: null
ready: false
restartcount: 0
image: registry.k8s.io/busybox
imageid: ""
containerid: ""
started: false
allocatedresources: {}
resources: null
volumemounts: []
user: null
allocatedresourcesstatus: []
`,
						},
					},
					&testchangeset.HasRevision{
						ResourcePath: "core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#nginx",
						WantRevision: history.StagingResourceRevision{
							ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:00:00Z"),
							State:      enum.RevisionStateContainerWaiting,
							Verb:       enum.RevisionVerbContainerWaiting,
							Body: `name: nginx
state:
  waiting:
    reason: ContainerCreating
    message: ""
  running: null
  terminated: null
lastterminationstate:
  waiting: null
  running: null
  terminated: null
ready: false
restartcount: 0
image: nginx:1.14.2
imageid: ""
containerid: ""
started: false
allocatedresources: {}
resources: null
volumemounts: []
user: null
allocatedresourcesstatus: []
`,
						},
					},
				},
				{
					&testchangeset.MatchResourcePathSet{
						WantResourcePaths: []string{
							"core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#late-startup",
							"core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#nginx",
						},
					},
					&testchangeset.HasRevision{

						ResourcePath: "core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#late-startup",
						WantRevision: history.StagingResourceRevision{
							ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:00:00Z"),
							State:      enum.RevisionStateContainerRunningNonReady,
							Verb:       enum.RevisionVerbContainerNonReady,
							Body: `name: late-startup
state:
  waiting: null
  running:
    startedat: "2024-11-29T11:38:31Z"
  terminated: null
lastterminationstate:
  waiting: null
  running: null
  terminated: null
ready: false
restartcount: 0
image: registry.k8s.io/busybox:latest
imageid: sha256:36a4dca0fe6fb2a5133dc11a6c8907a97aea122613fa3e98be033959a0821a1f
containerid: containerd://a0d5e6840fd995139f7d1b81c59c190bd6668997d0ea917ed49bac8e74ed2312
started: true
allocatedresources: {}
resources: null
volumemounts: []
user: null
allocatedresourcesstatus: []
`,
						}},
					&testchangeset.HasRevision{
						ResourcePath: "core/v1#pod#1-2-deployment-update#nginx-deployment-surge-8655b4b8c5-xf5f4#nginx",
						WantRevision: history.StagingResourceRevision{
							ChangeTime: testutil.MustParseTimeRFC3339("2024-01-01T01:00:00Z"),
							State:      enum.RevisionStateContainerRunningReady,
							Verb:       enum.RevisionVerbContainerReady,
							Body: `name: nginx
state:
  waiting: null
  running:
    startedat: "2024-11-29T11:38:30Z"
  terminated: null
lastterminationstate:
  waiting: null
  running: null
  terminated: null
ready: true
restartcount: 0
image: docker.io/library/nginx:1.14.2
imageid: docker.io/library/nginx@sha256:f7988fb6c02e0ce69257d9bd9cf37ae20a60f1df7563c3a2a6abe24160306b8d
containerid: containerd://5043bea481844f45aa284f214ef5fed1bf71eae0fc83f6633c293db765b3ed1d
started: true
allocatedresources: {}
resources: null
volumemounts: []
user: null
allocatedresourcesstatus: []
`,
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.logPaths) != len(tc.manifestPaths) {
				t.Fatalf("count of logs and manifests is not matching")
			}
			if len(tc.logPaths) != len(tc.asserters) {
				t.Fatalf("count of logs and asserters is not matching")
			}
			var prevPod *corev1.Pod
			builder := history.NewBuilder(&ioconfig.IOConfig{
				ApplicationRoot: "/",
				DataDestination: "/tmp/",
				TemporaryFolder: "/tmp/",
			})
			parsedLogs := []*types.AuditLogParserInput{}
			for i, log := range tc.logPaths {
				yamlStr := testutil.MustReadText(log)
				l := log_test.MustLogEntity(yamlStr)
				extractor := fieldextractor.GCPAuditLogFieldExtractor{}
				rsLog, err := extractor.ExtractFields(context.Background(), l)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				manifestStr := testutil.MustReadText(tc.manifestPaths[i])
				rsLog.ResourceBodyYaml = manifestStr
				store := structuredatastore.OnMemoryStructureDataStore{}
				yamlAdapter := adapter.Yaml(manifestStr)
				rsLog.ResourceBodyReader, err = yamlAdapter.GetReaderBackedByStore(&store)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				parsedLogs = append(parsedLogs, rsLog)
			}

			for i, log := range parsedLogs {
				cs := history.NewChangeSet(log.Log)
				nextPod, err := recordChangeSetForLog(context.Background(), tc.resourcePath, log, prevPod, cs, builder)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				for _, asserter := range tc.asserters[i] {
					asserter.Assert(t, cs)
				}
				prevPod = nextPod
			}
		})
	}
}
