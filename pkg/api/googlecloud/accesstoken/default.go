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

package accesstoken

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

var MinWaitTimeOnRetriableError = 5
var MaxWaitTimeOnRetriableError = 60
var MaxRetryCount = 3
var RetriableHttpResponseCodes = []int{
	429, 500, 501, 502, 503,
}

var DefaultOAuthTokenResolver = NewOAuthTokenResolver()

var DefaultAccessTokenStore = token.NewBasicTokenStore(
	"accesstoken", token.NewMultiTokenResolver(
		DefaultOAuthTokenResolver,
		token.NewOnceTokenResolver(func() string {
			if parameters.Auth.AccessToken == nil {
				return ""
			}
			return *parameters.Auth.AccessToken
		}),
		NewMetadataServerAccessTokenResolver(httpclient.NewJsonResponseHttpClient[MDSResponse](httpclient.NewRetryHttpClient(httpclient.NewBasicHttpClient(), MinWaitTimeOnRetriableError, MaxWaitTimeOnRetriableError, MaxRetryCount, RetriableHttpResponseCodes, []int{}, &token.NopTokenRefresher{}))),
		&GCloudCommandAccessTokenResolver{},
	),
)
