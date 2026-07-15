package main

import (
	"fmt"
	"sync"
)

func main() {
	// Créer les channels
	jobs := make(chan int)
	resultats := make(chan int)
	nbWorkers := 4

	var wg sync.WaitGroup
	wg.Add(nbWorkers)

	// Lancer les workers
	for i := 1; i <= nbWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			// Chaque worker lit les jobs jusqu'à fermeture du channel
			for job := range jobs {
				result := job * job
				fmt.Printf("Worker %d: %d² = %d\n", workerID, job, result)
				resultats <- result
			}
		}(i)
	}

	// Envoyer les 20 jobs
	go func() {
		for i := 1; i <= 20; i++ {
			jobs <- i
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(resultats)
	}()

	// Lire les résultats
	for result := range resultats {
		fmt.Printf("Résultat reçu: %d\n", result)
	}
}
