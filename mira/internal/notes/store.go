package notes

import (
	"context"
	"errors"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("note not found")
)

// NoteStore is implemented by PostgresStore. It is the single persistence
// boundary used by both the HTTP handlers and the enrichment workers, so
// that writing a note and writing its enrichment result go through the
// same contract.
type NoteStore interface {
	Create(ctx context.Context, n *Note) error
	Get(ctx context.Context, id string) (*Note, error)
	List(ctx context.Context, limit int) ([]*Note, error)
	Update(ctx context.Context, id string, input UpdateNoteInput) (*Note, error)
	Delete(ctx context.Context, id string) error

	// SearchHybrid combines full-text rank and vector cosine similarity.
	// queryEmbedding may be nil, in which case this behaves like Search.
	SearchHybrid(ctx context.Context, query string, queryEmbedding []float32, limit int) ([]*Note, error)

	// SetEnrichment persists the result of an enrichment job.
	SetEnrichment(ctx context.Context, id string, status EnrichmentStatus, summary string, score float64, tags []string, embedding []float32) error
}
