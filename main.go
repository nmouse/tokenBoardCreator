// tokenBoardCreator generates printable ABA therapy token boards as PDFs.
// Run with --web to start a browser-based creation server.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/owner/tokenBoardCreator/internal/board"
	"github.com/owner/tokenBoardCreator/internal/render"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var cfg board.Config
	var web bool

	flag.StringVar(&cfg.ChildName, "name", "", "child's name (optional)")
	flag.IntVar(&cfg.TokenCount, "tokens", 5, "number of token slots (3–10)")
	flag.StringVar(&cfg.TokenStyle, "token-style", "star", "token style: star, circle, smiley, thumbsup, png:star, png:smiley, png:thumbsup, or a file path")
	flag.StringVar(&cfg.RewardText, "reward", "", "reward text (e.g. \"iPad time\")")
	flag.StringVar(&cfg.RewardImage, "reward-image", "", "optional path to reward image (PNG/JPG)")
	flag.StringVar(&cfg.Theme, "theme", "default", "color theme: default, blue, green, pink")
	flag.StringVar(&cfg.Title, "title", "", "header title (default: \"I am working for:\")")
	flag.StringVar(&cfg.Output, "output", "./tokenboard.pdf", "output PDF path")
	flag.StringVar(&cfg.PageSize, "page-size", "letter", "page size: letter or a4")
	flag.BoolVar(&web, "web", false, "start web server for browser-based creation")
	flag.IntVar(&cfg.WebPort, "port", 8080, "web server port (used with --web)")
	flag.Parse()

	if web {
		return render.WebServer(context.Background(), cfg.WebPort)
	}

	if err := cfg.Validate(); err != nil {
		flag.Usage()
		return fmt.Errorf("invalid options: %w", err)
	}

	if err := render.PDF(context.Background(), cfg); err != nil {
		return fmt.Errorf("generating PDF: %w", err)
	}

	fmt.Printf("Token board written to %s\n", cfg.Output)
	return nil
}
