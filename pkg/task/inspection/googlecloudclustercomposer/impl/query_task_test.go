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
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestLogFiltersGeneratesComposerQuery(t *testing.T) {
	ctx := context.Background()
	projectId := "test-project"
	environmentName := "test-environment"
	taskDependentValues := typedmap.NewTypedMap()
	typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](googlecloudcommon_contract.InputProjectIdTaskID.ReferenceIDString()), projectId)
	typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.ReferenceIDString()), environmentName)
	ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)

	expected := `log_id("airflow-scheduler")
resource.type="cloud_composer_environment"
resource.labels.project_id="test-project"
resource.labels.environment_name="test-environment"`

	setting := &composerListLogEntriesTaskSetting{
		taskId:        googlecloudclustercomposer_contract.ComposerSchedulerLogQueryTaskID,
		queryName:     "Composer Environment/Airflow Scheduler",
		componentName: "airflow-scheduler",
	}

	taskMode := inspectioncore_contract.TaskModeDryRun // any int is fine
	actual, err := setting.LogFilters(ctx, taskMode)
	if err != nil {
		t.Fatalf("LogFilters: %v", err)
	}
	if len(actual) != 1 {
		t.Errorf("Unexpected query count %d", len(actual))
	}
	if actual[0] != expected {
		t.Errorf("LogFilters: expected %q, got %q", expected, actual[0])
	}
}

func TestDependenciesAndDefaultResourceNames(t *testing.T) {
	ctx := context.Background()
	projectId := "test-project"
	taskDependentValues := typedmap.NewTypedMap()
	typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](googlecloudcommon_contract.InputProjectIdTaskID.ReferenceIDString()), projectId)
	ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)

	setting := &composerListLogEntriesTaskSetting{}

	deps := setting.Dependencies()
	if len(deps) != 2 {
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
