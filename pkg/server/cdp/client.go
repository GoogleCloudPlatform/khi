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

package cdp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type request struct {
	ID     int            `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
}

type protocolError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *protocolError  `json:"error"`
}

type evaluateResult struct {
	Result struct {
		Value json.RawMessage `json:"value"`
	} `json:"result"`
	ExceptionDetails *struct {
		Text      string `json:"text"`
		Exception *struct {
			Description string `json:"description"`
		} `json:"exception"`
	} `json:"exceptionDetails"`
}

// TransportError wraps WebSocket send and receive failures so callers can distinguish
// unrecoverable connection errors from transient CDP protocol or JavaScript evaluation errors.
type TransportError struct {
	// Err is the underlying WebSocket transport error.
	Err error
}

// Error returns the string representation of the underlying transport error.
func (e *TransportError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying transport error for errors.Is and errors.As checks.
func (e *TransportError) Unwrap() error {
	return e.Err
}

// Client manages a single Chrome DevTools Protocol WebSocket connection.
type Client struct {
	mu     sync.Mutex
	ws     *websocket.Conn
	nextID int
}

// Dial establishes a new Chrome DevTools Protocol WebSocket client connection to wsURL.
func Dial(ctx context.Context, wsURL string) (*Client, error) {
	cfg, err := websocket.NewConfig(wsURL, "http://127.0.0.1")
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket config: %w", err)
	}
	var d net.Dialer
	rawConn, err := d.DialContext(ctx, "tcp", cfg.Location.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to dial CDP tcp endpoint: %w", err)
	}
	conn, err := websocket.NewClient(cfg, rawConn)
	if err != nil {
		_ = rawConn.Close()
		return nil, fmt.Errorf("failed to establish CDP websocket client: %w", err)
	}
	return &Client{ws: conn}, nil
}

// Navigate enables the Page and Runtime domains and navigates the page to url.
func (c *Client) Navigate(ctx context.Context, url string) error {
	if err := c.call(ctx, "Page.enable", nil, nil); err != nil {
		return err
	}
	if err := c.call(ctx, "Runtime.enable", nil, nil); err != nil {
		return err
	}
	return c.call(ctx, "Page.navigate", map[string]any{"url": url}, nil)
}

// Evaluate executes a JavaScript expression via Runtime.evaluate and decodes the returned value into out.
func (c *Client) Evaluate(ctx context.Context, expression string, awaitPromise bool, out any) error {
	var evalRes evaluateResult
	err := c.call(ctx, "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"awaitPromise":  awaitPromise,
		"returnByValue": true,
	}, &evalRes)
	if err != nil {
		return err
	}
	if evalRes.ExceptionDetails != nil {
		msg := evalRes.ExceptionDetails.Text
		if evalRes.ExceptionDetails.Exception != nil && evalRes.ExceptionDetails.Exception.Description != "" {
			msg = evalRes.ExceptionDetails.Exception.Description
		}
		return fmt.Errorf("javascript evaluation error: %s", msg)
	}
	if out != nil && len(evalRes.Result.Value) > 0 {
		if err := json.Unmarshal(evalRes.Result.Value, out); err != nil {
			return fmt.Errorf("failed to decode evaluation result: %w", err)
		}
	}
	return nil
}

// SetViewportSize overrides the browser device metrics with the specified viewport width and height.
func (c *Client) SetViewportSize(ctx context.Context, width, height int) error {
	return c.call(ctx, "Emulation.setDeviceMetricsOverride", map[string]any{
		"width":             width,
		"height":            height,
		"deviceScaleFactor": 1,
		"mobile":            false,
	}, nil)
}

// CapturePNGScreenshot captures a PNG screenshot of the current page and returns the decoded PNG bytes.
func (c *Client) CapturePNGScreenshot(ctx context.Context) ([]byte, error) {
	var screenshotRes struct {
		Data string `json:"data"`
	}
	if err := c.call(ctx, "Page.captureScreenshot", map[string]any{
		"format": "png",
	}, &screenshotRes); err != nil {
		return nil, err
	}
	pngBytes, err := base64.StdEncoding.DecodeString(screenshotRes.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot PNG data: %w", err)
	}
	return pngBytes, nil
}

// Close closes the underlying WebSocket connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.Close()
}

func (c *Client) call(ctx context.Context, method string, params map[string]any, out any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	reqID := c.nextID

	if deadline, ok := ctx.Deadline(); ok {
		_ = c.ws.SetDeadline(deadline)
	}
	stop := context.AfterFunc(ctx, func() {
		_ = c.ws.SetDeadline(time.Now())
	})
	defer func() {
		stop()
		_ = c.ws.SetDeadline(time.Time{})
	}()

	if err := websocket.JSON.Send(c.ws, request{
		ID:     reqID,
		Method: method,
		Params: params,
	}); err != nil {
		return &TransportError{Err: fmt.Errorf("failed to send CDP command %s: %w", method, err)}
	}

	for {
		var raw string
		if err := websocket.Message.Receive(c.ws, &raw); err != nil {
			return &TransportError{Err: fmt.Errorf("failed to receive CDP response for %s: %w", method, err)}
		}
		var resp response
		if err := json.Unmarshal([]byte(raw), &resp); err != nil {
			continue
		}
		if resp.ID != reqID {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("CDP command %s failed (%d): %s", method, resp.Error.Code, resp.Error.Message)
		}
		if out != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, out); err != nil {
				return fmt.Errorf("failed to unmarshal CDP result for %s: %w", method, err)
			}
		}
		return nil
	}
}
