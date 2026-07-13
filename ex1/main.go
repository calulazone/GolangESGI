package main

import (
	"fmt"
	"os"
)

const MAX_DISPLAY = 10

func display(args []string) error {
	if len(args) <= 1 {
		return fmt.Errorf("pas d'arguments fournis")
	}

	var nbTotalWord int
	var nbWordLen []string

	for i := 1; i < len(args); i++ {
		nbTotalWord++

		if len(args[i]) > 4 {
			nbWordLen = append(nbWordLen, args[i])
		}
	}

	if nbTotalWord > MAX_DISPLAY {
		fmt.Printf("Nombre total de mots : %d\n", MAX_DISPLAY)
	} else {
		fmt.Printf("Nombre total de mots : %d\n", nbTotalWord)
	}

	fmt.Printf("Mots de longueur > 4 : %v\n", nbWordLen)

	return nil
}

func main() {
	err := display(os.Args)
	if err != nil {
		fmt.Println("Erreur :", err)
	}
}
