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
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
)

// InitializerIDK8sMergeConfig generates default merge configs.
const InitializerIDK8sMergeConfig coreinit.InitializerID = "khi.default/k8s-merge-config"

// K8sMergeConfigInitializer initializes the default Kubernetes merge configurations.
var K8sMergeConfigInitializer = &coreinit.Initializer{
	ID: InitializerIDK8sMergeConfig,
	Dependencies: []coreinit.InitializerID{
		InitializerIDParameterParse,
	},
	Init: func(ctx *coreinit.InitContext) error {
		k8s.GenerateDefaultMergeConfig()
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(K8sMergeConfigInitializer)
}
