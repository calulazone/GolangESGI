package main

import (
	"fmt"
	"log"
	"os"

	"mira/internal/notes"
	"mira/internal/search"
)

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  mira add \"title\" \"content\"")
	fmt.Println("  mira list")
	fmt.Println("  mira search <query>")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	store, err := notes.NewJSONLStore("")
	if err != nil {
		log.Fatalf("store init: %v", err)
	}

	switch cmd {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("add requires title and content")
			os.Exit(2)
		}
		title := os.Args[2]
		content := os.Args[3]
		n := notes.NewNote(title, content)
		if err := store.Save(n); err != nil {
			log.Fatalf("save: %v", err)
		}
		fmt.Println("Saved:", n.ID)

	case "list":
		notesList, err := store.List(10)
		if err != nil {
			log.Fatalf("list: %v", err)
		}
		for i := len(notesList) - 1; i >= 0; i-- {
			n := notesList[i]
			fmt.Printf("%s  %s\n", n.ID, n.Title)
			fmt.Println(n.Preview())
			fmt.Println()
		}

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("search requires a query")
			os.Exit(2)
		}
		q := os.Args[2]
		res, err := search.Search(store, q)
		if err != nil {
			log.Fatalf("search: %v", err)
		}
		for _, n := range res {
			fmt.Printf("%s  %s\n", n.ID, n.Title)
			fmt.Println(n.Preview())
			fmt.Println()
		}

	default:
		usage()
		os.Exit(1)
	}
}
