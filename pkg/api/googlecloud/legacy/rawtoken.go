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

package legacy

import "golang.org/x/oauth2"

// rawTokenTokenSource is an oauth2.TokenSource that always returns the same access token.
//
// Using this authentication method is highly discouraged, this is only for keeping compatibility after supporting authentication via ADC.
type rawTokenTokenSource struct {
	AccessToken string
}

// NewRawTokenTokenSource creates a new RawTokenTokenSource with the given access token.
func NewRawTokenTokenSource(accessToken string) oauth2.TokenSource {
	return &rawTokenTokenSource{
		AccessToken: accessToken,
	}
}

// Token returns an oauth2.Token containing the stored AccessToken.
// It implements the oauth2.TokenSource interface.
func (r *rawTokenTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: r.AccessToken,
	}, nil
}

var _ oauth2.TokenSource = (*rawTokenTokenSource)(nil)
