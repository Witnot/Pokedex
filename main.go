package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"github.com/Witnot/Pokedex/internal/pokecache"
	"io"  
	"math/rand" 
	"time"  
)
const pokedexFile = "pokedex.json"

// Save userPokedex to file
func savePokedex() error {
	file, err := os.Create(pokedexFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(userPokedex)
}

// Load userPokedex from file
func loadPokedex() error {
	file, err := os.Open(pokedexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no file yet, ok
		}
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&userPokedex)
}

type config struct {
	Next     *string
	Previous *string
}

type cliCommand struct {
	name        string
	description string
	callback    func(cfg *config, args []string) error
}

type locationAreaDetail struct {
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}
type Pokemon struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	BaseExperience int    `json:"base_experience"`
	Height         int    `json:"height"`
	Weight         int    `json:"weight"`
	IsDefault      bool   `json:"is_default"`
	Order          int    `json:"order"`

	Stats []Stat `json:"-"`
	Types []Type `json:"-"`
}

type Stat struct {
	Name     string
	BaseStat int
}

type Type struct {
	Name string
}




// User's Pokedex
var userPokedex = make(map[string]Pokemon)


func getCachedJSON(url string, cache *pokecache.Cache, target interface{}) error {
	// Check cache first
	if data, ok := cache.Get(url); ok {
		return json.Unmarshal(data, target)
	}

	// Not cached, make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("network error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading body: %v", err)
	}

	// Add to cache
	cache.Add(url, body)

	// Unmarshal
	return json.Unmarshal(body, target)
}

func cleanInput(text string) []string {
	text = strings.TrimSpace(text)
	text = strings.ToLower(text)
	return strings.Fields(text)
}

func commandExit(cfg *config) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil // unreachable
}

func commandHelp(cfg *config) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	fmt.Println("help: Displays a help message")
	fmt.Println("exit: Exit the Pokedex")
	fmt.Println("map: Display the next 20 location areas")
	fmt.Println("mapb: Display the previous 20 location areas")
	return nil
}

// API response for location-area list
type locationAreaResponse struct {
	Count    int `json:"count"`
	Next     *string
	Previous *string
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

func commandCatch(cfg *config, cache *pokecache.Cache, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: catch <pokemon_name>")
		return nil
	}

	pokemonName := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%s/", pokemonName)

	// Intermediate struct matching API response
	var raw struct {
		ID             int `json:"id"`
		Name           string `json:"name"`
		BaseExperience int `json:"base_experience"`
		Height         int `json:"height"`
		Weight         int `json:"weight"`
		Order          int `json:"order"`
		IsDefault      bool `json:"is_default"`
		Stats []struct {
			BaseStat int `json:"base_stat"`
			Stat     struct {
				Name string `json:"name"`
			} `json:"stat"`
		} `json:"stats"`
		Types []struct {
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
		} `json:"types"`
	}

	if err := getCachedJSON(url, cache, &raw); err != nil {
		return err
	}

	fmt.Printf("Throwing a Pokeball at %s...\n", raw.Name)

	// Seed the random generator
	rand.Seed(time.Now().UnixNano())

	// Determine catch chance: e.g., 255 - base_experience
	catchChance := 255 - raw.BaseExperience
	if catchChance < 10 {
		catchChance = 10
	}

	if rand.Intn(255) < catchChance {
		fmt.Printf("You caught %s!\n", raw.Name)

		p := Pokemon{
			ID:             raw.ID,
			Name:           raw.Name,
			BaseExperience: raw.BaseExperience,
			Height:         raw.Height,
			Weight:         raw.Weight,
			Order:          raw.Order,
			IsDefault:      raw.IsDefault,
		}

		// Populate stats
		for _, s := range raw.Stats {
			p.Stats = append(p.Stats, Stat{
				Name:     s.Stat.Name,
				BaseStat: s.BaseStat,
			})
		}

		// Populate types
		for _, t := range raw.Types {
			p.Types = append(p.Types, Type{
				Name: t.Type.Name,
			})
		}

		userPokedex[p.Name] = p
		// Save pokedex after catching
		if err := savePokedex(); err != nil {
			fmt.Printf("Warning: failed to save pokedex: %v\n", err)
		}
	} else {
		fmt.Printf("%s escaped!\n", raw.Name)
	}


	return nil
}

func commandPokedex(cfg *config, args []string) error {
	if len(userPokedex) == 0 {
		fmt.Println("You have not caught any Pokémon yet.")
		return nil
	}

	fmt.Println("Your Pokémon:")
	for name := range userPokedex {
		fmt.Println("-", name)
	}

	return nil
}

func commandInspect(cfg *config, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: inspect <pokemon_name>")
		return nil
	}

	name := args[0]
	p, ok := userPokedex[name]
	if !ok {
		fmt.Printf("You have not caught %s yet.\n", name)
		return nil
	}

	fmt.Printf("Name: %s\n", p.Name)
	fmt.Printf("Height: %d\n", p.Height)
	fmt.Printf("Weight: %d\n", p.Weight)

	// Display stats with their base values
	if len(p.Stats) > 0 {
		fmt.Println("Stats:")
		for _, s := range p.Stats {
			fmt.Printf("  %s: %d\n", s.Name, s.BaseStat)
		}
	}

	// Display types
	if len(p.Types) > 0 {
		var typeNames []string
		for _, t := range p.Types {
			typeNames = append(typeNames, t.Name)
		}
		fmt.Printf("Types: %s\n", strings.Join(typeNames, ", "))
	}

	return nil
}




func commandMap(cfg *config, cache *pokecache.Cache) error {
	url := "https://pokeapi.co/api/v2/location-area/"
	if cfg.Next != nil {
		url = *cfg.Next
	}

	var data locationAreaResponse
	if err := getCachedJSON(url, cache, &data); err != nil {
		return err
	}

	for _, loc := range data.Results {
		fmt.Println(loc.Name)
	}

	cfg.Next = data.Next
	cfg.Previous = data.Previous

	return nil
}


func commandMapb(cfg *config, cache *pokecache.Cache) error {
	if cfg.Previous == nil {
		fmt.Println("No previous page available.")
		return nil
	}

	url := *cfg.Previous

	var data locationAreaResponse
	if err := getCachedJSON(url, cache, &data); err != nil {
		return err
	}

	for _, loc := range data.Results {
		fmt.Println(loc.Name)
	}

	cfg.Next = data.Next
	cfg.Previous = data.Previous

	return nil
}

func commandExplore(cfg *config, cache *pokecache.Cache, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: explore <area_name>")
		return nil
	}

	areaName := args[0]
	url := fmt.Sprintf("https://pokeapi.co/api/v2/location-area/%s/", areaName)

	var data locationAreaDetail
	if err := getCachedJSON(url, cache, &data); err != nil {
		return err
	}

	if len(data.PokemonEncounters) == 0 {
		fmt.Printf("No Pokémon found in %s\n", areaName)
		return nil
	}

	fmt.Printf("Pokémon in %s:\n", areaName)
	for _, encounter := range data.PokemonEncounters {
		fmt.Println(encounter.Pokemon.Name)
	}

	return nil
}


func main() {
	cfg := &config{}
	cache := pokecache.NewCache(5 * time.Minute)
	// Load saved pokedex
	if err := loadPokedex(); err != nil {
		fmt.Printf("Warning: could not load pokedex: %v\n", err)
	}
	commands := map[string]cliCommand{
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback: func(cfg *config, args []string) error {
				return commandExit(cfg)
			},
		},
		"catch": {
			name:        "catch",
			description: "Catch a Pokémon and add it to your Pokedex",
			callback: func(cfg *config, args []string) error {
				return commandCatch(cfg, cache, args)
			},
		},
		"inspect": {
			name:        "inspect",
			description: "Inspect a Pokémon you have caught",
			callback: func(cfg *config, args []string) error {
				return commandInspect(cfg, args)
			},
		},
		"pokedex": {
			name:        "pokedex",
			description: "List all Pokémon you have caught",
			callback: func(cfg *config, args []string) error {
				return commandPokedex(cfg, args)
			},
		},

		"help": {
			name:        "help",
			description: "Displays a help message",
			callback: func(cfg *config, args []string) error {
				return commandHelp(cfg)
			},
		},
		"map": {
			name:        "map",
			description: "Display the next 20 location areas",
			callback: func(cfg *config, args []string) error {
				return commandMap(cfg, cache)
			},
		},
		"mapb": {
			name:        "mapb",
			description: "Display the previous 20 location areas",
			callback: func(cfg *config, args []string) error {
				return commandMapb(cfg, cache)
			},
		},
		"explore": {
			name:        "explore",
			description: "Explore a location area and list Pokémon",
			callback: func(cfg *config, args []string) error {
				return commandExplore(cfg, cache, args)
			},
		},
	}


	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("Pokedex > ")

		if !scanner.Scan() {
			break
		}

		words := cleanInput(scanner.Text())
		if len(words) == 0 {
			continue
		}

		cmdName := words[0]
		cmd, ok := commands[cmdName]
		if !ok {
			fmt.Println("Unknown command")
			continue
		}

		args := words[1:] // everything after the command
		if err := cmd.callback(cfg, args); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

