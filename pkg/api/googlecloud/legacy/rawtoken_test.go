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

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
)

func TestRawTokenTokenSource_Token(t *testing.T) {
	accessToken := "test-access-token"
	tokenSource := NewRawTokenTokenSource(accessToken)

	token, err := tokenSource.Token()

	if err != nil {
		t.Fatalf("Token() error = %v, wantErr nil", err)
	}
	want := &oauth2.Token{
		AccessToken: accessToken,
	}
	if diff := cmp.Diff(want, token, cmp.AllowUnexported(oauth2.Token{})); diff != "" {
		t.Errorf("Token() mismatch (-want +got):\n%s", diff)
	}
}
