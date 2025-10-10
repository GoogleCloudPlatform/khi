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
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/popup"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// oauthTokenSource is an implementation of oauth2.TokenSource that requests tokens from the OAuthServer.
type oauthTokenSource struct {
	server *OAuthServer
}

// Token retrieves an OAuth2 token from the associated OAuthServer.
// Token implements oauth2.TokenSource.
func (o *oauthTokenSource) Token() (*oauth2.Token, error) {
	return o.server.requestToken()
}

var _ oauth2.TokenSource = (*oauthTokenSource)(nil)

type oauthTokenPopup struct {
	oauthCodeURL  string
	popupClosable bool
}

// GetMetadata implements popup.PopupForm.
func (o *oauthTokenPopup) GetMetadata() popup.PopupFormMetadata {
	return popup.PopupFormMetadata{
		Title:       "OAuth Token",
		Type:        "popup_redirect",
		Description: "Please login to your Google account to get the access token.",
		Options: map[string]string{
			popup.PopupOptionRedirectTargetKey: o.oauthCodeURL,
		},
	}
}

// Validate implements popup.PopupForm.
func (o *oauthTokenPopup) Validate(req *popup.PopupAnswerResponse) string {
	if o.popupClosable {
		return ""
	} else {
		return "Authentication is not finished yet. Please check another tab."
	}
}

func newoauthTokenPopup(redirectTarget string) *oauthTokenPopup {
	return &oauthTokenPopup{
		oauthCodeURL:  redirectTarget,
		popupClosable: false,
	}
}

// tokenExchanger defines an interface for exchanging an authorization code for an OAuth2 token.
// This interface is introduced for testing the exchange call.
type tokenExchanger interface {
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
}

type defaultTokenExchanger struct {
	oauthConfig *oauth2.Config
}

// Exchange implements tokenExchanger.
func (d *defaultTokenExchanger) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return d.oauthConfig.Exchange(ctx, code)
}

var _ tokenExchanger = (*defaultTokenExchanger)(nil)

// OAuthServer provides server logic to use OAuth token of the user.
// !!VERY IMPORTANT; PLEASE READ!!: KHI is not expected to be shared by multiple users because it's designed as just a log visualizer used in local environment of each engineers.
//
// After authenticating an user with OAuth, then the other accessing the same KHI process gain the same access of the first user authenticated.
type OAuthServer struct {
	// engine is the Gin engine used to handle HTTP requests.
	engine      *gin.Engine
	oauthConfig *oauth2.Config

	// oauthRedirectTargetServingPath is the path from the given gin root path receiving token with redirect.
	oauthRedirectTargetServingPath string
	oauthStateCodeSuffix           string
	oauthRedirectTimeout           time.Duration
	oauthStateCodesMutex           sync.Mutex
	oauthStateCodes                map[string]struct{}
	resolvedToken                  chan *oauth2.Token
	tokenResolutionError           chan error
	tokenExchanger                 tokenExchanger

	tokenSource oauth2.TokenSource
}

// NewOAuthServer creates and initializes a new OAuthServer.
func NewOAuthServer(engine *gin.Engine, oauthConfig *oauth2.Config, oauthRedirectTargetServingPath string, oauthStateCodeSuffix string) *OAuthServer {
	server := &OAuthServer{
		engine:                         engine,
		oauthConfig:                    oauthConfig,
		oauthRedirectTargetServingPath: oauthRedirectTargetServingPath,
		oauthRedirectTimeout:           5 * time.Minute,
		oauthStateCodeSuffix:           oauthStateCodeSuffix,
		oauthStateCodes:                map[string]struct{}{},
		resolvedToken:                  make(chan *oauth2.Token),
		tokenResolutionError:           make(chan error),
		tokenExchanger:                 &defaultTokenExchanger{oauthConfig: oauthConfig},
	}
	server.configureServer()
	server.tokenSource = oauth2.ReuseTokenSource(nil, &oauthTokenSource{
		server: server,
	})
	return server
}

// configureServer configures the Gin engine to handle OAuth redirect callbacks. It registers a GET handler for the specified
// oauthhRedirectTargetServingPath.
func (s *OAuthServer) configureServer() {
	s.engine.GET(s.oauthRedirectTargetServingPath, func(ctx *gin.Context) {
		if handleErrorRedirect(ctx) {
			s.tokenResolutionError <- fmt.Errorf("authentication failed with redirect error")
			return
		}

		state := ctx.Query("state")
		s.oauthStateCodesMutex.Lock()
		_, found := s.oauthStateCodes[state]
		if found {
			delete(s.oauthStateCodes, state)
		}
		s.oauthStateCodesMutex.Unlock()
		if !found {
			ctx.String(http.StatusBadRequest, "invalid state code")
			s.tokenResolutionError <- fmt.Errorf("invalid state code received: %s", state)
			return
		}

		code := ctx.Query("code") // The authorization code received from the OAuth provider.
		token, err := s.tokenExchanger.Exchange(ctx, code)
		if err != nil {
			ctx.String(http.StatusInternalServerError, "Failed to exchange token: "+err.Error())
			s.tokenResolutionError <- err
			return
		}

		s.resolvedToken <- token
		statusOkWithCloseHTML(ctx)
	})
}

// requestToken initiates an OAuth authentication flow and waits for a token to be resolved. It generates a state code, stores it, and then waits for either a resolved token or an error.
// This method is called by the internal oauthTokenSource when a new token is needed.
func (s *OAuthServer) requestToken() (*oauth2.Token, error) {
	state, err := s.generateStateCode()
	if err != nil {
		return nil, err
	}

	s.oauthStateCodesMutex.Lock()
	s.oauthStateCodes[state] = struct{}{}
	s.oauthStateCodesMutex.Unlock()

	redirectPopup := newoauthTokenPopup(s.oauthConfig.AuthCodeURL(state))
	go func() {
		_, err := popup.Instance.ShowPopup(redirectPopup) // This method blocks the current goroutine. Needs to be called in another goroutine.
		if err != nil {
			s.tokenResolutionError <- err
		}
	}()
	defer func() {
		redirectPopup.popupClosable = true
	}()

	select {
	case token := <-s.resolvedToken:
		return token, nil
	case err := <-s.tokenResolutionError:
		return nil, fmt.Errorf("authentication error: %w", err)
	case <-time.After(s.oauthRedirectTimeout):
		return nil, fmt.Errorf("timed out waiting for authentication")
	}
}

// TokenSource returns an oauth2.TokenSource that can be used to retrieve OAuth2 tokens from this server. This is the primary method for external components to obtain tokens managed by this OAuthServer.
func (s *OAuthServer) TokenSource() oauth2.TokenSource {
	return s.tokenSource
}

// generateStateCode generates a random state code for OAuth authentication. This state code is used to prevent CSRF attacks.
func (s *OAuthServer) generateStateCode() (string, error) {
	randomSeed := make([]byte, 32)
	_, err := rand.Read(randomSeed)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x%s", randomSeed, s.oauthStateCodeSuffix), nil
}

// handleErrorRedirect checks for OAuth error parameters in the redirect URL. If an error is found, it sends an HTTP 400 Bad Request response and returns true.
// This function is used by the OAuth callback handler to gracefully handle authentication failures reported by the OAuth provider.
func handleErrorRedirect(ctx *gin.Context) bool {
	errType := ctx.DefaultQuery("error", "ok")
	if errType == "ok" {
		return false
	}
	errDescription := ctx.DefaultQuery("error_description", "")
	if errDescription != "" {
		errDescription = "Description: " + errDescription
	}
	errorUri := ctx.DefaultQuery("error_uri", "")
	if errorUri != "" {
		errorUri = "URI: " + errorUri
	}
	ctx.String(http.StatusBadRequest, fmt.Sprintf("The authorization server redirected with an error: %s\n%s\n%s", errType, errDescription, errorUri))
	return true
}

// statusOkWithCloseHTML sends an HTML response to the client that closes the current browser window/tab. This is typically used after a successful OAuth authentication to close the popup window.
// It provides a user-friendly message indicating successful authentication.
func statusOkWithCloseHTML(ctx *gin.Context) {
	ctx.Writer.Write([]byte(`<html>
	<head>
		<title>Authentication successful</title>
		<script>window.close();</script>
	</head>
	<body>Authentication successful. You can close this tab.</body>
</html>`))
	ctx.Status(http.StatusOK)
}
