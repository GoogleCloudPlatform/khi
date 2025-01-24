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

package gcp

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/common"
	inspection_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/inspection"
)

func testPrepareInspectionServer(inspectionServer *inspection.InspectionTaskServer) error {
	err := commonPreparation(inspectionServer)
	if err != nil {
		return err
	}
	return nil
}

func TestInspectionTasksAreResolvable(t *testing.T) {
	inspection_test.ConformanceEveryInspectionTasksAreResolvable(t, "gcp", []inspection.PrepareInspectionServerFunc{
		common.PrepareInspectionServer,
		testPrepareInspectionServer,
	})
}

func TestConformanceTestForInspectionTypes(t *testing.T) {
	inspection_test.ConformanceTestForInspectionTypes(t, []inspection.PrepareInspectionServerFunc{
		common.PrepareInspectionServer,
		testPrepareInspectionServer,
	})
}
