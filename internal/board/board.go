// Package board defines the token board configuration and validation logic.
package board

import (
	"errors"
	"fmt"
	"strings"
)

// Config holds all parameters needed to generate a token board.
type Config struct {
	ChildName            string
	TokenCount           int
	TokenStyle           string
	RewardText           string
	RewardImage          string
	Theme                string
	Title                string
	Output               string
	PageSize             string
	WebPort              int
	BackgroundPrompt     string
	BackgroundImageBytes []byte
}

// Layout constants define the proportional zones of the page.
// All values are fractions of the page height.
const (
	HeaderFraction = 0.25
	NameFraction   = 0.10
	TokenFraction  = 0.40
	FooterFraction = 0.25

	// MinTokens and MaxTokens define the allowed token count range.
	MinTokens = 3
	MaxTokens = 10
)

// validThemes is the set of supported theme names.
var validThemes = map[string]bool{
	"default": true,
	"blue":    true,
	"green":   true,
	"pink":    true,
}

// validPageSizes is the set of supported page size names.
var validPageSizes = map[string]bool{
	"letter": true,
	"a4":     true,
}

// builtinTokenStyles are styles rendered with fpdf primitives (no image files).
var builtinTokenStyles = map[string]bool{
	"star":     true,
	"circle":   true,
	"smiley":   true,
	"thumbsup": true,
}

// Validate checks the Config for invalid or missing values and applies defaults.
func (c *Config) Validate() error {
	var errs []string

	if c.TokenCount < MinTokens || c.TokenCount > MaxTokens {
		errs = append(errs, fmt.Sprintf("token count must be between %d and %d, got %d", MinTokens, MaxTokens, c.TokenCount))
	}

	if c.RewardText == "" && c.RewardImage == "" {
		errs = append(errs, "at least one of --reward or --reward-image must be set")
	}

	if c.Theme == "" {
		c.Theme = "default"
	}
	if !validThemes[c.Theme] {
		errs = append(errs, fmt.Sprintf("unknown theme %q; valid themes: default, blue, green, pink", c.Theme))
	}

	if c.PageSize == "" {
		c.PageSize = "letter"
	}
	if !validPageSizes[c.PageSize] {
		errs = append(errs, fmt.Sprintf("unknown page size %q; valid sizes: letter, a4", c.PageSize))
	}

	if c.Title == "" {
		c.Title = "I am working for:"
	}

	if c.Output == "" {
		c.Output = "./tokenboard.pdf"
	}

	if c.TokenStyle == "" {
		c.TokenStyle = "star"
	}
	if err := validateTokenStyle(c.TokenStyle); err != nil {
		errs = append(errs, err.Error())
	}

	if c.WebPort == 0 {
		c.WebPort = 8080
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// validateTokenStyle returns an error if style is not a known builtin, png: prefix, or non-empty path.
func validateTokenStyle(style string) error {
	if builtinTokenStyles[style] {
		return nil
	}
	if strings.HasPrefix(style, "png:") {
		name := strings.TrimPrefix(style, "png:")
		if name == "" {
			return fmt.Errorf("png: token style requires a name after the colon")
		}
		return nil
	}
	// Treat anything else as a file path — existence is checked at render time.
	return nil
}

// IsBuiltinStyle reports whether the style uses fpdf primitives.
func IsBuiltinStyle(style string) bool {
	return builtinTokenStyles[style]
}

// IsPNGAssetStyle reports whether the style refers to an embedded PNG asset.
func IsPNGAssetStyle(style string) bool {
	return strings.HasPrefix(style, "png:")
}

// PNGAssetName returns the asset name for a png: style (e.g. "png:star" → "star").
func PNGAssetName(style string) string {
	return strings.TrimPrefix(style, "png:")
}
