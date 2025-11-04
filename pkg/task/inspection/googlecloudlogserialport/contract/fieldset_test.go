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

package googlecloudlogserialport_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestFieldSet(t *testing.T) {
	testCases := []struct {
		desc         string
		log          string
		wantFieldSet *GCESerialPortLogFieldSet
	}{
		{
			desc: "all fields present",
			log: `logName: projects/project-foo/logs/serialconsole.googleapis.com%2Fserial_port_1_output
textPayload: bar
labels:
  compute.googleapis.com/resource_name: node-name-qux`,
			wantFieldSet: &GCESerialPortLogFieldSet{
				Message:  "bar",
				Port:     "serial_port_1_output",
				NodeName: "node-name-qux",
			},
		},
		{
			desc: "missing node namelabel",
			log: `logName: projects/project-foo/logs/serialconsole.googleapis.com%2Fserial_port_1_output
textPayload: bar`,
			wantFieldSet: &GCESerialPortLogFieldSet{
				Message:  "bar",
				Port:     "serial_port_1_output",
				NodeName: "unknown",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.MustLogFromYAML(tc.log, &GCESerialPortLogFieldSetReader{})
			gotFieldSet := log.MustGetFieldSet(l, &GCESerialPortLogFieldSet{})

			if diff := cmp.Diff(tc.wantFieldSet, gotFieldSet); diff != "" {
				t.Errorf("MustGetFieldSet() got diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSerialPortSpecialSequenceConverter(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strip ansi escape sequences",
			input: "\\x1b[31mthis is red text\\x1b[0m",
			want:  "this is red text",
		},
		{
			name:  "strip \\r\\n sequences",
			input: "this is\\r\\n text\\r\\n",
			want:  "this is text",
		},
		{
			name:  "strip \\x1bM sequences",
			input: "this is\\x1bM text\\x1bM",
			want:  "this is text",
		},
		{
			name:  "strip \\t sequences",
			input: "this is\\t text\\t",
			want:  "this is  text ",
		},
		{
			name:  "strip \\x2d sequences",
			input: "this is\\x2d text\\x2d",
			want:  "this is- text-",
		},
		{
			name:  "unicode unquote",
			input: "Job cri-containerd-06a622d26bbe9788\\xe2\\x80\\xa6/stop running (1min 7s / 1min 30s)",
			want:  "Job cri-containerd-06a622d26bbe9788…/stop running (1min 7s / 1min 30s)",
		},
		{
			name:  "unicode and the hyphen escape sequence",
			input: `         Unmounting \x1b[0;1;39mvar-lib-kubelet\xe2\x80\xa6-collection\\x2dsecret.mount\x1b[0m...\r\n`,
			want:  "         Unmounting var-lib-kubelet…-collection-secret.mount...",
		},
		{
			name:  "strip prefix timestamp for journal",
			input: `2025-09-03T11:25:30+0000 gke-p0-gke-basic-1-default-9a2bebb4-ckm8 kubelet[1959]: E0903 11:25:30.474268    1959 configmap.go:199] Couldn\''t get configMap gmp-system/collector: object \"gmp-system\"/\"collector\" not registered\r\n`,
			want:  `kubelet[1959]: E0903 11:25:30.474268    1959 configmap.go:199] Couldn\''t get configMap gmp-system/collector: object \"gmp-system\"/\"collector\" not registered`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := logutil.ConvertSpecialSequences(tc.input, serialportSequenceConverters...)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("the result is not matching with the expected result\n%s", diff)
			}
		})
	}
}
