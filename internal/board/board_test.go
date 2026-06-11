package board

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string // substring; empty means no error
	}{
		{
			name: "valid minimal config",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
				TokenStyle: "star",
			},
		},
		{
			name: "defaults applied",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
			},
			wantErr: "",
		},
		{
			name: "token count too low",
			cfg: Config{
				TokenCount: 2,
				RewardText: "Sticker",
				TokenStyle: "star",
			},
			wantErr: "token count must be between",
		},
		{
			name: "token count too high",
			cfg: Config{
				TokenCount: 11,
				RewardText: "Sticker",
				TokenStyle: "star",
			},
			wantErr: "token count must be between",
		},
		{
			name: "token count at minimum",
			cfg: Config{
				TokenCount: MinTokens,
				RewardText: "Sticker",
				TokenStyle: "star",
			},
		},
		{
			name: "token count at maximum",
			cfg: Config{
				TokenCount: MaxTokens,
				RewardText: "Sticker",
				TokenStyle: "star",
			},
		},
		{
			name: "missing reward",
			cfg: Config{
				TokenCount: 5,
				TokenStyle: "star",
			},
			wantErr: "at least one of",
		},
		{
			name: "reward via image path",
			cfg: Config{
				TokenCount:  5,
				RewardImage: "/some/image.png",
				TokenStyle:  "star",
			},
		},
		{
			name: "unknown theme",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
				TokenStyle: "star",
				Theme:      "rainbow",
			},
			wantErr: "unknown theme",
		},
		{
			name: "unknown page size",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
				TokenStyle: "star",
				PageSize:   "a3",
			},
			wantErr: "unknown page size",
		},
		{
			name: "png: style with name",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
				TokenStyle: "png:star",
			},
		},
		{
			name: "png: style without name",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
				TokenStyle: "png:",
			},
			wantErr: "png: token style requires a name",
		},
		{
			name: "circle builtin style",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
				TokenStyle: "circle",
			},
		},
		{
			name: "file path style treated as disk path",
			cfg: Config{
				TokenCount: 5,
				RewardText: "Sticker",
				TokenStyle: "/tmp/custom_token.png",
			},
		},
		{
			name: "multiple errors",
			cfg: Config{
				TokenCount: 0,
			},
			wantErr: "token count must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateAppliesDefaults(t *testing.T) {
	cfg := Config{
		TokenCount: 5,
		RewardText: "Sticker",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Theme != "default" {
		t.Errorf("expected default theme, got %q", cfg.Theme)
	}
	if cfg.PageSize != "letter" {
		t.Errorf("expected letter page size, got %q", cfg.PageSize)
	}
	if cfg.Title != "I am working for:" {
		t.Errorf("expected default title, got %q", cfg.Title)
	}
	if cfg.Output != "./tokenboard.pdf" {
		t.Errorf("expected default output path, got %q", cfg.Output)
	}
	if cfg.TokenStyle != "star" {
		t.Errorf("expected default token style, got %q", cfg.TokenStyle)
	}
	if cfg.WebPort != 8080 {
		t.Errorf("expected default web port 8080, got %d", cfg.WebPort)
	}
}

func TestIsBuiltinStyle(t *testing.T) {
	tests := []struct {
		style string
		want  bool
	}{
		{"star", true},
		{"circle", true},
		{"smiley", true},
		{"thumbsup", true},
		{"png:star", false},
		{"/tmp/foo.png", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			if got := IsBuiltinStyle(tt.style); got != tt.want {
				t.Errorf("IsBuiltinStyle(%q) = %v, want %v", tt.style, got, tt.want)
			}
		})
	}
}

func TestIsPNGAssetStyle(t *testing.T) {
	tests := []struct {
		style string
		want  bool
	}{
		{"png:star", true},
		{"png:smiley", true},
		{"star", false},
		{"/tmp/star.png", false},
	}
	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			if got := IsPNGAssetStyle(tt.style); got != tt.want {
				t.Errorf("IsPNGAssetStyle(%q) = %v, want %v", tt.style, got, tt.want)
			}
		})
	}
}

func TestPNGAssetName(t *testing.T) {
	if got := PNGAssetName("png:star"); got != "star" {
		t.Errorf("PNGAssetName(%q) = %q, want %q", "png:star", got, "star")
	}
}
