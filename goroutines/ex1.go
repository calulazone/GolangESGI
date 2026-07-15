package main

import (
	"fmt"
	"sync"
	"time"
)

var LETTRES = []string{"a", "b", "c", "d", "e"}
var MAX_CHIFFRES = 5

func afficherLettres(wg *sync.WaitGroup) {
	defer wg.Done()
	for _, lettre := range LETTRES {
		fmt.Println(lettre)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres(wg *sync.WaitGroup) {
	defer wg.Done()
	for chiffre := 1; chiffre <= MAX_CHIFFRES; chiffre++ {
		fmt.Println(chiffre)
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go afficherLettres(&wg)
	go afficherChiffres(&wg)
	wg.Wait()
}
