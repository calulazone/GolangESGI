package search

import (
	"strings"

	"mira/internal/notes"
)

// Search does a naive substring search over title and content (case-insensitive).
func Search(store notes.NoteStore, query string) ([]*notes.Note, error) {
	q := strings.ToLower(query)
	all, err := store.All()
	if err != nil {
		return nil, err
	}
	var out []*notes.Note
	for _, n := range all {
		s := strings.ToLower(n.Title + "\n" + n.Content)
		if strings.Contains(s, q) {
			out = append(out, n)
		}
	}
	return out, nil
}
