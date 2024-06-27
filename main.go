package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	secretWord string
	dictionary []string
	wordLength int
	guesses    []string
	mutex      sync.Mutex
)

type GuessResponse struct {
	CloserTo string    `json:"closerTo"`
	Distance int       `json:"distance"`
	Guesses  []string  `json:"guesses"`
	Between  [2]string `json:"between"`
}

func loadDictionary() error {
	file, err := os.Open("/usr/share/dict/words")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if len(word) > 0 {
			dictionary = append(dictionary, strings.ToLower(word))
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	sort.Strings(dictionary)
	return nil
}

func setSecretWord(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	secretWord = strings.ToLower(requestBody["word"])
	wordLength = len(secretWord)
	guesses = []string{} // Reset guesses
	fmt.Fprintf(w, "Secret word set successfully!")
}

func makeGuess(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	var requestBody map[string]string
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	guess, ok := requestBody["word"]
	if !ok {
		http.Error(w, "Word not found in request", http.StatusBadRequest)
		return
	}

	guess = strings.ToLower(guess)
	guesses = append(guesses, guess)
	guessIndex := findWordIndex(guess)
	secretIndex := findWordIndex(secretWord)

	if guessIndex == -1 || secretIndex == -1 {
		http.Error(w, "Word not in dictionary", http.StatusBadRequest)
		return
	}

	distance := abs(guessIndex - secretIndex)
	closerTo := "top"
	if guessIndex < secretIndex {
		closerTo = "bottom"
	}

	between := findBetweenWords()

	response := GuessResponse{
		CloserTo: closerTo,
		Distance: (distance * 100) / len(dictionary),
		Guesses:  guesses,
		Between:  between,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func findWordIndex(word string) int {
	for i, w := range dictionary {
		if w == word {
			return i
		}
	}
	return -1
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func findBetweenWords() [2]string {
	sortedGuesses := make([]string, len(guesses))
	copy(sortedGuesses, guesses)
	sort.Strings(sortedGuesses)

	var lower, upper string
	for i, guess := range sortedGuesses {
		if guess == secretWord {
			if i > 0 {
				lower = sortedGuesses[i-1]
			} else {
				lower = "A"
			}
			if i < len(sortedGuesses)-1 {
				upper = sortedGuesses[i+1]
			} else {
				upper = "Z"
			}
			break
		}
		if guess > secretWord {
			if i > 0 {
				lower = sortedGuesses[i-1]
			} else {
				lower = "A"
			}
			upper = guess
			break
		}
		lower = guess
		upper = "Z"
	}
	return [2]string{lower, upper}
}

func serveFile(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.ServeFile(w, r, "index.html")
	} else {
		http.ServeFile(w, r, filepath.Join(".", r.URL.Path))
	}
}

func main() {
	err := loadDictionary()
	if err != nil {
		fmt.Printf("Error loading dictionary: %v\n", err)
		return
	}

	http.HandleFunc("/setSecretWord", setSecretWord)
	http.HandleFunc("/makeGuess", makeGuess)
	http.HandleFunc("/", serveFile)

	fmt.Println("Starting server on port 8080...")
	http.ListenAndServe(":8080", nil)
}
