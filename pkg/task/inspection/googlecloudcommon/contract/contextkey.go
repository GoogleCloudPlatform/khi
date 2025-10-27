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

package googlecloudcommon_contract

import (
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
)

// APIClientFactoryOptionsContextKey is the key to retrieve googlecloud.ClientFactoryOption from task context.
// The value is injected on the task server during the initialization.
var APIClientFactoryOptionsContextKey = typedmap.NewTypedKey[*[]googlecloud.ClientFactoryOption]("api-client-factory-options")

// APICallOptionsInjectorContextKey is the key to retrieve the list of googlecloud.CallOptionInjectorOption from task context.
// The value is injected on the task server during the initialization.
var APICallOptionsInjectorContextKey = typedmap.NewTypedKey[*[]googlecloud.CallOptionInjectorOption]("api-call-option-injector-options")
