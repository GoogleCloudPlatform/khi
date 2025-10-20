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

package coreinit

import (
	"slices"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
)

// Registered init extensions.
var initExtensions = map[int]InitExtension{}

// InitExtension provides the extensible points for initialization step.
type InitExtension interface {
	// BeforeAll is called before starting everything used for initializing tools for this process itself(e.g profiler, logger, ...etc)
	BeforeAll() error

	// ConfigureParameterStore is called after calling BeforeAll(). It used for registering own parameter store.
	ConfigureParameterStore() error

	// AfterParsingParameters is called after parsing parameters.
	AfterParsingParameters() error

	// ConfigureInspectionTaskServer is called after AfterParsingParameters to configure the task server.
	ConfigureInspectionTaskServer(taskServer *coreinspection.InspectionTaskServer) error

	// ConfigureKHIWebServerFactory is called after ConfigureInspectionTaskServer to configure the web server factory.
	ConfigureKHIWebServerFactory(serverFactory *server.ServerFactory) error

	// BeforeTerminate is called before terminating KHI process.
	BeforeTerminate() error
}

// RegisterInitExtension registers an InitExtension with a specified order.
func RegisterInitExtension(order int, extension InitExtension) {
	initExtensions[order] = extension
}

// CallInitExtension iterates through registered InitExtensions in order and calls the provided function for each.
func CallInitExtension(caller func(e InitExtension) error) error {
	return callInitExtensionInternal(initExtensions, caller)
}

func callInitExtensionInternal(extensions map[int]InitExtension, caller func(e InitExtension) error) error {
	var keys []int
	for k := range extensions {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, key := range keys {
		extension := extensions[key]
		err := caller(extension)
		if err != nil {
			return err
		}
	}
	return nil
}
