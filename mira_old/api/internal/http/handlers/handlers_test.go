package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/m/v2/internal/store"
)

func TestCreateNoteHandler_SuccessAndBadRequest(t *testing.T) {
	h := NewHandler(store.NewMemoryStore())

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"title":"Titre","content":"Contenu"}`))
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}

		var resp struct {
			Success bool           `json:"success"`
			Data    map[string]any `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if !resp.Success {
			t.Fatalf("expected success response")
		}
		if resp.Data["title"] != "Titre" {
			t.Fatalf("expected title in response data")
		}
	})

	t.Run("bad request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", strings.NewReader(`{"content":"Contenu"}`))
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestGetNoteHandler_NotFound(t *testing.T) {
	h := NewHandler(store.NewMemoryStore())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes/unknown", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
