package notes

import "time"

// EnrichmentStatus tracks the lifecycle of the async enrichment job for a note.
type EnrichmentStatus string

const (
	EnrichmentPending EnrichmentStatus = "pending"
	EnrichmentDone    EnrichmentStatus = "done"
	EnrichmentFailed  EnrichmentStatus = "failed"
)

// Note is the persisted representation of a note, including enrichment
// data produced asynchronously by the enrichment workers.
type Note struct {
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	Content          string           `json:"content"`
	Tags             []string         `json:"tags,omitempty"`
	Summary          string           `json:"summary,omitempty"`
	Score            float64          `json:"score,omitempty"`
	Embedding        []float32        `json:"-"` // never serialized to API clients
	EnrichmentStatus EnrichmentStatus `json:"enrichment_status"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type CreateNoteInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
}

type UpdateNoteInput struct {
	Title   *string  `json:"title,omitempty"`
	Content *string  `json:"content,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}
