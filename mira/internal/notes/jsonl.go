package notes

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type JSONLStore struct {
	path string
}

func NewJSONLStore(path string) (NoteStore, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(home, ".mira")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
		path = filepath.Join(dir, "notes.jsonl")
	}
	// ensure file exists
	f, err := os.OpenFile(path, os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &JSONLStore{path: path}, nil
}

func (s *JSONLStore) Save(n *Note) error {
	if n == nil {
		return errors.New("nil note")
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(n)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *JSONLStore) All() ([]*Note, error) {
	f, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []*Note
	rdr := bufio.NewReader(f)
	for {
		line, err := rdr.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(line) == 0 {
					break
				}
				// fallthrough to decode last line
			} else {
				return nil, err
			}
		}
		var n Note
		if err := json.Unmarshal(line, &n); err != nil {
			// skip malformed lines
			continue
		}
		out = append(out, &n)
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return out, nil
}

func (s *JSONLStore) List(limit int) ([]*Note, error) {
	all, err := s.All()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit >= len(all) {
		return all, nil
	}
	return all[len(all)-limit:], nil
}
