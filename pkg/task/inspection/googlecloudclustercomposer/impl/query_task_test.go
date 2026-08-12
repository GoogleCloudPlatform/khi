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

package googlecloudclustercomposer_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestLogFiltersGeneratesComposerQuery(t *testing.T) {
	testCases := []struct {
		name               string
		selectedComponents []string
		wantQuery          string
	}{
		{
			name:               "specific component selected",
			selectedComponents: []string{"scheduler"},
			wantQuery: `(LOG_ID("scheduler"))
resource.type="cloud_composer_environment"
resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.environment_name="test-environment"

-LOG_ID("cloudaudit.googleapis.com/activity")
-LOG_ID("cloudaudit.googleapis.com/data_access")
`,
		},
		{
			name:               "any component selected",
			selectedComponents: []string{"@any"},
			wantQuery: `-- no component filter specified. fetching all logs.
resource.type="cloud_composer_environment"
resource.labels.project_id="test-project"
resource.labels.location="test-location"
resource.labels.environment_name="test-environment"

-LOG_ID("cloudaudit.googleapis.com/activity")
-LOG_ID("cloudaudit.googleapis.com/data_access")
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			projectId := "test-project"
			environmentName := "test-environment"
			taskDependentValues := typedmap.NewTypedMap()
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[googlecloudk8scommon_contract.GoogleCloudClusterIdentity](googlecloudclustercomposer_contract.ClusterIdentityTaskID.ReferenceIDString()), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{ProjectID: projectId, Location: "test-location"})
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.ReferenceIDString()), environmentName)
			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[[]string](googlecloudclustercomposer_contract.InputComposerComponentsTaskID.ReferenceIDString()), tc.selectedComponents)
			ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)

			setting := &composerListLogEntriesTaskSetting{
				taskId:    googlecloudclustercomposer_contract.ComposerLogsQueryTaskID,
				queryName: "Composer Logs",
			}

			taskMode := inspectioncore_contract.TaskModeDryRun
			actual, err := setting.LogFilters(ctx, taskMode)
			if err != nil {
				t.Fatalf("LogFilters() returned unexpected error: %v", err)
			}
			if len(actual) != 1 {
				t.Fatalf("LogFilters() returned unexpected query count %d", len(actual))
			}
			if diff := cmp.Diff(tc.wantQuery, actual[0]); diff != "" {
				t.Errorf("LogFilters() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDependenciesAndDefaultResourceNames(t *testing.T) {
	ctx := context.Background()
	projectId := "test-project"
	taskDependentValues := typedmap.NewTypedMap()
	typedmap.Set(taskDependentValues, typedmap.NewTypedKey[googlecloudk8scommon_contract.GoogleCloudClusterIdentity](googlecloudclustercomposer_contract.ClusterIdentityTaskID.ReferenceIDString()), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{ProjectID: projectId})
	ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)

	setting := &composerListLogEntriesTaskSetting{}

	deps := setting.Dependencies()
	if len(deps) != 3 {
		t.Errorf("Unexpected dependencies count %d", len(deps))
	}

	resourceNames, err := setting.DefaultResourceNames(ctx)
	if err != nil {
		t.Fatalf("DefaultResourceNames: %v", err)
	}
	if len(resourceNames) != 1 || resourceNames[0] != "projects/test-project" {
		t.Errorf("Unexpected resource names: %v", resourceNames)
	}
}

func TestDescription(t *testing.T) {
	setting := &composerListLogEntriesTaskSetting{
		queryName: "Test Query",
	}
	desc := setting.Description()
	if desc.QueryName != "Test Query" {
		t.Errorf("Description().QueryName mismatch (-want +got): -%s +%s", "Test Query", desc.QueryName)
	}

	wantExampleQuery := `(LOG_ID("airflow-worker") OR LOG_ID("worker") OR LOG_ID("airflow-scheduler") OR LOG_ID("scheduler"))
resource.type="cloud_composer_environment"
resource.labels.project_id="test-project"
resource.labels.environment_name="sample-composer-environment"

-LOG_ID("cloudaudit.googleapis.com/activity")
-LOG_ID("cloudaudit.googleapis.com/data_access")`

	if diff := cmp.Diff(wantExampleQuery, desc.ExampleQuery); diff != "" {
		t.Errorf("Description().ExampleQuery mismatch (-want +got):\n%s", diff)
	}
}
