package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxResponseBytes = 4 << 20

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	doer    HTTPDoer
	timeout time.Duration
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		doer:    &http.Client{Timeout: timeout},
		timeout: timeout,
	}
}

func NewClientWithDoer(doer HTTPDoer, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{doer: doer, timeout: timeout}
}

type Response struct {
	StatusCode int
	Body       []byte
}

func (r Response) OK() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

func (r Response) DecodeJSON(out any) error {
	if len(r.Body) == 0 {
		return errors.New("empty response body")
	}
	if err := json.Unmarshal(r.Body, out); err != nil {
		return fmt.Errorf("decode json error: %w", err)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, url string, headers map[string]string, body []byte) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return Response{}, fmt.Errorf("build request error: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.doer.Do(req)
	if err != nil {
		return Response{}, sanitizeError(err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		return Response{}, fmt.Errorf("read response error: %w", err)
	}
	if len(data) > maxResponseBytes {
		return Response{}, fmt.Errorf("response exceed %d bytes", maxResponseBytes)
	}

	return Response{
		StatusCode: res.StatusCode,
		Body:       data,
	}, nil
}

func (c *Client) Get(ctx context.Context, url string, headers map[string]string) (Response, error) {
	return c.do(ctx, http.MethodGet, url, headers, nil)
}

func (c *Client) PostJSON(ctx context.Context, url string, headers map[string]string, payload any) (Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, fmt.Errorf("encode request body error: %w", err)
	}
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/json"
	return c.do(ctx, http.MethodPost, url, headers, body)
}
