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
	"flag"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestRateLimitParameters(t *testing.T) {
	testCases := []struct {
		name   string
		want   *RateLimitParameters
		before func()
	}{
		{
			name: "default values",
			want: &RateLimitParameters{
				LoggingInitialQPS:               testutil.P(1.0),
				LoggingMinQPS:                   testutil.P(1.0),
				LoggingMaxQPS:                   testutil.P(100.0),
				LoggingAdaptiveRateLimitEnabled: testutil.P(true),
			},
			before: func() {
				os.Args = []string{os.Args[0]}
				flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prepareFlagParsingTest(t)
			tc.before()
			store := &RateLimitParameters{}
			ResetStore()
			AddStore(store)
			err := Parse()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.want, store); diff != "" {
				t.Errorf("unexpected result (-want +got)\n%s", diff)
			}
		})
	}
}

func TestRateLimitParameters_PostProcess(t *testing.T) {
	testCases := []struct {
		name      string
		params    *RateLimitParameters
		expectErr bool
	}{
		{
			name: "valid default values",
			params: &RateLimitParameters{
				LoggingInitialQPS:               testutil.P(1.0),
				LoggingMinQPS:                   testutil.P(1.0),
				LoggingMaxQPS:                   testutil.P(100.0),
				LoggingAdaptiveRateLimitEnabled: testutil.P(true),
			},
			expectErr: false,
		},
		{
			name: "valid custom values",
			params: &RateLimitParameters{
				LoggingInitialQPS:               testutil.P(5.0),
				LoggingMinQPS:                   testutil.P(1.0),
				LoggingMaxQPS:                   testutil.P(10.0),
				LoggingAdaptiveRateLimitEnabled: testutil.P(true),
			},
			expectErr: false,
		},
		{
			name: "invalid: non-positive min qps",
			params: &RateLimitParameters{
				LoggingInitialQPS:               testutil.P(1.0),
				LoggingMinQPS:                   testutil.P(0.0),
				LoggingMaxQPS:                   testutil.P(100.0),
				LoggingAdaptiveRateLimitEnabled: testutil.P(true),
			},
			expectErr: true,
		},
		{
			name: "invalid: max qps less than min qps",
			params: &RateLimitParameters{
				LoggingInitialQPS:               testutil.P(1.0),
				LoggingMinQPS:                   testutil.P(5.0),
				LoggingMaxQPS:                   testutil.P(2.0),
				LoggingAdaptiveRateLimitEnabled: testutil.P(true),
			},
			expectErr: true,
		},
		{
			name: "invalid: initial qps less than min qps",
			params: &RateLimitParameters{
				LoggingInitialQPS:               testutil.P(0.5),
				LoggingMinQPS:                   testutil.P(1.0),
				LoggingMaxQPS:                   testutil.P(100.0),
				LoggingAdaptiveRateLimitEnabled: testutil.P(true),
			},
			expectErr: true,
		},
		{
			name: "invalid: initial qps greater than max qps",
			params: &RateLimitParameters{
				LoggingInitialQPS:               testutil.P(150.0),
				LoggingMinQPS:                   testutil.P(1.0),
				LoggingMaxQPS:                   testutil.P(100.0),
				LoggingAdaptiveRateLimitEnabled: testutil.P(true),
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.PostProcess()
			if (err != nil) != tc.expectErr {
				t.Errorf("PostProcess() error = %v, expectErr %v", err, tc.expectErr)
			}
		})
	}
}
