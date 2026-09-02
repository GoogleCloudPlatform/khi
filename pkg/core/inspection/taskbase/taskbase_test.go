// Copyright 2026 Google LLC
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

package inspectiontaskbase

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func mustNewLogFromYAML(t *testing.T, ctx context.Context, yaml string) *log.Log {
	t.Helper()
	idGen, err := khictx.GetValue(ctx, inspectioncore_contract.IDGenerator)
	if err != nil {
		idGen = id.NewGenerator()
	}
	l, err := log.NewLogFromYAMLString(idGen, yaml)
	if err != nil {
		t.Fatalf("failed to create log from YAML: %v", err)
	}
	return l
}
