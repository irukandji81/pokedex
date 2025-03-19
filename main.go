package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type cliCommand struct {
	name        string
	description string
	callback    func(*config) error
}

type config struct {
	Next     string
	Previous string
	Commands map[string]cliCommand
}

func main() {
	cfg := &config{
		Commands: make(map[string]cliCommand),
	}

	cfg.Commands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    commandExit,
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback: func(c *config) error {
				return commandHelp(c)
			},
		},
		"map": {
			name:        "map",
			description: "Display the next 20 locations",
			callback:    commandMap,
		},
		"mapb": {
			name:        "mapb",
			description: "Display the previous 20 locations",
			callback:    commandMapBack,
		},
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Pokedex > ")

		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		words := cleanInput(input)

		if len(words) == 0 {
			continue
		}

		command := words[0]

		if cmd, found := cfg.Commands[command]; found {
			if err := cmd.callback(cfg); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		} else {
			fmt.Println("Unknown command")
		}
	}
}

func cleanInput(text string) []string {
	trimmed := strings.TrimSpace(text)
	lowercased := strings.ToLower(trimmed)
	words := strings.Fields(lowercased)
	return words
}

func commandExit(cfg *config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	for _, cmd := range cfg.Commands {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func commandMap(cfg *config) error {
	if cfg.Next == "" {
		cfg.Next = "https://pokeapi.co/api/v2/location-area?limit=20"
	}

	response, err := http.Get(cfg.Next)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var data struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
	}

	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	for _, location := range data.Results {
		fmt.Println(location.Name)
	}

	cfg.Next = data.Next
	cfg.Previous = data.Previous
	return nil
}

func commandMapBack(cfg *config) error {
	if cfg.Previous == "" {
		fmt.Println("you're on the first page")
		return nil
	}

	response, err := http.Get(cfg.Previous)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	var data struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
	}

	if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	for _, location := range data.Results {
		fmt.Println(location.Name)
	}

	cfg.Next = data.Next
	cfg.Previous = data.Previous
	return nil
}
