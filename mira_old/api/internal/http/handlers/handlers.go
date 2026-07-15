package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"example.com/m/v2/internal/core"
	"example.com/m/v2/internal/store"
)

type Handler struct {
	store *store.MemoryStore
}

func NewHandler(st *store.MemoryStore) *Handler {
	return &Handler{store: st}
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
	var input core.CreateNoteInput
	if err := decodeCreatePayload(r, &input); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errorResponse("invalid json payload"))
		return
	}
	if err := validateCreateInput(input); err != nil {
		h.writeJSON(w, http.StatusBadRequest, errorResponse(err.Error()))
		return
	}

	note := h.store.Create(core.Note{Title: input.Title, Content: input.Content, Tags: input.Tags})
	h.writeJSON(w, http.StatusCreated, successResponse(note))
}

func (h *Handler) listNotes(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, successResponse(h.store.List()))
}

func (h *Handler) getNote(w http.ResponseWriter, r *http.Request, id string) {
	note, ok := h.store.Get(id)
	if !ok {
		h.writeJSON(w, http.StatusNotFound, errorResponse("note not found"))
		return
	}
	h.writeJSON(w, http.StatusOK, successResponse(note))
}

func (h *Handler) updateNote(w http.ResponseWriter, r *http.Request, id string) {
	var input core.UpdateNoteInput
	if err := decodeUpdatePayload(r, &input); err != nil {
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

	note, ok := h.store.Update(id, input)
	if !ok {
		h.writeJSON(w, http.StatusNotFound, errorResponse("note not found"))
		return
	}
	h.writeJSON(w, http.StatusOK, successResponse(note))
}

func (h *Handler) deleteNote(w http.ResponseWriter, r *http.Request, id string) {
	if !h.store.Delete(id) {
		h.writeJSON(w, http.StatusNotFound, errorResponse("note not found"))
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
	h.writeJSON(w, http.StatusOK, successResponse(h.store.Search(q)))
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

func decodeCreatePayload(r *http.Request, dst *core.CreateNoteInput) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func decodeUpdatePayload(r *http.Request, dst *core.UpdateNoteInput) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func validateCreateInput(input core.CreateNoteInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(input.Content) == "" {
		return errors.New("content is required")
	}
	return nil
}
