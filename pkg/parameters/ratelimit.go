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

package parameters

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/flag"
)

// RateLimit is the singleton instance of RateLimitParameters.
var RateLimit *RateLimitParameters = &RateLimitParameters{}

// RateLimitParameters stores parameters related to API rate limiting.
type RateLimitParameters struct {
	// LoggingInitialQPS is the starting request rate in QPS for Cloud Logging.
	LoggingInitialQPS *float64
	// LoggingMinQPS is the minimum request rate in QPS for Cloud Logging.
	LoggingMinQPS *float64
	// LoggingMaxQPS is the maximum request rate in QPS for Cloud Logging.
	LoggingMaxQPS *float64
	// LoggingAdaptiveRateLimitEnabled specifies whether adaptive rate limiting is enabled for Cloud Logging.
	LoggingAdaptiveRateLimitEnabled *bool
}

// Prepare implements ParameterStore.
func (r *RateLimitParameters) Prepare() error {
	r.LoggingInitialQPS = flag.Float64("logging-initial-qps", 1.0, "The initial QPS for Cloud Logging API queries.", "")
	r.LoggingMinQPS = flag.Float64("logging-min-qps", 1.0, "The minimum QPS for Cloud Logging API queries.", "")
	r.LoggingMaxQPS = flag.Float64("logging-max-qps", 100.0, "The maximum QPS for Cloud Logging API queries.", "")
	r.LoggingAdaptiveRateLimitEnabled = flag.Bool("logging-adaptive-rate-limit-enabled", true, "Enable adaptive rate limiting for Cloud Logging API queries.", "")
	return nil
}

// PostProcess implements ParameterStore.
func (r *RateLimitParameters) PostProcess() error {
	if *r.LoggingMinQPS <= 0 {
		return fmt.Errorf("--logging-min-qps must be positive")
	}
	if *r.LoggingMaxQPS < *r.LoggingMinQPS {
		return fmt.Errorf("--logging-max-qps must be greater than or equal to --logging-min-qps")
	}
	if *r.LoggingInitialQPS < *r.LoggingMinQPS || *r.LoggingInitialQPS > *r.LoggingMaxQPS {
		return fmt.Errorf("--logging-initial-qps must be between --logging-min-qps and --logging-max-qps")
	}
	return nil
}

var _ ParameterStore = (*RateLimitParameters)(nil)
