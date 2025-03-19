package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Create a new scanner to read user input
	scanner := bufio.NewScanner(os.Stdin)

	// Start the infinite loop for the REPL
	for {
		// Print the prompt without a newline
		fmt.Print("Pokedex > ")

		// Wait for user input
		if !scanner.Scan() {
			break // Exit loop if scanner cannot scan (e.g., EOF)
		}

		// Get the input as a string
		input := scanner.Text()

		// Clean the input using the cleanInput function
		words := cleanInput(input)

		// If there are no words, continue to the next loop iteration
		if len(words) == 0 {
			continue
		}

		// Capture the first word and print it
		fmt.Printf("Your command was: %s\n", words[0])
	}
}

func cleanInput(text string) []string {
	trimmed := strings.TrimSpace(text)     // Trim leading and trailing whitespace
	lowercased := strings.ToLower(trimmed) // Convert to lowercase
	words := strings.Fields(lowercased)    // Split into words based on whitespace
	return words
}
