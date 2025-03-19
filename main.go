package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/irukandji81/pokedex/internal/pokecache"
)

type cliCommand struct {
	name        string
	description string
	callback    func(*config, []string) error
}

type config struct {
	Next     string
	Previous string
	Commands map[string]cliCommand
	Pokedex  map[string]pokemonData
}

type pokemonData struct {
	Name   string
	Height int
	Weight int
	Stats  map[string]int
	Types  []string
}

func main() {
	rand.Seed(time.Now().UnixNano())
	cache := pokecache.NewCache(5 * time.Second)

	cfg := &config{
		Commands: make(map[string]cliCommand),
		Pokedex:  make(map[string]pokemonData),
	}

	cfg.Commands = map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback: func(c *config, args []string) error {
				return commandExit(c)
			},
		},
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback: func(c *config, args []string) error {
				return commandHelp(c)
			},
		},
		"map": {
			name:        "map",
			description: "Display the next 20 locations",
			callback: func(c *config, args []string) error {
				return commandMapWithCache(c, cache)
			},
		},
		"mapb": {
			name:        "mapb",
			description: "Display the previous 20 locations",
			callback: func(c *config, args []string) error {
				return commandMapBackWithCache(c, cache)
			},
		},
		"explore": {
			name:        "explore",
			description: "Explore a location area and list Pokémon",
			callback: func(c *config, args []string) error {
				return commandExploreWithCache(c, args, cache)
			},
		},
		"catch": {
			name:        "catch",
			description: "Attempt to catch a Pokémon by name",
			callback: func(c *config, args []string) error {
				return commandCatchWithCache(c, args, cache)
			},
		},
		"inspect": {
			name:        "inspect",
			description: "Inspect a caught Pokémon's details",
			callback: func(c *config, args []string) error {
				return commandInspect(c, args)
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
		args := words[1:]

		if cmd, found := cfg.Commands[command]; found {
			if err := cmd.callback(cfg, args); err != nil {
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

func commandExploreWithCache(cfg *config, args []string, cache *pokecache.Cache) error {
	if len(args) < 1 {
		return fmt.Errorf("you must specify a location area to explore")
	}

	locationArea := args[0]
	apiURL := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s", locationArea)

	if val, found := cache.Get(apiURL); found {
		fmt.Println("Using cached data")
		return displayPokemonFromResponse(val)
	}

	response, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	cache.Add(apiURL, body)
	return displayPokemonFromResponse(body)
}

func displayPokemonFromResponse(data []byte) error {
	var result struct {
		PokemonEncounters []struct {
			Pokemon struct {
				Name string `json:"name"`
			} `json:"pokemon"`
		} `json:"pokemon_encounters"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if len(result.PokemonEncounters) == 0 {
		fmt.Println("No Pokémon found in this area!")
		return nil
	}

	fmt.Println("Found Pokémon:")
	for _, encounter := range result.PokemonEncounters {
		fmt.Printf(" - %s\n", encounter.Pokemon.Name)
	}
	return nil
}

func commandCatchWithCache(cfg *config, args []string, cache *pokecache.Cache) error {
	if len(args) < 1 {
		return fmt.Errorf("you must specify a Pokémon to catch")
	}

	pokemonName := strings.ToLower(args[0])
	apiURL := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s", pokemonName)

	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)

	if val, found := cache.Get(apiURL); found {
		return determineCatchResult(cfg, pokemonName, val)
	}

	response, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	cache.Add(apiURL, body)
	return determineCatchResult(cfg, pokemonName, body)
}

func determineCatchResult(cfg *config, pokemonName string, data []byte) error {
	var result struct {
		Name           string `json:"name"`
		Height         int    `json:"height"`
		Weight         int    `json:"weight"`
		BaseExperience int    `json:"base_experience"`
		Stats          []struct {
			Stat struct {
				Name string `json:"name"`
			} `json:"stat"`
			BaseStat int `json:"base_stat"`
		} `json:"stats"`
		Types []struct {
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
		} `json:"types"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse Pokémon data: %v", err)
	}

	chance := 100.0 / float64(result.BaseExperience)
	caught := rand.Float64() < chance

	if caught {
		stats := make(map[string]int)
		for _, stat := range result.Stats {
			stats[stat.Stat.Name] = stat.BaseStat
		}

		types := []string{}
		for _, t := range result.Types {
			types = append(types, t.Type.Name)
		}

		cfg.Pokedex[pokemonName] = pokemonData{
			Name:   result.Name,
			Height: result.Height,
			Weight: result.Weight,
			Stats:  stats,
			Types:  types,
		}

		fmt.Printf("%s was caught!\n", pokemonName)
	} else {
		fmt.Printf("%s escaped!\n", pokemonName)
	}
	return nil
}

func commandInspect(cfg *config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("you must specify a Pokémon to inspect")
	}

	pokemonName := strings.ToLower(args[0])
	if data, found := cfg.Pokedex[pokemonName]; found {
		fmt.Printf("Name: %s\n", data.Name)
		fmt.Printf("Height: %d\n", data.Height)
		fmt.Printf("Weight: %d\n", data.Weight)
		fmt.Println("Stats:")
		for stat, value := range data.Stats {
			fmt.Printf("  -%s: %d\n", stat, value)
		}
		fmt.Println("Types:")
		for _, t := range data.Types {
			fmt.Printf("  - %s\n", t)
		}
	} else {
		fmt.Println("you have not caught that Pokémon")
	}

	return nil
}
