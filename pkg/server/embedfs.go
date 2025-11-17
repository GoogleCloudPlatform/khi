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

package server

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-contrib/static"
)

// embedFileSystem is a gin middleware to serve static files from embedded file system.
// The original implementation in gin-contrib/static somehow didn't consider the prefix thus this implementation just ported them here with replacing the prefix handling.
type embedFileSystem struct {
	http.FileSystem
}

func (e embedFileSystem) Exists(prefix string, path string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	_, err := e.Open(strings.TrimPrefix(path, prefix))
	return err == nil
}

func embedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	fsys, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		slog.Error("Failed to embed folder",
			"targetPath", targetPath,
			"error", err,
		)
		return nil
	}
	return embedFileSystem{
		FileSystem: http.FS(fsys),
	}
}
