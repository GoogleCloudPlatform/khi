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
	"log/slog"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
)

type RetryHttpClient struct {
	Client                             HTTPClient[*http.Response]
	MinWaitSeconds                     int
	MaxWaitSeconds                     int
	MaxRetryCount                      int
	RetriableHttpCodes                 []int
	RetriableWithRefreshTokenHttpCodes []int
	currentWaitSeconds                 int
	timeUnit                           time.Duration // For testing purpose to make test faster
	tokenRefresher                     token.TokenRefresher
}

func NewRetryHttpClient(baseClient HTTPClient[*http.Response], minWaitSeconds int, maxWaitSeconds int, maxRetryCount int, retriableHttpCodes []int, retriableWithRefreshTokenHttpCodes []int, tokenRefresher token.TokenRefresher) *RetryHttpClient {
	return &RetryHttpClient{
		Client:                             baseClient,
		MinWaitSeconds:                     minWaitSeconds,
		MaxWaitSeconds:                     maxWaitSeconds,
		MaxRetryCount:                      maxRetryCount,
		RetriableHttpCodes:                 retriableHttpCodes,
		RetriableWithRefreshTokenHttpCodes: retriableWithRefreshTokenHttpCodes,
		currentWaitSeconds:                 minWaitSeconds,
		timeUnit:                           time.Second,
		tokenRefresher:                     tokenRefresher,
	}
}

// DoWithContext implements HttpClient.
func (r *RetryHttpClient) DoWithContext(ctx context.Context, originalRequest *http.Request) (*http.Response, error) {
	// Clone request body into array to create another reader of Body on retry.
	var clonedRequest []byte
	if originalRequest.Body != nil {
		var err error
		clonedRequest, err = io.ReadAll(originalRequest.Body)
		if err != nil {
			return nil, err
		}
	}
	statusCodes := []int{}
	for i := 0; i < r.MaxRetryCount; i++ {
		request, err := http.NewRequestWithContext(ctx, originalRequest.Method, originalRequest.URL.String(), bytes.NewBuffer(clonedRequest))
		if err != nil {
			return nil, err
		}
		request.Header = originalRequest.Header.Clone()
		response, err := r.Client.DoWithContext(ctx, request)
		if err != nil {
			return nil, err
		}
		if response.StatusCode < 400 {
			r.currentWaitSeconds = r.MinWaitSeconds
			// Treat this response is ok not to retry
			return response, nil
		}
		if !r.isRetriable(response.StatusCode) {
			body := []byte{}
			if response.Body != nil {
				body, _ = io.ReadAll(response.Body)
			}
			return response, fmt.Errorf("unretriable error returned(%d):%s\nBODY:%s", response.StatusCode, response.Status, string(body))
		} else {
			statusCodes = append(statusCodes, response.StatusCode)
			if r.isRetriableWithRefreshingToken(response.StatusCode) {
				slog.DebugContext(ctx, fmt.Sprintf("Previous request to %s got %d response. Attempting retrying with refreshing the token.", request.RequestURI, response.StatusCode))
				r.tokenRefresher.Refresh(ctx)
				r.currentWaitSeconds = r.MinWaitSeconds
			} else {
				r.currentWaitSeconds *= 2
				if r.currentWaitSeconds > r.MaxWaitSeconds {
					r.currentWaitSeconds = r.MaxWaitSeconds
				}
				slog.DebugContext(ctx, fmt.Sprintf("Previous request to %s got %d response. Next retry after %d seconds", request.RequestURI, response.StatusCode, r.currentWaitSeconds))
				time.Sleep(r.timeUnit * time.Duration(r.currentWaitSeconds))
			}
		}
	}
	return nil, fmt.Errorf("maximum retry count exceeded %d\nStatus codes:%v", r.MaxRetryCount, statusCodes)
}

func (r *RetryHttpClient) isRetriable(code int) bool {
	for _, retryCode := range r.RetriableHttpCodes {
		if code == retryCode {
			return true
		}
	}
	return r.isRetriableWithRefreshingToken(code)
}

func (r *RetryHttpClient) isRetriableWithRefreshingToken(code int) bool {
	for _, retryCode := range r.RetriableWithRefreshTokenHttpCodes {
		if code == retryCode {
			return true
		}
	}
	return false
}

var _ (HTTPClient[*http.Response]) = (*RetryHttpClient)(nil)
