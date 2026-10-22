// Copyright 2026 Google LLC
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

package defaultinit

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
)

var (
	// CDPBrowserKey stores the headless Chrome cdp.Browser instance.
	CDPBrowserKey = typedmap.NewTypedKey[*cdp.Browser]("khi.google.com/init/cdp-browser")
)

// InitializerIDCDPBrowser identifies the Initializer that launches the headless Chrome browser.
const InitializerIDCDPBrowser coreinit.InitializerID = "khi.default/cdp-browser"

// CDPBrowserInitializer starts a headless Chrome browser when not in job mode and not disabled via --mcp-disable-renderer.
var CDPBrowserInitializer = &coreinit.Initializer{
	ID: InitializerIDCDPBrowser,
	Dependencies: []coreinit.InitializerID{
		InitializerIDParameterParse,
	},
	Before: []coreinit.InitializerID{
		InitializerIDMCPServer,
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}
		serverParams := coreinit.MustGet(ctx, ServerParametersKey)
		if serverParams.MCPDisableRenderer != nil && *serverParams.MCPDisableRenderer {
			return nil
		}

		startCtx, cancel := context.WithTimeout(ctx.Context, cdp.DefaultStartTimeout)
		defer cancel()

		browser, err := cdp.StartBrowser(startCtx)
		if err != nil {
			return fmt.Errorf("failed to start headless Chrome: %w", err)
		}
		ctx.OnTerminate(func() error {
			return browser.Close()
		})
		coreinit.Set(ctx, CDPBrowserKey, browser)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(CDPBrowserInitializer)
}
