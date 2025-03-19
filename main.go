package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/irukandji81/pokedex/internal/pokecache"
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
	cache := pokecache.NewCache(5 * time.Second)

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
			callback: func(c *config) error {
				return commandMapWithCache(c, cache)
			},
		},
		"mapb": {
			name:        "mapb",
			description: "Display the previous 20 locations",
			callback: func(c *config) error {
				return commandMapBackWithCache(c, cache)
			},
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

func commandMapWithCache(cfg *config, cache *pokecache.Cache) error {
	if cfg.Next == "" {
		cfg.Next = "https://pokeapi.co/api/v2/location-area?limit=20"
	}

	if val, found := cache.Get(cfg.Next); found {
		fmt.Println("Using cached data")
		var data struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
			Next     string `json:"next"`
			Previous string `json:"previous"`
		}
		if err := json.Unmarshal(val, &data); err != nil {
			return fmt.Errorf("failed to decode cached response: %v", err)
		}
		for _, location := range data.Results {
			fmt.Println(location.Name)
		}
		cfg.Next = data.Next
		cfg.Previous = data.Previous
		return nil
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

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	cache.Add(cfg.Next, body)

	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	for _, location := range data.Results {
		fmt.Println(location.Name)
	}

	cfg.Next = data.Next
	cfg.Previous = data.Previous
	return nil
}

func commandMapBackWithCache(cfg *config, cache *pokecache.Cache) error {
	if cfg.Previous == "" {
		fmt.Println("you're on the first page")
		return nil
	}

	if val, found := cache.Get(cfg.Previous); found {
		fmt.Println("Using cached data")
		var data struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
			Next     string `json:"next"`
			Previous string `json:"previous"`
		}
		if err := json.Unmarshal(val, &data); err != nil {
			return fmt.Errorf("failed to decode cached response: %v", err)
		}
		for _, location := range data.Results {
			fmt.Println(location.Name)
		}
		cfg.Next = data.Next
		cfg.Previous = data.Previous
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

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	cache.Add(cfg.Previous, body)

	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	for _, location := range data.Results {
		fmt.Println(location.Name)
	}

	cfg.Next = data.Next
	cfg.Previous = data.Previous
	return nil
}
