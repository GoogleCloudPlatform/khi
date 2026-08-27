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

package workbench

import (
	"context"
	"errors"
	"testing"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"google.golang.org/protobuf/proto"
)

func TestWorkbench_GetArchitectureGraph(t *testing.T) {
	testCases := []struct {
		name      string
		setup     func() *Workbench
		req       *apiv1.GetArchitectureGraphRequest
		wantLen   int
		wantErrIs error
		wantErr   bool
	}{
		{
			name: "returns ErrWorkbenchClosed when workbench is closed",
			setup: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.Close()
				return wb
			},
			req: &apiv1.GetArchitectureGraphRequest{
				TimestampNs: proto.Int64(100),
			},
			wantErrIs: ErrWorkbenchClosed,
			wantErr:   true,
		},
		{
			name: "returns error when search index is uninitialized",
			setup: func() *Workbench {
				return NewWorkbench("wb-1", "insp-1")
			},
			req: &apiv1.GetArchitectureGraphRequest{
				TimestampNs: proto.Int64(100),
			},
			wantErr: true,
		},
		{
			name: "successfully builds architecture graph when search index is present",
			setup: func() *Workbench {
				wb := NewWorkbench("wb-1", "insp-1")
				wb.searchIndex = &SearchIndex{
					Timelines:   []*cel.TimelineData{},
					TimelineMap: make(map[uint32]*cel.TimelineData),
				}
				return wb
			},
			req: &apiv1.GetArchitectureGraphRequest{
				TimestampNs: proto.Int64(100),
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wb := tc.setup()
			resp, err := wb.GetArchitectureGraph(context.Background(), tc.req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GetArchitectureGraph() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("GetArchitectureGraph() error = %v, wantErrIs = %v", err, tc.wantErrIs)
			}
			if !tc.wantErr && resp == nil {
				t.Fatalf("GetArchitectureGraph() returned nil response")
			}
		})
	}
}
