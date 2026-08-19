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

type dummyParameterStore struct {
	prepared bool
}

func (d *dummyParameterStore) Prepare() error {
	d.prepared = true
	return nil
}

func (d *dummyParameterStore) PostProcess() error {
	return nil
}

var _ parameters.ParameterStore = (*dummyParameterStore)(nil)

func TestRegisterParameterStore(t *testing.T) {
	coreinit.ResetInitializersForTest()
	parameters.ResetStore()
	defer func() {
		coreinit.ResetInitializersForTest()
		parameters.ResetStore()
	}()

	dummy := &dummyParameterStore{}
	RegisterParameterStore("dummy-store", dummy)

	parseRan := false
	coreinit.RegisterInitializer(&coreinit.Initializer{
		ID: InitializerIDParameterParse,
		Init: func(ctx *coreinit.InitContext) error {
			parseRan = true
			return nil
		},
	})

	engine := coreinit.NewEngine(context.Background())
	if err := engine.Init(); err != nil {
		t.Fatalf("Engine.Init() failed: %v", err)
	}

	if !parseRan {
		t.Errorf("expected parse initializer to run")
	}
}
