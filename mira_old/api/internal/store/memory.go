package store

import (
	"fmt"
	"sync"

	"example.com/m/v2/internal/core"
)

type MemoryStore struct {
	mu     sync.RWMutex
	notes  map[string]core.Note
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{notes: make(map[string]core.Note)}
}

func (s *MemoryStore) Create(note core.Note) core.Note {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	note.ID = fmt.Sprintf("note-%d", s.nextID)
	s.notes[note.ID] = note
	return note
}

func (s *MemoryStore) List() []core.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	notes := make([]core.Note, 0, len(s.notes))
	for _, note := range s.notes {
		notes = append(notes, note)
	}
	return notes
}

func (s *MemoryStore) Get(id string) (core.Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	note, ok := s.notes[id]
	return note, ok
}

func (s *MemoryStore) Update(id string, input core.UpdateNoteInput) (core.Note, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[id]
	if !ok {
		return core.Note{}, false
	}
	if input.Title != nil {
		note.Title = *input.Title
	}
	if input.Content != nil {
		note.Content = *input.Content
	}
	if input.Tags != nil {
		note.Tags = input.Tags
	}
	s.notes[id] = note
	return note, true
}

func (s *MemoryStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.notes[id]; !ok {
		return false
	}
	delete(s.notes, id)
	return true
}

func (s *MemoryStore) Search(query string) []core.Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := make([]core.Note, 0)
	for _, note := range s.notes {
		if contains(note.Title, query) || contains(note.Content, query) {
			matches = append(matches, note)
		}
	}
	return matches
}

func contains(value, query string) bool {
	return len(query) == 0 || (len(value) >= len(query) && (value == query || containsSubstring(value, query)))
}

func containsSubstring(value, query string) bool {
	for i := 0; i+len(query) <= len(value); i++ {
		if value[i:i+len(query)] == query {
			return true
		}
	}
	return false
}
