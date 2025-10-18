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

package oauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/popup"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2"
)

const (
	testRedirectPath = "/oauth/callback"
	testStateSuffix  = "-test-suffix"
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
	testAuthURL      = "http://localhost/auth"
	testTokenURL     = "http://localhost/token"
)

// mockTokenExchanger is a mock implementation of the tokenExchanger interface for testing.
type mockTokenExchanger struct {
	token *oauth2.Token
	err   error
}

// Exchange implements the tokenExchanger interface.
func (m *mockTokenExchanger) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	if code == "valid-code" {
		return m.token, nil
	}
	return nil, errors.New("invalid code")
}

// newTestOAuthServer creates a new OAuthServer with a mock token exchanger for testing.
func newTestOAuthServer(t *testing.T, exchanger tokenExchanger) (*OAuthServer, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	// Reset the popup status
	popup.Instance = popup.NewPopupManager()
	engine := gin.New()

	conf := &oauth2.Config{
		RedirectURL:  "http://localhost" + testRedirectPath,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
		Scopes:       []string{"test-scope"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  testAuthURL,
			TokenURL: testTokenURL,
		},
	}

	server := NewOAuthServer(engine, conf, testRedirectPath, testStateSuffix)
	// Inject the mock exchanger for test purposes.
	if exchanger != nil {
		server.tokenExchanger = exchanger
	}

	return server, engine
}

func TestNewOAuthServer(t *testing.T) {
	server, _ := newTestOAuthServer(t, nil)
	if server == nil {
		t.Fatal("NewOAuthServer() returned nil")
	}
	if server.oauthRedirectTargetServingPath != testRedirectPath {
		t.Errorf("got %q, want %q", server.oauthRedirectTargetServingPath, testRedirectPath)
	}
	if server.oauthStateCodeSuffix != testStateSuffix {
		t.Errorf("got %q, want %q", server.oauthStateCodeSuffix, testStateSuffix)
	}
	if server.engine == nil {
		t.Error("server.engine is nil")
	}
	if server.oauthConfig == nil {
		t.Error("server.oauthConfig is nil")
	}
	if server.tokenSource == nil {
		t.Error("server.tokenSource is nil")
	}

	if _, ok := server.tokenExchanger.(*defaultTokenExchanger); !ok {
		t.Error("default token exchanger should be of type defualtTokenExchanger")
	}
}

func TestOAuthCallbackHandler_Success(t *testing.T) {
	wantToken := &oauth2.Token{AccessToken: "test-token", TokenType: "Bearer"}
	mockExchanger := &mockTokenExchanger{token: wantToken}
	server, engine := newTestOAuthServer(t, mockExchanger)

	state, err := server.generateStateCode()
	if err != nil {
		t.Fatalf("generateStateCode() failed: %v", err)
	}
	server.oauthStateCodes[state] = struct{}{}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case token := <-server.resolvedToken:
			if diff := cmp.Diff(wantToken.AccessToken, token.AccessToken); diff != "" {
				t.Errorf("resolved token mismatch (-want +got):\n%s", diff)
			}
		case err := <-server.tokenResolutionError:
			t.Errorf("Expected token, but got error: %v", err)
		case <-time.After(1 * time.Second):
			t.Error("Timed out waiting for token")
		}
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s?code=valid-code&state=%s", testRedirectPath, state), nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "Authentication successful") {
		t.Errorf("response body does not contain 'Authentication successful'")
	}

	wg.Wait()
}

func TestOAuthCallbackHandler_InvalidState(t *testing.T) {
	server, engine := newTestOAuthServer(t, nil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case err := <-server.tokenResolutionError:
			if err == nil {
				t.Error("expected an error but got nil")
			}
			if !strings.Contains(err.Error(), "invalid state code received") {
				t.Errorf("error message does not contain 'invalid state code received'")
			}
		case <-server.resolvedToken:
			t.Error("Expected error, but got token")
		case <-time.After(1 * time.Second):
			t.Error("Timed out waiting for error")
		}
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s?code=any-code&state=invalid-state", testRedirectPath), nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "invalid state code" {
		t.Errorf("got body %q, want %q", w.Body.String(), "invalid state code")
	}

	wg.Wait()
}

func TestOAuthCallbackHandler_ExchangeError(t *testing.T) {
	exchangeErr := errors.New("exchange failed")
	mockExchanger := &mockTokenExchanger{err: exchangeErr}
	server, engine := newTestOAuthServer(t, mockExchanger)

	state, err := server.generateStateCode()
	if err != nil {
		t.Fatalf("generateStateCode() failed: %v", err)
	}
	server.oauthStateCodes[state] = struct{}{}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case err := <-server.tokenResolutionError:
			if !errors.Is(err, exchangeErr) {
				t.Errorf("expected error %v, got %v", exchangeErr, err)
			}
		case <-server.resolvedToken:
			t.Error("Expected error, but got token")
		case <-time.After(1 * time.Second):
			t.Error("Timed out waiting for error")
		}
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s?code=valid-code&state=%s", testRedirectPath, state), nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(w.Body.String(), "Failed to exchange token: "+exchangeErr.Error()) {
		t.Errorf("response body does not contain the correct error message")
	}

	wg.Wait()
}

func TestRequestToken_Timeout(t *testing.T) {
	server, _ := newTestOAuthServer(t, nil)
	server.oauthRedirectTimeout = 100 * time.Millisecond

	// request the token but user didn't visit the redirected page for long time.
	token, err := server.requestToken()
	if err == nil {
		t.Fatal("expected a timeout error but got nil")
	}
	if token != nil {
		t.Errorf("expected token to be nil, got %v", token)
	}
	if !strings.Contains(err.Error(), "timed out waiting for authentication") {
		t.Errorf("expected timeout error message, got %v", err)
	}
}

func TestOAuthCallbackHandler_RedirectError(t *testing.T) {
	server, engine := newTestOAuthServer(t, nil)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case err := <-server.tokenResolutionError:
			if err == nil {
				t.Error("expected an error but got nil")
			}
			if !strings.Contains(err.Error(), "authentication failed with redirect error") {
				t.Errorf("error message does not contain 'authentication failed with redirect error'")
			}
		case <-server.resolvedToken:
			t.Error("Expected error, but got token")
		case <-time.After(1 * time.Second):
			t.Error("Timed out waiting for error")
		}
	}()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s?error=access_denied&error_description=user+denied", testRedirectPath), nil)
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "access_denied") {
		t.Errorf("response body does not contain 'access_denied'")
	}
	if !strings.Contains(w.Body.String(), "user denied") {
		t.Errorf("response body does not contain 'user denied'")
	}

	wg.Wait()
}

func TestRequestToken_Success(t *testing.T) {
	server, _ := newTestOAuthServer(t, nil)
	wantToken := &oauth2.Token{AccessToken: "success"}
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		time.Sleep(10 * time.Millisecond)
		currentPopup := popup.Instance.GetCurrentPopup()
		if currentPopup == nil {
			t.Error("popup was not shown")
		}

		if len(server.oauthStateCodes) != 1 {
			t.Errorf("expected 1 state code, got %d", len(server.oauthStateCodes))
		}

		var currentState string
		for state := range server.oauthStateCodes {
			currentState = state
		}

		wantPopupFormRequest := &popup.PopupFormRequest{
			Title:       "OAuth Token",
			Type:        "popup_redirect",
			Description: "Please login to your Google account to get the access token.",
			Options: map[string]string{
				"redirectTo": fmt.Sprintf("http://localhost/auth?client_id=test-client-id&redirect_uri=http%%3A%%2F%%2Flocalhost%%2Foauth%%2Fcallback&response_type=code&scope=test-scope&state=%s", currentState),
			},
		}
		if diff := cmp.Diff(wantPopupFormRequest, currentPopup, cmpopts.IgnoreFields(popup.PopupFormRequest{}, "Id")); diff != "" {
			t.Errorf("popup metadata mismatch (-want +got):\n%s", diff)
		}

		server.resolvedToken <- wantToken
		wg.Done()
	}()

	token, err := server.requestToken()
	if err != nil {
		t.Fatalf("requestToken() failed: %v", err)
	}
	if diff := cmp.Diff(wantToken.AccessToken, token.AccessToken); diff != "" {
		t.Errorf("token mismatch (-want +got):\n%s", diff)
	}
	wg.Wait()
}

func TestRequestToken_Error(t *testing.T) {
	server, _ := newTestOAuthServer(t, nil)
	wantErr := errors.New("auth error")

	go func() {
		time.Sleep(10 * time.Millisecond)
		server.tokenResolutionError <- wantErr
	}()

	token, err := server.requestToken()
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
	if token != nil {
		t.Errorf("expected token to be nil, got %v", token)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected error %v to wrap %v", err, wantErr)
	}
}

func TestGenerateStateCode(t *testing.T) {
	server, _ := newTestOAuthServer(t, nil)
	state, err := server.generateStateCode()
	if err != nil {
		t.Fatalf("generateStateCode() failed: %v", err)
	}
	if state == "" {
		t.Error("generated state is empty")
	}
	if !strings.HasSuffix(state, testStateSuffix) {
		t.Errorf("generated state does not have the correct suffix")
	}

	state2, err := server.generateStateCode()
	if err != nil {
		t.Fatalf("generateStateCode() failed on second call: %v", err)
	}
	if state == state2 {
		t.Error("generated states are not random")
	}
}

func TestHandleErrorRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var capturedErr error
	r.GET("/test", func(c *gin.Context) {
		if handleErrorRedirect(c) {
			capturedErr = errors.New("redirect error handled")
		}
	})

	t.Run("No error", func(t *testing.T) {
		capturedErr = nil
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
		}
		if capturedErr != nil {
			t.Errorf("capturedErr should be nil, got %v", capturedErr)
		}
	})

	t.Run("With error", func(t *testing.T) {
		capturedErr = nil
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test?error=some_error&error_description=details&error_uri=http://example.com", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
		if capturedErr == nil {
			t.Fatal("capturedErr should not be nil")
		}
		body := w.Body.String()
		if !strings.Contains(body, "some_error") {
			t.Errorf("body does not contain 'some_error'")
		}
		if !strings.Contains(body, "Description: details") {
			t.Errorf("body does not contain 'Description: details'")
		}
		if !strings.Contains(body, "URI: http://example.com") {
			t.Errorf("body does not contain 'URI: http://example.com'")
		}
	})
}

func TestStatusOkWithCloseHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", statusOkWithCloseHTML)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Errorf("body does not contain '<html>'")
	}
	if !strings.Contains(body, "window.close()") {
		t.Errorf("body does not contain 'window.close()'")
	}
	if !strings.Contains(body, "Authentication successful") {
		t.Errorf("body does not contain 'Authentication successful'")
	}
}
