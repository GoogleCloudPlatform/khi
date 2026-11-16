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
	"os"

	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
)

// Help holds the singleton instance of HelpParameters.
var Help = &HelpParameters{}

var _ ParameterStore = (*HelpParameters)(nil)

// HelpParameters stores parameters for displaying CLI usage help.
type HelpParameters struct {
	// Help prints CLI usage and exits when set.
	Help *bool
}

// PostProcess implements ParameterStore.
func (h *HelpParameters) PostProcess() error {
	if *h.Help {
		flag.PrintDefaults()
		os.Exit(0)
		return nil
	}
	return nil
}

// Prepare implements ParameterStore.
func (h *HelpParameters) Prepare() error {
	h.Help = flag.Bool("help", false, "If this flag is set, KHI exits with printing the usage.", "")
	return nil
}
