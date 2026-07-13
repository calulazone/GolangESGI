package main

import (
	"fmt"
	"os"
	"sort"
)

type wordsCount struct {
	word  string
	count int
}

func display(args []string) error {
	if len(args) <= 1 {
		return fmt.Errorf("pas d'arguments")
	}

	// Compteur des mots
	counter := make(map[string]int)

	for i := 1; i < len(args); i++ {
		counter[args[i]]++
	}

	var tagCount []wordsCount

	for word, count := range counter {
		tagCount = append(tagCount, wordsCount{
			word:  word,
			count: count,
		})
	}

	sort.Slice(tagCount, func(i, j int) bool {
		return tagCount[i].count > tagCount[j].count
	})

	for _, item := range tagCount {
		fmt.Printf("%s : %d\n", item.word, item.count)
	}

	return nil
}

func main() {
	err := display(os.Args)
	if err != nil {
		fmt.Println("Erreur :", err)
	}
}
