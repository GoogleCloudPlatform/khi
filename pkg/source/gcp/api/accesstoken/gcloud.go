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
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
)

type GCloudCommandAccessTokenResolver struct {
}

// Resolve implements token.TokenResolver.
func (g *GCloudCommandAccessTokenResolver) Resolve(ctx context.Context) (*token.Token, error) {
	slog.InfoContext(ctx, `Environment variable "GCP_ACCESS_TOKEN" was not found. Trying to get access token with gcloud command...`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token via gcloud command. \nstderr:\n%s\n\nstdout:\n%s\n\nerr:%s", stderr.String(), stdout.String(), err.Error())
	}
	return token.NewWithExpiry(strings.ReplaceAll(stdout.String(), "\n", ""), time.Now().Add(time.Hour)), nil
}

var _ token.TokenResolver = (*GCloudCommandAccessTokenResolver)(nil)
