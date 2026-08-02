package embedstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const DefaultSearchLimit = 5

var (
	ErrInvalidFile        = errors.New("invalid embed file")
	ErrUnsupportedVersion = errors.New("unsupported version")
	ErrDimensionMismatch  = errors.New("dimension mismatch")
	ErrModelMismatch      = errors.New("model mismatch")
	ErrInvalidQuery       = errors.New("invalid query")
)

type Manifest struct {
	FormatVersion   int       `json:"formatVersion"`
	DatasetName     string    `json:"datasetName"`
	DatasetVersion  string    `json:"datasetVersion"`
	Embedding       string    `json:"embedding"`
	Dimensions      int       `json:"dimensions"`
	VectorType      string    `json:"vectorType"`
	Normalized      bool      `json:"normalized"`
	ItemCount       int       `json:"itemCount"`
	CreatedAt       time.Time `json:"createdAt"`
	SourceChecksum  string    `json:"sourceChecksum"`
	ContentIncluded bool      `json:"contentIncluded"`
}
type Item struct {
	ID      string          `json:"id"`
	Content string          `json:"content,omitempty"`
	Data    json.RawMessage `json:"data"`
}
type SearchOptions struct {
	Limit    int
	MinScore *float32
}
type SearchResult struct {
	Rank    int             `json:"rank"`
	ID      string          `json:"id"`
	Score   float32         `json:"score"`
	Content string          `json:"content,omitempty"`
	Data    json.RawMessage `json:"data"`
	Vector  []float32       `json:"vector,omitempty"`
}
type Store interface {
	Manifest() Manifest
	Count() int
	SearchVector(context.Context, []float32, SearchOptions) ([]SearchResult, error)
	Close() error
}
type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
	Embedding() string
	Dimensions() int
}
type MemoryStore struct {
	manifest Manifest
	items    []Item
	vectors  []float32
}

func NewMemoryStore(m Manifest, items []Item, vectors []float32) (*MemoryStore, error) {
	if m.Dimensions <= 0 || m.ItemCount != len(items) || len(vectors) != len(items)*m.Dimensions {
		return nil, fmt.Errorf("%w: invalid shape", ErrInvalidFile)
	}
	clonedItems := make([]Item, len(items))
	for i, item := range items {
		clonedItems[i] = item
		clonedItems[i].Data = append(json.RawMessage(nil), item.Data...)
	}
	vv := append([]float32(nil), vectors...)
	for i := range items {
		if items[i].ID == "" || !json.Valid(items[i].Data) {
			return nil, fmt.Errorf("%w: invalid item", ErrInvalidFile)
		}
		if err := normalize(vv[i*m.Dimensions : (i+1)*m.Dimensions]); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidFile, err)
		}
	}
	m.Normalized = true
	return &MemoryStore{m, clonedItems, vv}, nil
}
func (s *MemoryStore) Manifest() Manifest { return s.manifest }
func (s *MemoryStore) Count() int         { return len(s.items) }

// Items returns a copy of stored metadata in file order.
func (s *MemoryStore) Items() []Item {
	items := make([]Item, len(s.items))
	for i, item := range s.items {
		items[i] = item
		items[i].Data = append(json.RawMessage(nil), item.Data...)
	}
	return items
}
func (s *MemoryStore) Close() error { return nil }
func (s *MemoryStore) SearchVector(ctx context.Context, q []float32, o SearchOptions) ([]SearchResult, error) {
	if len(q) != s.manifest.Dimensions {
		return nil, ErrDimensionMismatch
	}
	q = append([]float32(nil), q...)
	if normalize(q) != nil {
		return nil, ErrInvalidQuery
	}
	l := o.Limit
	if l == 0 {
		l = DefaultSearchLimit
	}
	if l < 0 {
		return nil, ErrInvalidQuery
	}
	type c struct {
		i int
		r SearchResult
	}
	cs := []c{}
	for i, x := range s.items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var score float32
		for j := range q {
			score += q[j] * s.vectors[i*s.manifest.Dimensions+j]
		}
		if o.MinScore == nil || score >= *o.MinScore {
			cs = append(cs, c{i, SearchResult{ID: x.ID, Score: score, Content: x.Content, Data: x.Data, Vector: append([]float32(nil), s.vectors[i*s.manifest.Dimensions:(i+1)*s.manifest.Dimensions]...)}})
		}
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].r.Score != cs[j].r.Score {
			return cs[i].r.Score > cs[j].r.Score
		}
		return cs[i].i < cs[j].i
	})
	if l > len(cs) {
		l = len(cs)
	}
	out := make([]SearchResult, l)
	for i := range out {
		out[i] = cs[i].r
		out[i].Rank = i + 1
	}
	return out, nil
}
func normalize(v []float32) error {
	var n float64
	for _, x := range v {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return ErrInvalidQuery
		}
		n += float64(x) * float64(x)
	}
	if n == 0 {
		return ErrInvalidQuery
	}
	d := float32(math.Sqrt(n))
	for i := range v {
		v[i] /= d
	}
	return nil
}

type Engine struct {
	store    Store
	embedder Embedder
}

func NewEngine(s Store, e Embedder) (*Engine, error) {
	if s == nil || e == nil {
		return nil, ErrInvalidQuery
	}
	m := s.Manifest()
	if m.Embedding != e.Embedding() {
		return nil, ErrModelMismatch
	}
	if m.Dimensions != e.Dimensions() {
		return nil, ErrDimensionMismatch
	}
	return &Engine{s, e}, nil
}
func (e *Engine) Search(c context.Context, q string, o SearchOptions) ([]SearchResult, error) {
	if strings.TrimSpace(q) == "" {
		return nil, ErrInvalidQuery
	}
	v, err := e.embedder.Embed(c, []string{q})
	if err != nil {
		return nil, err
	}
	if len(v) != 1 {
		return nil, ErrInvalidQuery
	}
	return e.store.SearchVector(c, v[0], o)
}
func DecodeData[T any](result SearchResult) (T, error) {
	var v T
	err := json.Unmarshal(result.Data, &v)
	return v, err
}

// FileError adds a path and byte offset to a file validation error.
type FileError struct {
	Path   string
	Offset int64
	Err    error
}

func (e *FileError) Error() string {
	return fmt.Sprintf("embed file %s at offset %d: %v", e.Path, e.Offset, e.Err)
}
func (e *FileError) Unwrap() error { return e.Err }
