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

package parameters

import (
	"errors"

	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
)

var Debug = &DebugParameters{}

// DebugParameters is the ParameterStore for debug purpose parameters.
type DebugParameters struct {
	// Profiler decides if KHI uses CloudProfiler or not.
	Profiler *bool
	// ProfilerService is the service name given to CloudProfiler.
	ProfilerService *string
	// ProfilerProject is the GCP project ID where the profiler sends the data to.
	ProfilerProject *string

	// Verbose
	// If this flag is set, KHI prints verbose logs.
	Verbose *bool

	// NoColor
	// If this flag is set, KHI prints logs without color.
	NoColor *bool
}

// PostProcess implements ParameterStore.
func (d *DebugParameters) PostProcess() error {
	if *d.Profiler && (d.ProfilerProject == nil || *d.ProfilerProject == "") {
		return errors.New("--profiler-project is required when --profiler is set")
	}
	return nil
}

// Prepare implements ParameterStore.
func (d *DebugParameters) Prepare() error {
	d.Profiler = flag.Bool("profiler", false, "Decides if KHI uses CloudProfiler or not.", "")
	d.ProfilerProject = flag.String("profiler-project", "", "The GCP project ID where the profiler sends the data to.", "")
	d.ProfilerService = flag.String("profiler-service", "khi", "The service name given to CloudProfiler.", "")
	d.Verbose = flag.Bool("verbose", false, "If this flag is set, KHI prints verbose logs.", "")
	d.NoColor = flag.Bool("no-color", false, "If this flag is set, KHI prints logs without color.", "")
	return nil
}

var _ ParameterStore = (*DebugParameters)(nil)
