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

var Help = &HelpParameters{}

type HelpParameters struct {
	// Help
	// If this parameter is set, KHI exits with printing the usage.
	Help *bool
}

func (h *HelpParameters) PostProcess() error {
	if *h.Help {
		flag.PrintDefaults()
		os.Exit(0)
		return nil
	}
	return nil
}

func (h *HelpParameters) Prepare() error {
	h.Help = flag.Bool("help", false, "If this flag is set, KHI exits with printing the usage.", "")
	return nil
}

var _ ParameterStore = (*HelpParameters)(nil)
