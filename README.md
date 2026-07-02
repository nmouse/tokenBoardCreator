# tokenBoardCreator

A Go CLI tool that generates printable ABA therapy token boards as PDFs.

Token boards are used in applied behavior analysis (ABA) to give children a visual way to track earned tokens toward a reward. This tool generates a ready-to-print PDF with a customizable header, name band, token row, and reward footer — with an optional AI-generated background image.

![Example token board](qa_2.5_happy_path.png)

## Features

- Generates a single-page PDF token board (letter or A4)
- 3–10 token slots with multiple token styles
- Four built-in color themes
- Custom reward text and/or reward image in the footer
- AI-generated background art via Hugging Face FLUX.1-schnell (free, no paid API key needed)
- Browser-based web UI (`--web`) for point-and-click creation

## Installation

```bash
git clone https://github.com/owner/tokenBoardCreator
cd tokenBoardCreator
go build .
```

Requires Go 1.24+. The only runtime dependency is [`github.com/go-pdf/fpdf`](https://github.com/go-pdf/fpdf).

### Prebuilt binaries

Prebuilt binaries for macOS, Windows, and Linux are attached to each [GitHub Release](https://github.com/nmouse/tokenBoardCreator/releases). On macOS, download the `.zip` matching your Mac's chip — `tokenBoardCreator-darwin-arm64.zip` for Apple Silicon (M1/M2/M3/M4), `tokenBoardCreator-darwin-amd64.zip` for Intel — then unzip it.

The binary isn't code-signed, so the first time you run it macOS Gatekeeper will block it ("cannot be opened because the developer cannot be verified", or "is damaged and can't be opened"). To allow it:

```bash
xattr -d com.apple.quarantine ./tokenBoardCreator-darwin-arm64   # use -amd64 on Intel Macs
./tokenBoardCreator-darwin-arm64 --web
```

Or in Finder: right-click the binary → **Open** → **Open** again in the dialog that appears.

## CLI usage

```bash
./tokenBoardCreator --name "Alex" --tokens 5 --reward "iPad time" --output ./board.pdf
```

### All flags

| Flag | Default | Description |
|---|---|---|
| `--name` | _(blank)_ | Child's name, printed in the name band |
| `--tokens` | `5` | Number of token slots (3–10) |
| `--token-style` | `star` | Token appearance (see below) |
| `--reward` | _(required*)_ | Reward text shown in the footer |
| `--reward-image` | _(blank)_ | Path to a PNG/JPG image shown in the footer |
| `--theme` | `default` | Color theme: `default`, `blue`, `green`, `pink` |
| `--title` | `"I am working for:"` | Header title text |
| `--page-size` | `letter` | Page size: `letter` or `a4` |
| `--background-prompt` | _(blank)_ | Scene description for AI-generated background art |
| `--output` | `./tokenboard.pdf` | Output PDF path |
| `--web` | `false` | Start the web UI instead of generating a PDF |
| `--port` | `8080` | Port for the web UI |

*At least one of `--reward` or `--reward-image` is required.

### Token styles

| Value | Description |
|---|---|
| `star` | Drawn star (default) |
| `circle` | Drawn circle |
| `smiley` | Drawn smiley face |
| `thumbsup` | Drawn thumbs-up |
| `png:star` | Embedded PNG star |
| `png:smiley` | Embedded PNG smiley |
| `png:thumbsup` | Embedded PNG thumbs-up |
| `path/to/image.png` | Any PNG or JPG file on disk |

### AI-generated backgrounds

Use `--background-prompt` with a scene description to generate a background image with the [FLUX.1-schnell](https://huggingface.co/black-forest-labs/FLUX.1-schnell) model. A free Hugging Face token is required.

```bash
export HF_TOKEN=hf_...
./tokenBoardCreator --name "Sam" --tokens 6 --reward "movie night" \
  --background-prompt "friendly dinosaurs in a sunny jungle" --output ./board.pdf
```

Get a free token at <https://huggingface.co/settings/tokens> (read access is sufficient). Generation takes roughly 10–30 seconds.

## Web UI

```bash
./tokenBoardCreator --web --port 8080
```

Open <http://localhost:8080> in a browser. Fill in the form and click **Generate PDF** to download the board directly. All CLI options are available through the form, including image uploads for custom tokens and reward images.

## Themes

| Name | Description |
|---|---|
| `default` | Warm yellow/orange |
| `blue` | Cool blue tones |
| `green` | Calm green tones |
| `pink` | Soft pink tones |

## Development

```bash
# Run tests
go test ./... -race

# Vet
go vet ./...

# Regenerate embedded token PNGs (only needed if cmd/gentokens/main.go changes)
go run ./cmd/gentokens/
```

### Project layout

```
main.go                    # Flag parsing, CLI/web dispatch
internal/
  board/                   # Config struct, Validate(), layout constants
  assets/                  # Embedded PNGs and theme JSON files
  render/
    pdf.go                 # PDF generation (fpdf)
    html.go                # Web server and HTML form
  imagegen/                # Hugging Face image generation client
cmd/
  gentokens/               # Generator for embedded token PNG assets
```

## License

MIT
