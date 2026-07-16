package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mira/internal/embed"
	"mira/internal/enrichment"
	"mira/internal/notes"
)

type Handler struct {
	store    notes.NoteStore
	enricher *enrichment.Enricher
	embedder embed.Embedder
	logger   *slog.Logger
}

func NewHandler(store notes.NoteStore, enricher *enrichment.Enricher, embedder embed.Embedder, logger *slog.Logger) *Handler {
	return &Handler{store: store, enricher: enricher, embedder: embedder, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case path == "/api/v1/notes":
		h.handleNotesCollection(w, r)
	case strings.HasPrefix(path, "/api/v1/notes/"):
		h.handleNoteByID(w, r, strings.TrimPrefix(path, "/api/v1/notes/"))
	case path == "/api/v1/search":
		h.handleSearch(w, r)
	default:
		h.writeJSON(w, http.StatusNotFound, errorResponse("route not found"))
	}
}

func (h *Handler) handleNotesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listNotes(w, r)
	case http.MethodPost:
		h.createNote(w, r)
	default:
		h.writeJSON(w, http.StatusMethodNotAllowed, errorResponse("method not allowed"))
	}
}

func (h *Handler) handleNoteByID(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		h.writeJSON(w, http.StatusNotFound, errorResponse("route not found"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getNote(w, r, id)
	case http.MethodPatch:
		h.updateNote(w, r, id)
	case http.MethodDelete:
		h.deleteNote(w, r, id)
	default:
		h.writeJSON(w, http.StatusMethodNotAllowed, errorResponse("method not allowed"))
	}
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeJSON(w, http.StatusMethodNotAllowed, errorResponse("method not allowed"))
		return
	}
	h.searchNotes(w, r)
}

func (h *Handler) createNote(w http.ResponseWriter, r *http.Request) {
	var input notes.CreateNoteInput
	if err := decodePayload(r, &input); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("invalid json payload"))
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("title is required"))
		return
	}
	if strings.TrimSpace(input.Content) == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("content is required"))
		return
	}

	n := &notes.Note{Title: input.Title, Content: input.Content, Tags: input.Tags}
	if err := h.store.Create(r.Context(), n); err != nil {
		h.writeJSON(w, http.StatusInternalServerError, errorResponse("could not create note"))
		return
	}

	h.enricher.Publish(enrichment.Job{NoteID: n.ID, Title: n.Title, Content: n.Content})

	h.writeJSON(w, http.StatusCreated, successResponse(n))
}

func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.List(r.Context(), parseLimit(r, 20))
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, errorResponse("could not list notes"))
		return
	}
	h.writeJSON(w, http.StatusOK, successResponse(list))
}

func (h *Handler) getNote(w http.ResponseWriter, r *http.Request, id string) {
	note, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.respondStoreErr(w, err, "note not found")
		return
	}
	h.writeJSON(w, http.StatusOK, successResponse(note))
}

func (h *Handler) updateNote(w http.ResponseWriter, r *http.Request, id string) {
	var input notes.UpdateNoteInput
	if err := decodePayload(r, &input); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("invalid json payload"))
		return
	}
	if input.Title == nil && input.Content == nil && input.Tags == nil {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("at least one field is required"))
		return
	}
	if input.Title != nil && strings.TrimSpace(*input.Title) == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("title cannot be empty"))
		return
	}
	if input.Content != nil && strings.TrimSpace(*input.Content) == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("content cannot be empty"))
		return
	}

	note, err := h.store.Update(r.Context(), id, input)
	if err != nil {
		h.respondStoreErr(w, err, "note not found")
		return
	}

	if input.Title != nil || input.Content != nil {
		h.enricher.Publish(enrichment.Job{NoteID: note.ID, Title: note.Title, Content: note.Content})
	}

	h.writeJSON(w, http.StatusOK, successResponse(note))
}

func (h *Handler) deleteNote(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.respondStoreErr(w, err, "note not found")
		return
	}
	h.writeJSON(w, http.StatusOK, successResponse(nil))
}

func (h *Handler) searchNotes(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("q query parameter is required"))
		return
	}

	embedCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	var queryEmbedding []float32
	if emb, err := h.embedder.Embed(embedCtx, q); err != nil {
		h.logger.Warn("query embedding failed, falling back to full-text search", "err", err)
	} else {
		queryEmbedding = emb
	}

	results, err := h.store.SearchHybrid(r.Context(), q, queryEmbedding, parseLimit(r, 20))
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, errorResponse("search failed"))
		return
	}
	h.writeJSON(w, http.StatusOK, successResponse(results))
}

// parseLimit reads an optional ?limit= query parameter, falling back to
// def when absent, invalid, or non-positive.
func parseLimit(r *http.Request, def int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func (h *Handler) respondStoreErr(w http.ResponseWriter, err error, notFoundMsg string) {
	if errors.Is(err, notes.ErrNotFound) {
		h.writeJSON(w, http.StatusNotFound, errorResponse(notFoundMsg))
		return
	}
	if errors.Is(err, notes.ErrValidation) {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("invalid input"))
		return
	}
	h.logger.Error("store error", "err", err)
	h.writeJSON(w, http.StatusInternalServerError, errorResponse("internal server error"))
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func successResponse(data any) map[string]any {
	return map[string]any{"success": true, "data": data}
}

func errorResponse(message string) map[string]any {
	return map[string]any{"success": false, "error": message}
}

func decodePayload(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
