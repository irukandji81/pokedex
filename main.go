package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/irukandji81/pokedex/internal/pokecache"

	"github.com/chzyer/readline"
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
	Current  string
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

	const logDirectory = "logs/"
	const pokedexFile = logDirectory + "pokedex.json"
	const historyFile = logDirectory + ".pokedex_history"

	if err := loadPokedexFromFile(cfg, pokedexFile); err != nil {
		fmt.Printf("Error loading Pokedex: %v\n", err)
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
			description: "Interactively explore locations by navigating left or right",
			callback: func(c *config, args []string) error {
				return commandExploreInteractive(c, args, cache)
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
		"pokedex": {
			name:        "pokedex",
			description: "List all the Pokémon you have caught",
			callback: func(c *config, args []string) error {
				return commandPokedex(c, args)
			},
		},
	}

	// Set up readline for command history
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "Pokedex > ",
		HistoryFile:     historyFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		return
	}
	defer rl.Close()

	//REPL Loop
	for {
		fmt.Print("Pokedex > ")

		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			}
			continue
		} else if err == io.EOF {
			break
		}

		words := cleanInput(line)

		if len(words) == 0 {
			continue
		}

		command := words[0]
		args := words[1:]

		if cmd, found := cfg.Commands[command]; found {
			if err := cmd.callback(cfg, args); err != nil {
				// Handle the "exit" signal
				if err.Error() == "exit" {
					break
				}
				fmt.Printf("Error: %v\n", err)
			}
		} else {
			fmt.Println("Unknown command")
		}
	}

	// Save the Pokedex to file when exiting the program
	if err := savePokedexToFile(cfg, pokedexFile); err != nil {
		fmt.Printf("Error saving Pokedex: %v\n", err)
	} else {
		fmt.Println("Pokedex saved successfully!")
	}

	// Clear the history file
	file, err := os.OpenFile(historyFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Error clearing command history: %v\n", err)
	} else {
		file.Close()
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
	return fmt.Errorf("exit") // A special error to indicate program termination
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

func commandExploreInteractive(cfg *config, args []string, cache *pokecache.Cache) error {
	if cfg.Current == "" {
		cfg.Current = "https://pokeapi.co/api/v2/location-area?limit=20"
	}

	// Fetch data for the current location
	if val, found := cache.Get(cfg.Current); found {
		fmt.Println("Using cached data")
		return displayNavigationChoices(cfg, val, cache)
	}

	response, err := http.Get(cfg.Current)
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

	cache.Add(cfg.Current, body)
	return displayNavigationChoices(cfg, body, cache)
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

func commandPokedex(cfg *config, args []string) error {
	if len(cfg.Pokedex) == 0 {
		fmt.Println("Your Pokedex is empty. Go catch some Pokémon!")
		return nil
	}

	fmt.Println("Your Pokedex:")
	for pokemon := range cfg.Pokedex {
		fmt.Printf(" - %s\n", pokemon)
	}
	return nil
}

func savePokedexToFile(cfg *config, filename string) error {
	// Convert the Pokedex map to pretty-printed JSON
	prettyJSON, err := json.MarshalIndent(cfg.Pokedex, "", "    ")
	if err != nil {
		fmt.Printf("Error formatting Pokedex: %v\n", err)
		return fmt.Errorf("failed to format Pokedex: %v", err)
	}

	// Create or overwrite the file
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write the pretty-printed JSON to the file
	_, err = file.Write(prettyJSON)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return fmt.Errorf("failed to write to file: %v", err)
	}
	return nil
}

func loadPokedexFromFile(cfg *config, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No saved Pokedex found. Starting fresh!")
			return nil
		}
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg.Pokedex); err != nil {
		return fmt.Errorf("failed to decode Pokedex: %v", err)
	}

	fmt.Println("Pokedex loaded successfully!")
	return nil
}

func displayNavigationChoices(cfg *config, data []byte, cache *pokecache.Cache) error {
	var result struct {
		Results []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"results"`
		Next     string `json:"next"`
		Previous string `json:"previous"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	cfg.Next = result.Next
	cfg.Previous = result.Previous

	fmt.Println("Where would you like to go next?")
	if cfg.Previous != "" {
		fmt.Println(" - Type 'left' to go to the previous location.")
	}
	if cfg.Next != "" {
		fmt.Println(" - Type 'right' to go to the next location.")
	}

	var choice string
	fmt.Print("> ")
	fmt.Scanln(&choice)

	if choice == "left" && cfg.Previous != "" {
		cfg.Current = cfg.Previous
	} else if choice == "right" && cfg.Next != "" {
		cfg.Current = cfg.Next
	} else {
		fmt.Println("Invalid choice. Please type 'left' or 'right'.")
	}
	return nil
}
