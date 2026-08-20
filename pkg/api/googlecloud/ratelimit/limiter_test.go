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
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/google/go-cmp/cmp"
)

func TestNewAdaptiveRateLimiter_DefaultValues(t *testing.T) {
	testCases := []struct {
		name       string
		config     AdaptiveRateLimiterConfig
		wantMin    float64
		wantMax    float64
		wantInit   float64
		wantStep   float64
		wantFactor float64
	}{
		{
			name: "valid custom config",
			config: AdaptiveRateLimiterConfig{
				InitialQPS:       5.0,
				MinQPS:           1.0,
				MaxQPS:           50.0,
				IncreaseStep:     2.0,
				DecreaseFactor:   0.6,
				DecreaseCooldown: 500 * time.Millisecond,
				Burst:            2,
			},
			wantMin:    1.0,
			wantMax:    50.0,
			wantInit:   5.0,
			wantStep:   2.0,
			wantFactor: 0.6,
		},
		{
			name:       "empty config normalized to defaults",
			config:     AdaptiveRateLimiterConfig{},
			wantMin:    1.0,
			wantMax:    1.0, // When MaxQPS=0 < MinQPS=1.0, MaxQPS normalized to MinQPS
			wantInit:   1.0,
			wantStep:   0.5,
			wantFactor: 0.5,
		},
		{
			name: "initial out of bounds clamped",
			config: AdaptiveRateLimiterConfig{
				InitialQPS: 200.0,
				MinQPS:     1.0,
				MaxQPS:     50.0,
			},
			wantMin:    1.0,
			wantMax:    50.0,
			wantInit:   50.0,
			wantStep:   0.5,
			wantFactor: 0.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewAdaptiveRateLimiter(tc.config)
			gotConfig := limiter.Config()
			if diff := cmp.Diff(tc.wantMin, gotConfig.MinQPS); diff != "" {
				t.Errorf("MinQPS mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantMax, gotConfig.MaxQPS); diff != "" {
				t.Errorf("MaxQPS mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantInit, limiter.CurrentRate()); diff != "" {
				t.Errorf("CurrentRate() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantStep, gotConfig.IncreaseStep); diff != "" {
				t.Errorf("IncreaseStep mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantFactor, gotConfig.DecreaseFactor); diff != "" {
				t.Errorf("DecreaseFactor mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdaptiveRateLimiter_OnSuccess(t *testing.T) {
	testCases := []struct {
		name         string
		initialQPS   float64
		maxQPS       float64
		increaseStep float64
		successCount int
		wantQPS      float64
	}{
		{
			name:         "increments by step up to max",
			initialQPS:   1.0,
			maxQPS:       5.0,
			increaseStep: 1.0,
			successCount: 3,
			wantQPS:      4.0,
		},
		{
			name:         "capped at maxQPS",
			initialQPS:   1.0,
			maxQPS:       2.5,
			increaseStep: 1.0,
			successCount: 5,
			wantQPS:      2.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewAdaptiveRateLimiter(AdaptiveRateLimiterConfig{
				InitialQPS:   tc.initialQPS,
				MinQPS:       0.1,
				MaxQPS:       tc.maxQPS,
				IncreaseStep: tc.increaseStep,
			})
			for i := 0; i < tc.successCount; i++ {
				limiter.OnSuccess()
			}
			got := limiter.CurrentRate()
			if diff := cmp.Diff(tc.wantQPS, got); diff != "" {
				t.Errorf("CurrentRate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdaptiveRateLimiter_OnResourceExhausted(t *testing.T) {
	testCases := []struct {
		name           string
		initialQPS     float64
		minQPS         float64
		decreaseFactor float64
		wantQPS        float64
	}{
		{
			name:           "decreases by factor",
			initialQPS:     10.0,
			minQPS:         0.1,
			decreaseFactor: 0.5,
			wantQPS:        5.0,
		},
		{
			name:           "clamped at minQPS",
			initialQPS:     0.15,
			minQPS:         0.1,
			decreaseFactor: 0.5,
			wantQPS:        0.1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewAdaptiveRateLimiter(AdaptiveRateLimiterConfig{
				InitialQPS:       tc.initialQPS,
				MinQPS:           tc.minQPS,
				MaxQPS:           100.0,
				DecreaseFactor:   tc.decreaseFactor,
				DecreaseCooldown: time.Second,
			})
			limiter.OnResourceExhausted()
			got := limiter.CurrentRate()
			if diff := cmp.Diff(tc.wantQPS, got); diff != "" {
				t.Errorf("CurrentRate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdaptiveRateLimiter_CooldownSuppression(t *testing.T) {
	testCases := []struct {
		name             string
		initialQPS       float64
		decreaseCooldown time.Duration
		callInterval     time.Duration
		totalCalls       int
		callSuccessAfter bool
		wantQPS          float64
	}{
		{
			name:             "consecutive calls within cooldown only decrease once",
			initialQPS:       10.0,
			decreaseCooldown: 500 * time.Millisecond,
			callInterval:     10 * time.Millisecond,
			totalCalls:       5,
			callSuccessAfter: false,
			wantQPS:          5.0, // Only first call decreases from 10.0 to 5.0
		},
		{
			name:             "success during cooldown does not increase rate",
			initialQPS:       10.0,
			decreaseCooldown: 500 * time.Millisecond,
			callInterval:     0,
			totalCalls:       1,
			callSuccessAfter: true,
			wantQPS:          5.0, // Decreased to 5.0, and OnSuccess during cooldown is suppressed
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewAdaptiveRateLimiter(AdaptiveRateLimiterConfig{
				InitialQPS:       tc.initialQPS,
				MinQPS:           0.1,
				MaxQPS:           100.0,
				IncreaseStep:     1.0,
				DecreaseFactor:   0.5,
				DecreaseCooldown: tc.decreaseCooldown,
			})
			for i := 0; i < tc.totalCalls; i++ {
				limiter.OnResourceExhausted()
				if tc.callInterval > 0 {
					time.Sleep(tc.callInterval)
				}
			}
			if tc.callSuccessAfter {
				limiter.OnSuccess()
			}
			got := limiter.CurrentRate()
			if diff := cmp.Diff(tc.wantQPS, got); diff != "" {
				t.Errorf("CurrentRate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAdaptiveRateLimiter_Wait(t *testing.T) {
	testCases := []struct {
		name        string
		cancelCtx   bool
		wantErr     bool
		expectedErr error
	}{
		{
			name:      "immediate success with token available",
			cancelCtx: false,
			wantErr:   false,
		},
		{
			name:        "cancelled context returns error",
			cancelCtx:   true,
			wantErr:     true,
			expectedErr: context.Canceled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewAdaptiveRateLimiter(AdaptiveRateLimiterConfig{
				InitialQPS: 10.0,
				MinQPS:     0.1,
				MaxQPS:     100.0,
				Burst:      1,
			})

			ctx, cancel := context.WithCancel(context.Background())
			if tc.cancelCtx {
				cancel()
			} else {
				defer cancel()
			}

			err := limiter.Wait(ctx)
			if (err != nil) != tc.wantErr {
				t.Errorf("Wait() error = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestAdaptiveRateLimiter_Concurrency(t *testing.T) {
	limiter := NewAdaptiveRateLimiter(AdaptiveRateLimiterConfig{
		InitialQPS:       5.0,
		MinQPS:           0.1,
		MaxQPS:           20.0,
		IncreaseStep:     0.1,
		DecreaseFactor:   0.9,
		DecreaseCooldown: 10 * time.Millisecond,
		Burst:            5,
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if id%2 == 0 {
					limiter.OnSuccess()
				} else {
					limiter.OnResourceExhausted()
				}
				_ = limiter.CurrentRate()
			}
		}(i)
	}
	wg.Wait()

	rate := limiter.CurrentRate()
	if rate < 1.0 || rate > 20.0 {
		t.Errorf("CurrentRate() out of bounds: %f", rate)
	}
}

func TestPool_GetLimiter(t *testing.T) {
	testCases := []struct {
		name           string
		quotaProjectID string
		container1     googlecloud.ResourceContainer
		container2     googlecloud.ResourceContainer
		wantSame       bool
	}{
		{
			name:           "with quota project returns same limiter for different containers",
			quotaProjectID: "quota-proj-123",
			container1:     googlecloud.Project("project-a"),
			container2:     googlecloud.Project("project-b"),
			wantSame:       true,
		},
		{
			name:           "without quota project returns different limiters for different containers",
			quotaProjectID: "",
			container1:     googlecloud.Project("project-a"),
			container2:     googlecloud.Project("project-b"),
			wantSame:       false,
		},
		{
			name:           "without quota project returns same limiter for identical container",
			quotaProjectID: "",
			container1:     googlecloud.Project("project-a"),
			container2:     googlecloud.Project("project-a"),
			wantSame:       true,
		},
		{
			name:           "nil container falls back to default key",
			quotaProjectID: "",
			container1:     nil,
			container2:     nil,
			wantSame:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool := NewPool(DefaultAdaptiveRateLimiterConfig(), tc.quotaProjectID)
			limiter1 := pool.GetLimiter(tc.container1)
			limiter2 := pool.GetLimiter(tc.container2)

			gotSame := (limiter1 == limiter2)
			if gotSame != tc.wantSame {
				t.Errorf("GetLimiter() same instance = %v, wantSame = %v", gotSame, tc.wantSame)
			}
		})
	}
}
