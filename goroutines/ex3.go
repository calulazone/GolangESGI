package main

import (
	"fmt"
)

func main() {
	// Créer un slice de 1000 entiers (1 à 1000)
	slice := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		slice[i] = i + 1
	}

	// Créer un channel pour récupérer les résultats partiels
	resultat := make(chan int)

	// Nombre de goroutines et taille de chaque morceau
	nbWorkers := 4
	taille := len(slice) / nbWorkers

	// Lancer les goroutines dans une boucle
	for i := 0; i < nbWorkers; i++ {
		start := i * taille
		end := (i + 1) * taille
		if i == nbWorkers-1 {
			end = len(slice)
		}

		go func(num, start, end int) {
			sum := 0
			for j := start; j < end; j++ {
				sum += slice[j]
			}
			fmt.Printf("Résultat partiel %d : %d\n", num, sum)
			resultat <- sum
		}(i+1, start, end)
	}

	// Recevoir les résultats partiels et les additionner
	total := 0
	for i := 0; i < nbWorkers; i++ {
		partiel := <-resultat
		total += partiel
	}

	fmt.Printf("\nSomme totale : %d\n", total)
}
