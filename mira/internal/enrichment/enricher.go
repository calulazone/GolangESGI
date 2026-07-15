// Package enrichment runs note enrichment (tags, summary, score, embedding)
// asynchronously off a bounded worker pool, so note creation/update stays
// fast and synchronous while enrichment "follows" in the background.
package enrichment

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"mira/internal/embed"
	"mira/internal/notes"
)

type Job struct {
	NoteID  string
	Title   string
	Content string
}

type Enricher struct {
	store    notes.NoteStore
	embedder embed.Embedder
	jobs     chan Job
	timeout  time.Duration
	logger   *slog.Logger
	wg       sync.WaitGroup
}

func New(store notes.NoteStore, embedder embed.Embedder, workers, queueSize int, timeout time.Duration, logger *slog.Logger) *Enricher {
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = 256
	}
	e := &Enricher{
		store:    store,
		embedder: embedder,
		jobs:     make(chan Job, queueSize),
		timeout:  timeout,
		logger:   logger,
	}
	for i := 0; i < workers; i++ {
		e.wg.Add(1)
		go e.worker()
	}
	return e
}

func (e *Enricher) Publish(job Job) {
	select {
	case e.jobs <- job:
	default:
		e.logger.Warn("enrichment queue full, dropping job", "note_id", job.NoteID)
	}
}

// Close stops accepting new jobs and waits for in-flight ones to finish.
func (e *Enricher) Close() {
	close(e.jobs)
	e.wg.Wait()
}

func (e *Enricher) worker() {
	defer e.wg.Done()
	for job := range e.jobs {
		e.process(job)
	}
}

func (e *Enricher) process(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	tags := extractTags(job.Content)
	summary := summarize(job.Content)
	score := scoreContent(job.Content)

	embedding, err := e.embedder.Embed(ctx, job.Title+"\n"+job.Content)
	if err != nil {
		e.logger.Error("enrichment embedding failed", "note_id", job.NoteID, "err", err)
		if setErr := e.store.SetEnrichment(context.Background(), job.NoteID, notes.EnrichmentFailed, summary, score, tags, nil); setErr != nil {
			e.logger.Error("failed to persist failed enrichment status", "note_id", job.NoteID, "err", setErr)
		}
		return
	}

	if err := e.store.SetEnrichment(ctx, job.NoteID, notes.EnrichmentDone, summary, score, tags, embedding); err != nil {
		e.logger.Error("failed to persist enrichment", "note_id", job.NoteID, "err", err)
	}
}
