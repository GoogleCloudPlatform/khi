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
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/popup"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// OAuthTokenPopup is an implementation of popup.Form. The validation of this form logic will return non-error only when the popupClosable is true.
type OAuthTokenPopup struct {
	oauthCodeURL  string
	popupClosable bool
}

// GetMetadata implements popup.PopupForm.
func (o *OAuthTokenPopup) GetMetadata() popup.PopupFormMetadata {
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
func (o *OAuthTokenPopup) Validate(req *popup.PopupAnswerResponse) string {
	if o.popupClosable {
		return ""
	} else {
		return "Authentication is not finished yet. Please check another tab."
	}
}

func newOauthTokenPopup(redirectTarget string) *OAuthTokenPopup {
	return &OAuthTokenPopup{
		oauthCodeURL:  redirectTarget,
		popupClosable: false,
	}
}

var _ popup.PopupForm = (*OAuthTokenPopup)(nil)

type OAuthTokenResolver struct {
	server        *gin.Engine
	popup         *OAuthTokenPopup
	stateCodes    map[string]struct{}
	resolvedToken *oauth2.Token
}

func NewOAuthTokenResolver() *OAuthTokenResolver {
	return &OAuthTokenResolver{
		stateCodes: map[string]struct{}{},
	}
}

// SetServer sets a gin.Engine instance to OAuthTokenResolver. This registers OAuth redirect handler on the given server.
func (o *OAuthTokenResolver) SetServer(server *gin.Engine) error {
	o.server = server
	if !parameters.Auth.OAuthEnabled() {
		return fmt.Errorf("OAuth is not enabled")
	}
	oauthConfig := parameters.Auth.GetOAuthConfig()
	o.server.GET(*parameters.Auth.OAuthRedirectTargetServingPath, func(ctx *gin.Context) {
		errType := ctx.DefaultQuery("error", "ok")
		if errType != "ok" {
			errDescription := ctx.DefaultQuery("error_description", "")
			if errDescription != "" {
				errDescription = "Description: " + errDescription
			}
			errorUri := ctx.DefaultQuery("error_uri", "")
			if errorUri != "" {
				errorUri = "URI: " + errorUri
			}
			ctx.String(http.StatusBadRequest, fmt.Sprintf("The authorization server redirected with an error: %s\n%s\n%s", errType, errDescription, errorUri))
			return
		}
		state := ctx.Query("state")
		if _, found := o.stateCodes[state]; !found {
			ctx.String(http.StatusBadRequest, "invalid state code")
			return
		}
		code := ctx.Query("code")
		token, err := oauthConfig.Exchange(ctx, code)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		if o.popup == nil {
			ctx.String(http.StatusInternalServerError, "popup is not initialized")
			return
		}
		delete(o.stateCodes, state)
		o.resolvedToken = token
		o.popup.popupClosable = true
		// Return the HTML closing the window itself.
		ctx.Writer.Write([]byte(`<html>
	<head>
		<title>Authentication successful</title>
		<script>window.close();</script>
	</head>
	<body>Authentication successful. You can close this tab.</body>
</html>`))
		ctx.Status(http.StatusOK)
	})
	return nil
}

// Resolve implements token.TokenResolver.
func (o *OAuthTokenResolver) Resolve(ctx context.Context) (*token.Token, error) {
	if !parameters.Auth.OAuthEnabled() {
		return nil, fmt.Errorf("OAuth is not enabled")
	}
	oauthConfig := parameters.Auth.GetOAuthConfig()
	stateCode, err := o.generateStateCode()
	if err != nil {
		return nil, err
	}
	o.stateCodes[stateCode] = struct{}{}
	o.popup = newOauthTokenPopup(oauthConfig.AuthCodeURL(stateCode))
	_, err = popup.Instance.ShowPopup(o.popup)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "obtained access token with OAuth")
	return token.NewWithExpiry(o.resolvedToken.AccessToken, o.resolvedToken.Expiry), nil
}

func (o *OAuthTokenResolver) generateStateCode() (string, error) {
	stateSuffix := ""
	if parameters.Auth.OAuthStateSuffix != nil {
		stateSuffix = *parameters.Auth.OAuthStateSuffix
	}
	randomSeed := make([]byte, 32)
	_, err := rand.Read(randomSeed)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x%s", randomSeed, stateSuffix), nil
}

var _ token.TokenResolver = (*OAuthTokenResolver)(nil)
