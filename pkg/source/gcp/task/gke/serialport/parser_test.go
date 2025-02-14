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

package serialport

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/parserutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	parser_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/parser"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestSerialPortLogParser_ParseBasicSerialPortLog(t *testing.T) {
	wantLogSummary := "[ OK ] Stopped getty@tty1.service."

	cs, err := parser_test.ParseFromYamlLogFile("test/logs/serialport/basic-serialport-log.yaml", &SerialPortLogParser{}, nil, nil)
	if err != nil {
		t.Errorf("got error %v, want nil", err)
	}

	event := cs.GetEvents(resourcepath.NodeSerialport("gke-sample-cluster-default-abcdefgh-abcd"))
	if len(event) != 1 {
		t.Errorf("got %d events, want 1", len(event))
	}

	gotLogSummary := cs.GetLogSummary()
	if gotLogSummary != wantLogSummary {
		t.Errorf("got %q log summary, want %q", gotLogSummary, wantLogSummary)
	}
}

func TestSerialPortSpecialSequenceConverter(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strip ansi escape sequences",
			input:    "\\x1b[31mthis is red text\\x1b[0m",
			expected: "this is red text",
		},
		{
			name:     "strip \\r\\n sequences",
			input:    "this is\\r\\n text\\r\\n",
			expected: "this is text",
		},
		{
			name:     "strip \\x1bM sequences",
			input:    "this is\\x1bM text\\x1bM",
			expected: "this is text",
		},
		{
			name:     "strip \\t sequences",
			input:    "this is\\t text\\t",
			expected: "this is  text ",
		},
		{
			name:     "strip \\x2d sequences",
			input:    "this is\\x2d text\\x2d",
			expected: "this is- text-",
		},
		{
			name:     "unicode unquote",
			input:    "Job cri-containerd-06a622d26bbe9788\\xe2\\x80\\xa6/stop running (1min 7s / 1min 30s)",
			expected: "Job cri-containerd-06a622d26bbe9788…/stop running (1min 7s / 1min 30s)",
		},
		{
			name:     "unicode unquote",
			input:    "Job cri-containerd-06a622d26bbe9788\\xe2\\x80\\xa6/stop running (1min 7s / 1min 30s)",
			expected: "Job cri-containerd-06a622d26bbe9788…/stop running (1min 7s / 1min 30s)",
		},
		{
			name:     "unicode and the hyphen escape sequence",
			input:    `         Unmounting \x1b[0;1;39mvar-lib-kubelet\xe2\x80\xa6-collection\\x2dsecret.mount\x1b[0m...\r\n`,
			expected: "         Unmounting var-lib-kubelet…-collection-secret.mount...",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := parserutil.ConvertSpecialSequences(tc.input, serialportSequenceConverters...)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("the result is not matching with the expected result\n%s", diff)
			}
		})
	}

}
