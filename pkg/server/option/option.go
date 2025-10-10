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

package option

import (
	"slices"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Option defines an interface for configuring a Gin engine.
type Option interface {
	// ID returns a unique identifier for the option.
	ID() string
	// Order returns the order in which this option should be applied relative to other options.
	Order() int
	// Apply applies the option's configuration to the given Gin engine.
	// It returns an error if the application fails.
	Apply(engine *gin.Engine) error
}

func ApplyOptions(engine *gin.Engine, options []Option) error {
	slices.SortFunc(options, func(a, b Option) int { return a.Order() - b.Order() })
	for _, option := range options {
		err := option.Apply(engine)
		if err != nil {
			return err
		}
	}
	return nil
}

// requiredOption is an Option implementation for setting several required middleware in KHI.
type requiredOption struct {
}

// Required creates a new Option to set several required middlewares and gin server mode.
func Required() Option {
	return &requiredOption{}
}

func (s *requiredOption) ID() string {
	return "required"
}

// Order returns the application order for the server mode option.
func (s *requiredOption) Order() int {
	return 0
}

// Apply adds required middlewares (currently the recovery is the only middleware.)
func (s *requiredOption) Apply(engine *gin.Engine) error {
	engine.Use(gin.Recovery())
	return nil
}

var _ Option = (*requiredOption)(nil)

// corsOption is an Option implementation for enabling CORS.
type corsOption struct {
	corsConfig cors.Config
}

// CORS creates a new Option to enable CORS.
func CORS(config cors.Config) Option {
	return &corsOption{config}
}

func (c *corsOption) ID() string {
	return "cors"
}

// Order returns the application order for the CORS option.
func (c *corsOption) Order() int {
	return 1
}

// Apply configures the Gin engine to use the gin-contrib/cors middleware with all origins allowed.
func (c *corsOption) Apply(engine *gin.Engine) error {
	engine.Use(cors.New(c.corsConfig))
	return nil
}

var _ Option = (*corsOption)(nil)

type accessLogOption struct {
	ignoredPath []string
}

// AccessLog creates a new Option to log access logs with ignoreing the provided paths.
func AccessLog(ignoredPath ...string) Option {
	return &accessLogOption{
		ignoredPath: ignoredPath,
	}
}

// Apply implements Option.
func (l *accessLogOption) Apply(engine *gin.Engine) error {
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: l.ignoredPath,
	}))
	return nil
}

// Order implements Option.
func (l *accessLogOption) Order() int {
	return 2
}

func (l *accessLogOption) ID() string {
	return "access-log"
}

var _ Option = (*accessLogOption)(nil)
