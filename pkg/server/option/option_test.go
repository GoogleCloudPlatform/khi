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

package option

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// mockOption is a helper for testing.
type mockOption struct {
	id    string
	order int
	apply func(e *gin.Engine) error
}

func (m *mockOption) ID() string {
	return m.id
}

func (m *mockOption) Order() int {
	return m.order
}

func (m *mockOption) Apply(engine *gin.Engine) error {
	if m.apply != nil {
		return m.apply(engine)
	}
	return nil
}

func TestApplyOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		options     []Option
		expectOrder []string
		wantErr     bool
	}{
		{
			name: "should apply in correct order",
			options: []Option{
				&mockOption{id: "option-2", order: 2},
				&mockOption{id: "option-1", order: 1},
				&mockOption{id: "option-3", order: 3},
			},
			expectOrder: []string{"option-1", "option-2", "option-3"},
			wantErr:     false,
		},
		{
			name:        "should handle empty options",
			options:     []Option{},
			expectOrder: []string{},
			wantErr:     false,
		},
		{
			name: "should return error on apply failure",
			options: []Option{
				&mockOption{id: "good-option", order: 1},
				&mockOption{id: "bad-option", order: 2, apply: func(e *gin.Engine) error {
					return errors.New("apply failed")
				}},
			},
			expectOrder: []string{"good-option", "bad-option"}, // bad-option will not be used but it must be called once in the order.
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			var appliedOrder []string

			// Wrap mock options to record apply calls
			recordingOptions := make([]Option, len(tc.options))
			for i, opt := range tc.options {
				mock, ok := opt.(*mockOption)
				if !ok {
					t.Fatalf("test setup error: expected mockOption")
				}
				// copy mock to avoid closure issues
				originalApply := mock.apply
				recordingOptions[i] = &mockOption{
					id:    mock.id,
					order: mock.order,
					apply: func(e *gin.Engine) error {
						appliedOrder = append(appliedOrder, mock.id)
						if originalApply != nil {
							return originalApply(e)
						}
						return nil
					},
				}
			}

			err := ApplyOptions(engine, recordingOptions)

			if (err != nil) != tc.wantErr {
				t.Fatalf("ApplyOptions() error = %v, wantErr %v", err, tc.wantErr)
			}

			if fmt.Sprint(appliedOrder) != fmt.Sprint(tc.expectOrder) {
				t.Errorf("ApplyOptions() applied in wrong order. got=%v, want=%v", appliedOrder, tc.expectOrder)
			}
		})
	}
}

func TestCorsOption(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := cors.Config{
		AllowOrigins: []string{"http://localhost:4200"},
	}

	opt := CORS(config)
	engine := gin.New()

	if err := opt.Apply(engine); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Check ID and Order
	if opt.ID() != "cors" {
		t.Errorf("ID() got = %q, want = \"cors\"", opt.ID())
	}
	if opt.Order() != 1 {
		t.Errorf("Order() got = %d, want = 1", opt.Order())
	}

	// Check if CORS header is present by making a request
	engine.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	gotHeader := w.Header().Get("Access-Control-Allow-Origin")
	if gotHeader != "http://localhost:4200" {
		t.Errorf("Access-Control-Allow-Origin header not set correctly. got=%q, want=%q", gotHeader, "http://localhost:4200")
	}
}
