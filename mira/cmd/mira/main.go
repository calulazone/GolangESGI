package main

import (
	"fmt"
	"log"
	"os"

	"mira/internal/httpclient"
)

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  mira add \"title\" \"content\" [tag1,tag2,...]")
	fmt.Println("  mira list")
	fmt.Println("  mira search <query>")
	fmt.Println()
	fmt.Println("Talks to the mira API (default http://localhost:8080, override with MIRA_API_URL).")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	apiURL := os.Getenv("MIRA_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	client := httpclient.New(apiURL)

	switch cmd := os.Args[1]; cmd {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("add requires title and content")
			os.Exit(2)
		}
		title, content := os.Args[2], os.Args[3]
		n, err := client.CreateNote(title, content, nil)
		if err != nil {
			log.Fatalf("create note: %v", err)
		}
		fmt.Println("Saved:", n.ID, "(enrichment:", n.EnrichmentStatus+")")

	case "list":
		list, err := client.ListNotes()
		if err != nil {
			log.Fatalf("list notes: %v", err)
		}
		printNotes(list)

	case "search":
		if len(os.Args) < 3 {
			fmt.Println("search requires a query")
			os.Exit(2)
		}
		list, err := client.Search(os.Args[2])
		if err != nil {
			log.Fatalf("search: %v", err)
		}
		printNotes(list)

	default:
		usage()
		os.Exit(1)
	}
}

func printNotes(list []httpclient.Note) {
	for _, n := range list {
		fmt.Printf("%s  %s  [%s]\n", n.ID, n.Title, n.EnrichmentStatus)
		if n.Summary != "" {
			fmt.Println("  ", n.Summary)
		} else {
			fmt.Println("  ", preview(n.Content))
		}
		fmt.Println()
	}
}

func preview(content string) string {
	if len(content) <= 80 {
		return content
	}
	return content[:80] + "…"
}
