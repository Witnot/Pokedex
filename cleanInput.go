package main

import (
	"strings"
)



// Version 2: real implementation
func cleanInputWord(text string) []string {
	// Trim leading/trailing spaces
	text = strings.TrimSpace(text)
	// Lowercase
	text = strings.ToLower(text)
	// Split by whitespace
	words := strings.Fields(text)
	return words
}
