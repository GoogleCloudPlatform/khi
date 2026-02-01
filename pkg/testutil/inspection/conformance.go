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

package inspection_test

import (
	"fmt"
	"strings"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func ConformanceTestForInspectionTypes(t *testing.T) {
	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	testServer, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	err = generated.RegisterAllInspectionTasks(testServer)
	if err != nil {
		t.Fatalf("unexpected error %v. failed to complete the preparation step", err)
	}

	for _, inspectionType := range testServer.GetAllInspectionTypes() {
		t.Run(fmt.Sprintf("%s-contains-at-least-one-feature", inspectionType.Name), func(t *testing.T) {
			taskId, err := testServer.CreateInspection(inspectionType.Id)
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			features, err := testServer.GetInspection(taskId).FeatureList()
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			if len(features) == 0 {
				t.Errorf("feature=`%s` had no feature", inspectionType.Name)
			}
			result := ""
			for _, feature := range features {
				result += fmt.Sprintf("* %s", feature.Label)
			}
			fmt.Printf("Feature=%s\n%s\n", inspectionType.Id, result)
		})

		// icons must be in relative path for frontend to read it when the base path was rewritten
		t.Run(fmt.Sprintf("%s-icon-must-be-relative-path", inspectionType.Name), func(t *testing.T) {
			if strings.HasPrefix(inspectionType.Icon, "/") {
				t.Errorf("icon path must be relative path, got %s", inspectionType.Icon)
			}
		})
	}
}
