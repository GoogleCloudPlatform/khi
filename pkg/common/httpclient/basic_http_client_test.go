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

package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type mockHeaderProvider struct {
	Key   string
	Value string
	Error error
}

// AddHeader implements HTTPHeaderProvider.
func (m *mockHeaderProvider) AddHeader(req *http.Request) error {
	req.Header.Add(m.Key, m.Value)
	return m.Error
}

var _ HTTPHeaderProvider = (*mockHeaderProvider)(nil)

func TestBasicHttpClient_DoWithContext(t *testing.T) {
	t.Run("should return response when server returns 200", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer ts.Close()

		client := NewBasicHttpClient()
		req, _ := http.NewRequest("GET", ts.URL, nil)

		resp, err := client.DoWithContext(context.Background(), req)

		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got status code %d, want %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("should add headers with header provider", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("test-key") != "test-value" {
				t.Errorf("Expected header test-key to be test-value, but got %s", r.Header.Get("test-key"))
			}
			if r.Header.Get("test-key-2") != "test-value-2" {
				t.Errorf("Expected header test-key-2 to be test-value-2, but got %s", r.Header.Get("test-key-2"))
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer ts.Close()

		client := NewBasicHttpClient()
		client = client.WithHeaderProvider(
			&mockHeaderProvider{
				Key:   "test-key",
				Value: "test-value",
				Error: nil,
			},
			&mockHeaderProvider{
				Key:   "test-key-2",
				Value: "test-value-2",
				Error: nil,
			},
		)

		req, _ := http.NewRequest("GET", ts.URL, nil)
		resp, err := client.DoWithContext(context.Background(), req)

		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("got status code %d, want %d", http.StatusOK, resp.StatusCode)
		}
	})

	t.Run("should return error when server is down", func(t *testing.T) {
		client := NewBasicHttpClient()
		req, _ := http.NewRequest("GET", "http://localhost:12345", nil)

		_, err := client.DoWithContext(context.Background(), req)

		if err == nil {
			t.Error("got nil, want error")
		}
	})

	t.Run("should return error when context is canceled", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(1 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		client := NewBasicHttpClient()
		req, _ := http.NewRequest("GET", ts.URL, nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.DoWithContext(ctx, req)

		if err == nil {
			t.Error("got nil, want error")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("got %v, want context.Canceled", err)
		}
	})
}
