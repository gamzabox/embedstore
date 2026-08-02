// Package openai implements the OpenAI embeddings API using net/http.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	ErrMissingAPIKey        = errors.New("openai API key is required")
	ErrUnsupportedEmbedding = errors.New("unsupported embedding")
	ErrInvalidResponse      = errors.New("invalid OpenAI embedding response")
)

type Client struct {
	apiKey, baseURL, model string
	dimensions, maxRetries int
	retryDelay             time.Duration
	httpClient             *http.Client
}
type Option func(*Client)

func WithAPIKey(s string) Option            { return func(c *Client) { c.apiKey = s } }
func WithBaseURL(s string) Option           { return func(c *Client) { c.baseURL = strings.TrimRight(s, "/") } }
func WithDimensions(n int) Option           { return func(c *Client) { c.dimensions = n } }
func WithMaxRetries(n int) Option           { return func(c *Client) { c.maxRetries = n } }
func WithRetryDelay(d time.Duration) Option { return func(c *Client) { c.retryDelay = d } }
func WithModel(m string) Option             { return func(c *Client) { c.model = m } }
func WithHTTPClient(h *http.Client) Option  { return func(c *Client) { c.httpClient = h } }
func New(opts ...Option) *Client {
	c := &Client{apiKey: os.Getenv("OPENAI_API_KEY"), baseURL: "https://api.openai.com/v1", model: "text-embedding-3-small", maxRetries: 5, retryDelay: 200 * time.Millisecond, httpClient: &http.Client{Timeout: 30 * time.Second}}
	for _, o := range opts {
		o(c)
	}
	return c
}
func ParseEmbedding(s string) (string, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] != "openai" || parts[1] == "" {
		return "", ErrUnsupportedEmbedding
	}
	return parts[1], nil
}
func (c *Client) Embedding() string { return "openai/" + c.model }
func (c *Client) Dimensions() int   { return c.dimensions }
func (c *Client) Embed(ctx context.Context, input []string) ([][]float32, error) {
	if c.apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	if len(input) == 0 {
		return [][]float32{}, nil
	}
	body := struct {
		Model      string   `json:"model"`
		Input      []string `json:"input"`
		Dimensions int      `json:"dimensions,omitempty"`
	}{c.model, input, c.dimensions}
	raw, _ := json.Marshal(body)
	var last error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var decoded struct {
				Data []struct {
					Index     int       `json:"index"`
					Embedding []float32 `json:"embedding"`
				} `json:"data"`
			}
			err = json.NewDecoder(resp.Body).Decode(&decoded)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
			}
			out := make([][]float32, len(input))
			for _, v := range decoded.Data {
				if v.Index < 0 || v.Index >= len(out) || out[v.Index] != nil {
					return nil, ErrInvalidResponse
				}
				if c.dimensions > 0 && len(v.Embedding) != c.dimensions {
					return nil, ErrInvalidResponse
				}
				out[v.Index] = v.Embedding
			}
			for _, v := range out {
				if v == nil {
					return nil, ErrInvalidResponse
				}
			}
			return out, nil
		}
		if resp != nil {
			resp.Body.Close()
			if resp.StatusCode < 500 && resp.StatusCode != 429 {
				return nil, fmt.Errorf("OpenAI embeddings request failed: %s", resp.Status)
			}
		}
		last = err
		if last == nil {
			last = errors.New("transient OpenAI response")
		}
		if attempt < c.maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(1<<attempt)):
			}
		}
	}
	return nil, last
}
