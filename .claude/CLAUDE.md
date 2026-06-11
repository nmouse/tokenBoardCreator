# tokenBoardCreator — Project Guide

## What this is
A Go CLI tool that generates printable ABA therapy token boards as PDFs. Token boards are used in applied behavior analysis to visually track earned tokens toward a reward.

## Module
`github.com/owner/tokenBoardCreator`

## Key commands
```bash
# Build
go build .

# CLI usage
./tokenBoardCreator --name "Alex" --tokens 5 --token-style star --reward "iPad time" --output ./board.pdf

# Web mode
./tokenBoardCreator --web --port 8080

# Tests
go test ./... -race

# Vet
go vet ./...

# Regenerate embedded token PNGs (only needed if you change cmd/gentokens/main.go)
go run ./cmd/gentokens/
```

## Architecture
- `internal/board/` — Config struct, Validate(), layout constants
- `internal/assets/` — embed.go with //go:embed for tokens/ and themes/; LoadTheme(); TokenPNG()
- `internal/render/pdf.go` — PDF generation via fpdf; drawToken() dispatcher
- `internal/render/html.go` — Web server, HTML form, CSS preview, PDF streaming
- `main.go` — flag parsing, CLI/web dispatch

## Important constraints
- **PDF coordinates always in mm** — never mix mm and px
- No external deps beyond `github.com/go-pdf/fpdf`
- Token slots are evenly spaced with a 4mm gap between them
- Layout fractions: header 25%, name band 10%, token row 40%, footer 25%
- Valid token counts: 3–10

## Token styles
- Builtin (fpdf primitives): `star`, `circle`, `smiley`, `thumbsup`
- Embedded PNG assets: `png:star`, `png:smiley`, `png:thumbsup`
- Disk path: any other value is treated as a file path

## Themes
JSON files in `internal/assets/themes/`: `default`, `blue`, `green`, `pink`
Each theme defines: headerBg, headerText, nameBg, nameText, tokenBg, tokenBorder, tokenFill, footerBg, footerBorder (all CSS hex colors)

## Style rules
- Wrap errors: `fmt.Errorf("doing X: %w", err)`
- Return early on error
- `context.Context` first param on blocking/I/O functions
- Table-driven tests with `t.Run`
- Every exported symbol needs a doc comment starting with its name
