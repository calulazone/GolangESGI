package store

import (
	"errors"
	"testing"
)

func TestSave_valid(t *testing.T) {
	s := NewMemoryStore()
	n := &Note{ID: "1", Title: "Title", Content: "content"}
	if err := s.Save(n); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestSave_emptyTitle(t *testing.T) {
	s := NewMemoryStore()
	n := &Note{ID: "2", Title: ""}
	err := s.Save(n)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestSave_duplicate(t *testing.T) {
	s := NewMemoryStore()
	n := &Note{ID: "3", Title: "T"}
	if err := s.Save(n); err != nil {
		t.Fatalf("unexpected error on first save: %v", err)
	}
	if err := s.Save(n); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate on second save, got %v", err)
	}
}

func TestGet_notFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get("?")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
