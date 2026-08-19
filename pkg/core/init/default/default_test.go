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

package defaultinit

import (
	"context"
	"testing"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

func TestDefaultInitializers_Resolution(t *testing.T) {
	parameters.ResetStore()
	defer parameters.ResetStore()

	engine := coreinit.NewEngine(context.Background())
	ctx := engine.Context()

	if err := engine.Init(); err != nil {
		t.Fatalf("Engine.Init() failed for default initializers: %v", err)
	}

	resolved, found := coreinit.Get(ctx, coreinit.ResolvedInitializersKey)
	if !found {
		t.Errorf("expected ResolvedInitializersKey to be present")
	} else if len(resolved) == 0 {
		t.Errorf("expected resolved initializers to be non-empty")
	}

	inspectionServer, found := coreinit.Get(ctx, InspectionTaskServerKey)
	if !found || inspectionServer == nil {
		t.Errorf("expected InspectionTaskServer to be created and set in context")
	}
}
