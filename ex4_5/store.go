package store

import (
	"errors"
	"sync"
)

var ErrDuplicate = errors.New("note already exists")
var ErrNotFound = errors.New("note not found")
var ErrValidation = errors.New("validation failed: empty title")

type Note struct {
	ID      string
	Title   string
	Content string
	Tags    []string
}

type NoteStore interface {
	Save(n *Note) error
	Get(id string) (*Note, error)
	All() []*Note
}

type MemoryStore struct {
	mu sync.Mutex
	m  map[string]*Note
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		m: make(map[string]*Note),
	}
}

func (s *MemoryStore) Save(n *Note) error {
	if n == nil || n.Title == "" {
		return ErrValidation
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if n.ID == "" {
		return ErrValidation
	}

	if _, ok := s.m[n.ID]; ok {
		return ErrDuplicate
	}

	s.m[n.ID] = n
	return nil
}

func (s *MemoryStore) Get(id string) (*Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, ok := s.m[id]
	if !ok {
		return nil, ErrNotFound
	}
	return n, nil
}

func (s *MemoryStore) All() []*Note {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]*Note, 0, len(s.m))
	for _, n := range s.m {
		out = append(out, n)
	}
	return out
}
