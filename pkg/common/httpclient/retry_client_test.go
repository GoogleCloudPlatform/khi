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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/token"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type mockFailClient struct {
	Responses    []*http.Response
	Requests     []*http.Request
	RequestCount int
}

type tokenRefresherClientSpy struct {
	CallCount int
}

// Refresh implements TokenRefresher.
func (t *tokenRefresherClientSpy) Refresh(ctx context.Context) error {
	t.CallCount++
	return nil
}

var _ token.TokenRefresher = (*tokenRefresherClientSpy)(nil)

// DoWithContext implements HttpClient.
func (m *mockFailClient) DoWithContext(ctx context.Context, request *http.Request) (*http.Response, error) {
	m.RequestCount += 1
	m.Requests = append(m.Requests, request)
	return m.Responses[m.RequestCount-1], nil
}

var _ HTTPClient[*http.Response] = (*mockFailClient)(nil)

func TestIsRetriable(t *testing.T) {
	type testCase struct {
		RetriableHttpCodes []int
		HttpCode           int
		Expected           bool
	}
	testCases := []testCase{
		{
			RetriableHttpCodes: []int{400, 401, 402},
			HttpCode:           400,
			Expected:           true,
		},
		{
			RetriableHttpCodes: []int{401, 402},
			HttpCode:           400,
			Expected:           false,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("with codes:%v", tc.RetriableHttpCodes), func(t *testing.T) {
			client := &RetryHttpClient{RetriableHttpCodes: tc.RetriableHttpCodes}
			actual := client.isRetriable(tc.HttpCode)
			if actual != tc.Expected {
				t.Errorf("got retriability %t, want %t", actual, tc.Expected)
			}
		})
	}
}

func TestRetryBehavior(t *testing.T) {
	type testCase struct {
		Title                       string
		ResponseCodes               []int
		RequestBody                 string
		RequestHeaders              map[string]string
		ExpectedRequestCount        int
		ExpectedError               string
		MinWaitTime                 int
		MaxWaitTime                 int
		MaxRetryCount               int
		ExpectedLastCurrentWaitTime int
		ExpectedTokenRefresherCall  int
		ExpectedHeaders             map[string]string
	}
	testCases := []testCase{
		{
			Title:         "Simple success",
			ResponseCodes: []int{200},
			RequestBody:   "foo",
			RequestHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
			ExpectedRequestCount:        1,
			ExpectedError:               "",
			MaxRetryCount:               3,
			MinWaitTime:                 1,
			MaxWaitTime:                 4,
			ExpectedLastCurrentWaitTime: 1,
			ExpectedTokenRefresherCall:  0,
			ExpectedHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
		},
		{
			Title:         "Non retriable",
			ResponseCodes: []int{500},
			RequestBody:   "foo",
			RequestHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
			ExpectedRequestCount:        1,
			ExpectedError:               "unretriable error returned(500):\nBODY:",
			MaxRetryCount:               3,
			MinWaitTime:                 1,
			MaxWaitTime:                 4,
			ExpectedLastCurrentWaitTime: 1,
			ExpectedTokenRefresherCall:  0,
			ExpectedHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
		},
		{
			Title:         "Multiple retries",
			ResponseCodes: []int{400, 400, 200},
			RequestBody:   "foo",
			RequestHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
			ExpectedRequestCount:        3,
			ExpectedError:               "",
			MaxRetryCount:               3,
			MinWaitTime:                 1,
			MaxWaitTime:                 4,
			ExpectedLastCurrentWaitTime: 1,
			ExpectedTokenRefresherCall:  0,
			ExpectedHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
		},
		{
			Title:         "Multiple retries and exceed maximum",
			ResponseCodes: []int{400, 400, 400},
			RequestBody:   "foo",
			RequestHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
			ExpectedRequestCount:        3,
			ExpectedError:               "maximum retry count exceeded 3\nStatus codes:[400 400 400]",
			MaxRetryCount:               3,
			MinWaitTime:                 1,
			MaxWaitTime:                 3,
			ExpectedLastCurrentWaitTime: 3,
			ExpectedTokenRefresherCall:  0,
			ExpectedHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
		},
		{
			Title:         "Wait time should be increased as exponential",
			ResponseCodes: []int{400, 400},
			RequestBody:   "foo",
			RequestHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
			ExpectedRequestCount:        2,
			ExpectedError:               "maximum retry count exceeded 2\nStatus codes:[400 400]",
			MaxRetryCount:               2,
			MinWaitTime:                 1,
			MaxWaitTime:                 10,
			ExpectedLastCurrentWaitTime: 4,
			ExpectedTokenRefresherCall:  0,
			ExpectedHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
		},
		{
			Title:         "Refresh token when response code require refreshing token",
			ResponseCodes: []int{401, 200},
			RequestBody:   "foo",
			RequestHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
			ExpectedRequestCount:        2,
			ExpectedError:               "",
			MaxRetryCount:               2,
			MinWaitTime:                 1,
			MaxWaitTime:                 10,
			ExpectedLastCurrentWaitTime: 1,
			ExpectedTokenRefresherCall:  1,
			ExpectedHeaders: map[string]string{
				"Test-Header1": "test-value1",
				"Test-Header2": "test-value2",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Title, func(t *testing.T) {
			responses := []*http.Response{}
			for _, respCode := range tc.ResponseCodes {
				responses = append(responses, &http.Response{
					StatusCode: respCode,
				})
			}
			baseClient := mockFailClient{
				Responses: responses,
				Requests:  make([]*http.Request, 0),
			}
			refresherSpy := tokenRefresherClientSpy{}
			retryClient := NewRetryHttpClient(&baseClient, tc.MinWaitTime, tc.MaxWaitTime, tc.MaxRetryCount, []int{400}, []int{401}, &refresherSpy)
			retryClient.timeUnit = time.Millisecond
			req, err := http.NewRequest("GET", "https://google.com", bytes.NewBuffer([]byte(tc.RequestBody)))
			if err != nil {
				t.Errorf("got error %v, want nil", err)
			}
			for headerKey, headerValue := range tc.RequestHeaders {
				req.Header.Add(headerKey, headerValue)
			}
			response, err := retryClient.DoWithContext(context.Background(), req)
			if tc.ExpectedError == "" {
				if response == nil {
					t.Error("got nil, want response")
				}
				if err != nil {
					t.Errorf("got error %v, want nil", err)
				}
				if baseClient.RequestCount != tc.ExpectedRequestCount {
					t.Errorf("got retry count %d, want %d", baseClient.RequestCount, tc.ExpectedRequestCount)
				}
			} else {
				if err.Error() != tc.ExpectedError {
					t.Errorf("got error %q, want %q", err.Error(), tc.ExpectedError)
				}
				if baseClient.RequestCount != tc.ExpectedRequestCount {
					t.Errorf("got retry count %d, want %d", baseClient.RequestCount, tc.ExpectedRequestCount)
				}
			}
			for _, req := range baseClient.Requests {
				requestBody, err := io.ReadAll(req.Body)
				if err != nil {
					t.Errorf("got error %v, wnt nil", err)
				}
				requestBodyStr := string(requestBody)
				if requestBodyStr != tc.RequestBody {
					t.Errorf("got requestBody %q, want %q", requestBody, tc.RequestBody)
				}
				for key, wantHeader := range tc.ExpectedHeaders {
					gotHeader := req.Header.Get(key)
					if wantHeader != gotHeader {
						t.Errorf("got unexpected header %q:%q, want %q:%q", key, gotHeader, key, wantHeader)
					}
				}
			}
			if tc.ExpectedLastCurrentWaitTime != retryClient.currentWaitSeconds {
				t.Errorf("got wait time %d, want %d", retryClient.currentWaitSeconds, tc.ExpectedLastCurrentWaitTime)
			}
			if tc.ExpectedTokenRefresherCall != refresherSpy.CallCount {
				t.Errorf("got token refresher call count %d, want %d", refresherSpy.CallCount, tc.ExpectedTokenRefresherCall)
			}
		})
	}
}
