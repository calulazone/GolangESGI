package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Note struct {
	Title   string
	Content string
	Tags    []string
}

func newNote(title, content string, tags []string) *Note {
	return &Note{
		Title:   title,
		Content: content,
		Tags:    tags,
	}
}

func preview(note *Note) string {
	limit := min(len(note.Content), 80)
	return note.Content[:limit]
}

func addTag(note *Note, tag string) {
	for _, t := range note.Tags {
		if t == tag {
			return
		}
	}
	note.Tags = append(note.Tags, tag)
}

func hasGoTag(note *Note) bool {
	for _, t := range note.Tags {
		if t == "go" {
			return true
		}
	}
	return false
}

func LoadFromFile(path string) ([]*Note, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var notes []*Note

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&notes)
	if err != nil {
		return nil, err
	}

	return notes, nil
}

func main() {
	notes, err := LoadFromFile("notes.json")
	if err != nil {
		panic(err)
	}
	for _, note := range notes {
		if hasGoTag(note) {
			fmt.Println("Title:", note.Title)
			fmt.Println("Preview:", preview(note))
			fmt.Println("Tags:", note.Tags)
			fmt.Println()
		}
	}
}
