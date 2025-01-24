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
	"flag"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestJobParameters(t *testing.T) {
	testCases := []struct {
		name   string
		want   *JobParameters
		before func()
	}{
		{
			name: "default",
			want: &JobParameters{
				JobMode:            testutil.P(false),
				InspectionType:     testutil.P(""),
				InspectionFeatures: testutil.P(""),
				InspectionValues:   testutil.P(""),
				ExportDestination:  testutil.P(""),
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
			store := &JobParameters{}
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
