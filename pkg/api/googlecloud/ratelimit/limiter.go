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

package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"golang.org/x/time/rate"
)

// AdaptiveRateLimiterConfig defines configuration parameters for AdaptiveRateLimiter.
type AdaptiveRateLimiterConfig struct {
	// InitialQPS is the starting request rate in queries per second.
	InitialQPS float64
	// MinQPS is the minimum lower bound for request rate.
	MinQPS float64
	// MaxQPS is the maximum upper bound for request rate.
	MaxQPS float64
	// IncreaseStep is the additive increment added to QPS on consecutive successes.
	IncreaseStep float64
	// DecreaseFactor is the multiplicative multiplier applied to QPS upon encountering a 429 ResourceExhausted error.
	DecreaseFactor float64
	// DecreaseCooldown is the minimum duration between successive rate reductions to prevent cascade decreases.
	DecreaseCooldown time.Duration
	// Burst is the maximum burst size permitted by the underlying token bucket.
	Burst int
}

// DefaultAdaptiveRateLimiterConfig returns standard default configuration values.
func DefaultAdaptiveRateLimiterConfig() AdaptiveRateLimiterConfig {
	return AdaptiveRateLimiterConfig{
		InitialQPS:       1.0,
		MinQPS:           1.0,
		MaxQPS:           100.0,
		IncreaseStep:     0.5,
		DecreaseFactor:   0.5,
		DecreaseCooldown: 1500 * time.Millisecond,
		Burst:            1,
	}
}

// AdaptiveRateLimiter dynamically controls request rates using an Additive Increase / Multiplicative Decrease (AIMD) algorithm.
type AdaptiveRateLimiter struct {
	config           AdaptiveRateLimiterConfig
	limiter          *rate.Limiter
	currentQPS       float64
	lastDecreaseTime time.Time
	mu               sync.Mutex
}

// NewAdaptiveRateLimiter creates a new AdaptiveRateLimiter instance with the specified configuration.
func NewAdaptiveRateLimiter(config AdaptiveRateLimiterConfig) *AdaptiveRateLimiter {
	if config.MinQPS <= 0 {
		config.MinQPS = 1.0
	}
	if config.MaxQPS < config.MinQPS {
		config.MaxQPS = config.MinQPS
	}
	if config.InitialQPS < config.MinQPS {
		config.InitialQPS = config.MinQPS
	}
	if config.InitialQPS > config.MaxQPS {
		config.InitialQPS = config.MaxQPS
	}
	if config.IncreaseStep <= 0 {
		config.IncreaseStep = 0.5
	}
	if config.DecreaseFactor <= 0 || config.DecreaseFactor >= 1.0 {
		config.DecreaseFactor = 0.5
	}
	if config.DecreaseCooldown <= 0 {
		config.DecreaseCooldown = 1500 * time.Millisecond
	}
	if config.Burst <= 0 {
		config.Burst = 1
	}

	return &AdaptiveRateLimiter{
		config:     config,
		limiter:    rate.NewLimiter(rate.Limit(config.InitialQPS), config.Burst),
		currentQPS: config.InitialQPS,
	}
}

// Wait blocks until the underlying rate limiter permits an event or the context is cancelled.
func (a *AdaptiveRateLimiter) Wait(ctx context.Context) error {
	return a.limiter.Wait(ctx)
}

// OnSuccess adjusts the rate upwards using Additive Increase.
func (a *AdaptiveRateLimiter) OnSuccess() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Suppress rate increases during the cooldown window following a rate reduction.
	if !a.lastDecreaseTime.IsZero() && time.Since(a.lastDecreaseTime) < a.config.DecreaseCooldown {
		return
	}

	if a.currentQPS >= a.config.MaxQPS {
		return
	}

	newQPS := math.Min(a.config.MaxQPS, a.currentQPS+a.config.IncreaseStep)
	a.currentQPS = newQPS
	a.limiter.SetLimit(rate.Limit(newQPS))
}

// OnResourceExhausted adjusts the rate downwards using Multiplicative Decrease with cooldown suppression.
func (a *AdaptiveRateLimiter) OnResourceExhausted() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	if !a.lastDecreaseTime.IsZero() && now.Sub(a.lastDecreaseTime) < a.config.DecreaseCooldown {
		// Suppress cascade decreases within the cooldown window.
		return
	}

	a.lastDecreaseTime = now
	newQPS := math.Max(a.config.MinQPS, a.currentQPS*a.config.DecreaseFactor)
	a.currentQPS = newQPS
	a.limiter.SetLimit(rate.Limit(newQPS))
}

// CurrentRate returns the current permitted QPS.
func (a *AdaptiveRateLimiter) CurrentRate() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentQPS
}

// Config returns the configuration of the limiter.
func (a *AdaptiveRateLimiter) Config() AdaptiveRateLimiterConfig {
	return a.config
}

// Pool manages and caches AdaptiveRateLimiter instances per resource container or quota project.
type Pool struct {
	config         AdaptiveRateLimiterConfig
	quotaProjectID string
	limiters       map[string]*AdaptiveRateLimiter
	mu             sync.Mutex
}

// NewPool creates a new Pool instance.
func NewPool(config AdaptiveRateLimiterConfig, quotaProjectID string) *Pool {
	return &Pool{
		config:         config,
		quotaProjectID: quotaProjectID,
		limiters:       make(map[string]*AdaptiveRateLimiter),
	}
}

// GetLimiter returns the AdaptiveRateLimiter corresponding to the given ResourceContainer or quota project.
func (p *Pool) GetLimiter(container googlecloud.ResourceContainer) *AdaptiveRateLimiter {
	key := p.quotaProjectID
	if key == "" && container != nil {
		key = container.Identifier()
	}
	if key == "" {
		key = "default"
	}
	return p.GetLimiterByKey(key)
}

// GetLimiterByKey returns the AdaptiveRateLimiter corresponding to the specified key.
func (p *Pool) GetLimiterByKey(key string) *AdaptiveRateLimiter {
	p.mu.Lock()
	defer p.mu.Unlock()

	if limiter, ok := p.limiters[key]; ok {
		return limiter
	}

	limiter := NewAdaptiveRateLimiter(p.config)
	p.limiters[key] = limiter
	return limiter
}
