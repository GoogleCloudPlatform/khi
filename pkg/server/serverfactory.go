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
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/GoogleCloudPlatform/khi/pkg/server/option"
)

// DefaultServerFactory is the default instance of ServerFactory.
// This instance Options will be modified to extend the behavior of the gin server.
var DefaultServerFactory *ServerFactory = &ServerFactory{}

// ServerFactory is responsible for creating and configuring Gin engine instances.
type ServerFactory struct {
	Options []option.Option
	mu      sync.Mutex
}

// AddOptions adds one or more Option instances to the factory's configuration.
func (s *ServerFactory) AddOptions(opt ...option.Option) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Options = append(s.Options, opt...)
}

// CreateInstance creates a new Gin engine and applies all registered options to it.
// It returns the configured Gin engine or an error if any option fails to apply.
func (s *ServerFactory) CreateInstance(mode string) (*gin.Engine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	gin.SetMode(mode)
	engine := gin.New()
	err := option.ApplyOptions(engine, s.Options)
	if err != nil {
		return nil, err
	}
	return engine, nil
}
