// Package assets exposes embedded static files (token images and themes).
package assets

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed tokens themes
var FS embed.FS

// Theme holds the color palette for a token board.
type Theme struct {
	Name         string `json:"name"`
	HeaderBg     string `json:"headerBg"`
	HeaderText   string `json:"headerText"`
	NameBg       string `json:"nameBg"`
	NameText     string `json:"nameText"`
	TokenBg      string `json:"tokenBg"`
	TokenBorder  string `json:"tokenBorder"`
	TokenFill    string `json:"tokenFill"`
	FooterBg     string `json:"footerBg"`
	FooterBorder string `json:"footerBorder"`
}

// LoadTheme reads and parses a theme by name from the embedded themes directory.
func LoadTheme(name string) (Theme, error) {
	data, err := FS.ReadFile("themes/" + name + ".json")
	if err != nil {
		return Theme{}, fmt.Errorf("loading theme %q: %w", name, err)
	}
	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return Theme{}, fmt.Errorf("parsing theme %q: %w", name, err)
	}
	return t, nil
}

// TokenPNG returns the raw bytes of an embedded token PNG by name (e.g. "star").
func TokenPNG(name string) ([]byte, error) {
	data, err := FS.ReadFile("tokens/" + name + ".png")
	if err != nil {
		return nil, fmt.Errorf("loading token PNG %q: %w", name, err)
	}
	return data, nil
}
