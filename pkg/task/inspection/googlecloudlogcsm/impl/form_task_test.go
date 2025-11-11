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

package googlecloudlogcsm_impl

import (
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/google/go-cmp/cmp"
)

func TestConvertInputOnlyResponseFlagToActualFlag(t *testing.T) {
	testCases := []struct {
		desc  string
		input []string
		want  []string
	}{
		{
			desc:  "basic conversion",
			input: []string{"ok", "uh", "ut", "NR"},
			want:  []string{"-", "UH", "UT", "NR"},
		},
		{
			desc:  "empty input",
			input: []string{},
			want:  []string{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := convertInputOnlyResponseFlagToActualFlag(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("convertInputOnlyResponseFlagToActualFlag() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestVerifyResponseFlags(t *testing.T) {
	testCases := []struct {
		desc    string
		input   []string
		wantErr bool
	}{
		{
			desc:    "valid flags",
			input:   []string{"-", "UH", "UT"},
			wantErr: false,
		},
		{
			desc:    "invalid flag",
			input:   []string{"-", "INVALID_FLAG"},
			wantErr: true,
		},
		{
			desc:    "empty input",
			input:   []string{},
			wantErr: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := verifyResponseFlags(tc.input)
			if tc.wantErr {
				if !errors.Is(err, khierrors.ErrInvalidInput) {
					t.Errorf("verifyResponseFlags() error = %v, want = %v", err, khierrors.ErrInvalidInput)
				}
			} else {
				if err != nil {
					t.Errorf("verifyResponseFlags() error = %v, want = nil", err)
				}
			}
		})
	}
}
