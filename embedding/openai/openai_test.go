package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmbedSendsConfiguredRequestAndOrdersResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/embeddings" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		var request struct {
			Model      string   `json:"model"`
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "text-embedding-3-small" || request.Dimensions != 2 || len(request.Input) != 2 {
			t.Fatalf("request = %#v", request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"index": 1, "embedding": []float32{0, 1}},
			map[string]any{"index": 0, "embedding": []float32{1, 0}},
		}})
	}))
	defer server.Close()

	client := New(WithAPIKey("test-key"), WithBaseURL(server.URL+"/v1"), WithDimensions(2))
	got, err := client.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0][0] != 1 || got[1][1] != 1 {
		t.Fatalf("vectors = %#v", got)
	}
	if client.Embedding() != "openai/text-embedding-3-small" || client.Dimensions() != 2 {
		t.Fatalf("identity = %q/%d", client.Embedding(), client.Dimensions())
	}
}

func TestEmbedRetriesTransientStatus(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": []float32{1, 0}}}})
	}))
	defer server.Close()
	client := New(WithAPIKey("key"), WithBaseURL(server.URL), WithMaxRetries(1), WithRetryDelay(time.Millisecond))
	if _, err := client.Embed(context.Background(), []string{"one"}); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestEmbedRejectsInvalidResponsesAndMissingKey(t *testing.T) {
	t.Parallel()
	if _, err := New().Embed(context.Background(), []string{"one"}); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("missing key error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 1, "embedding": []float32{1, 0}}}})
	}))
	defer server.Close()
	_, err := New(WithAPIKey("key"), WithBaseURL(server.URL)).Embed(context.Background(), []string{"one"})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("invalid response error = %v", err)
	}
}

func TestParseEmbedding(t *testing.T) {
	t.Parallel()
	model, err := ParseEmbedding("openai/text-embedding-3-large")
	if err != nil || model != "text-embedding-3-large" {
		t.Fatalf("ParseEmbedding() = %q, %v", model, err)
	}
	if _, err := ParseEmbedding("other/model"); !errors.Is(err, ErrUnsupportedEmbedding) {
		t.Fatalf("provider error = %v", err)
	}
}
