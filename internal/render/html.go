package render

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/owner/tokenBoardCreator/internal/assets"

	"github.com/owner/tokenBoardCreator/internal/board"
)

const formHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Token Board Creator</title>
<style>
  body { font-family: Arial, sans-serif; max-width: 600px; margin: 40px auto; padding: 0 20px; background: #f9f9f9; }
  h1 { color: #333; }
  label { display: block; margin-top: 12px; font-weight: bold; color: #555; }
  input, select { width: 100%; padding: 8px; margin-top: 4px; box-sizing: border-box; border: 1px solid #ccc; border-radius: 4px; }
  .btn-row { margin-top: 20px; display: flex; gap: 10px; }
  button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
  .btn-preview { background: #1565C0; color: white; }
  .btn-download { background: #2E7D32; color: white; }
  button:hover { opacity: 0.85; }
</style>
</head>
<body>
<h1>Token Board Creator</h1>
<form method="POST" action="/preview">
  <label>Child Name (optional)
    <input type="text" name="name" placeholder="e.g. Alex" value="{{.Name}}">
  </label>
  <label>Reward Text
    <input type="text" name="reward" required placeholder="e.g. iPad time" value="{{.Reward}}">
  </label>
  <label>Number of Tokens (3–10)
    <input type="number" name="tokens" min="3" max="10" value="{{.Tokens}}" required>
  </label>
  <label>Token Style
    <select name="token_style">
      <option value="star"{{if eq .TokenStyle "star"}} selected{{end}}>Star</option>
      <option value="circle"{{if eq .TokenStyle "circle"}} selected{{end}}>Circle</option>
      <option value="smiley"{{if eq .TokenStyle "smiley"}} selected{{end}}>Smiley</option>
      <option value="thumbsup"{{if eq .TokenStyle "thumbsup"}} selected{{end}}>Thumbs Up</option>
      <option value="png:star"{{if eq .TokenStyle "png:star"}} selected{{end}}>PNG Star</option>
      <option value="png:smiley"{{if eq .TokenStyle "png:smiley"}} selected{{end}}>PNG Smiley</option>
      <option value="png:thumbsup"{{if eq .TokenStyle "png:thumbsup"}} selected{{end}}>PNG Thumbs Up</option>
    </select>
  </label>
  <label>Theme
    <select name="theme">
      <option value="default"{{if eq .Theme "default"}} selected{{end}}>Default</option>
      <option value="blue"{{if eq .Theme "blue"}} selected{{end}}>Blue</option>
      <option value="green"{{if eq .Theme "green"}} selected{{end}}>Green</option>
      <option value="pink"{{if eq .Theme "pink"}} selected{{end}}>Pink</option>
    </select>
  </label>
  <label>Page Size
    <select name="page_size">
      <option value="letter"{{if eq .PageSize "letter"}} selected{{end}}>Letter</option>
      <option value="a4"{{if eq .PageSize "a4"}} selected{{end}}>A4</option>
    </select>
  </label>
  <label>Custom Title
    <input type="text" name="title" placeholder="I am working for:" value="{{.Title}}">
  </label>
  <div class="btn-row">
    <button type="submit" class="btn-preview">Preview</button>
  </div>
</form>
</body>
</html>`

// formData holds values for pre-populating the form template.
type formData struct {
	Name       string
	Reward     string
	Tokens     int
	TokenStyle string
	Theme      string
	PageSize   string
	Title      string
}

var formTmpl = template.Must(template.New("form").Parse(formHTML))

const previewHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Token Board Preview</title>
<style>
  body { font-family: Arial, sans-serif; background: #eee; margin: 0; padding: 20px; }
  .board { width: 680px; margin: 0 auto; background: white; border: 2px solid #999; box-shadow: 2px 2px 8px rgba(0,0,0,0.2); }
  .header { background: {{.Theme.HeaderBg}}; color: {{.Theme.HeaderText}}; padding: 16px 20px; display: flex; align-items: center; justify-content: space-between; min-height: 80px; }
  .header-left { font-size: 20px; font-weight: bold; }
  .header-right { font-size: 24px; font-weight: bold; }
  .name-band { background: {{.Theme.NameBg}}; color: {{.Theme.NameText}}; text-align: center; padding: 10px; font-size: 22px; font-weight: bold; {{if not .HasName}}display:none;{{end}} }
  .token-row { background: {{.Theme.TokenBg}}; display: flex; justify-content: center; align-items: center; gap: 12px; padding: 24px 16px; }
  .token-slot { border: 2px solid {{.Theme.TokenBorder}}; border-radius: 8px; background: {{.Theme.TokenBg}}; display: flex; align-items: center; justify-content: center; font-size: 36px; }
  .footer { background: {{.Theme.FooterBg}}; border-top: 2px solid {{.Theme.FooterBorder}}; height: 80px; display: flex; align-items: center; justify-content: center; }
  .footer-inner { border: 2px dashed {{.Theme.FooterBorder}}; width: calc(100% - 20px); height: 60px; margin: 0 10px; border-radius: 4px; }
  .back { display: block; text-align: center; margin: 20px auto; color: #1565C0; text-decoration: none; font-size: 16px; }
  .dl-form { text-align: center; margin: 16px; }
  .dl-btn { padding: 10px 24px; background: #2E7D32; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 15px; }
</style>
</head>
<body>
<div class="board">
  <div class="header">
    <div class="header-left">{{.Title}}</div>
    <div class="header-right">{{.RewardText}}</div>
  </div>
  {{if .HasName}}<div class="name-band">{{.ChildName}}</div>{{end}}
  <div class="token-row">
    {{range .Tokens}}<div class="token-slot" style="width:{{$.SlotSize}}px;height:{{$.SlotSize}}px;">{{.}}</div>{{end}}
  </div>
  <div class="footer"><div class="footer-inner"></div></div>
</div>
<div class="dl-form">
  <form method="POST" action="/generate">
    <input type="hidden" name="name" value="{{.ChildName}}">
    <input type="hidden" name="reward" value="{{.RewardText}}">
    <input type="hidden" name="tokens" value="{{.TokenCount}}">
    <input type="hidden" name="token_style" value="{{.TokenStyle}}">
    <input type="hidden" name="theme" value="{{.ThemeName}}">
    <input type="hidden" name="page_size" value="{{.PageSize}}">
    <input type="hidden" name="title" value="{{.Title}}">
    <button type="submit" class="dl-btn">Download PDF</button>
  </form>
</div>
<a href="{{.BackURL}}" class="back">&#8592; Back to form</a>
</body>
</html>`

var previewTmpl = template.Must(template.New("preview").Parse(previewHTML))

// tokenEmoji maps builtin styles to a display emoji for the HTML preview.
var tokenEmoji = map[string]string{
	"star":     "⭐",
	"circle":   "⬤",
	"smiley":   "😊",
	"thumbsup": "👍",
	"png:star":     "⭐",
	"png:smiley":   "😊",
	"png:thumbsup": "👍",
}

// previewData is the data model for the HTML preview template.
type previewData struct {
	Title      string
	RewardText string
	ChildName  string
	HasName    bool
	Tokens     []string
	TokenCount int
	SlotSize   int
	TokenStyle string
	ThemeName  string
	PageSize   string
	Theme      previewTheme
	BackURL    string
}

type previewTheme struct {
	HeaderBg     template.CSS
	HeaderText   template.CSS
	NameBg       template.CSS
	NameText     template.CSS
	TokenBg      template.CSS
	TokenBorder  template.CSS
	TokenFill    template.CSS
	FooterBg     template.CSS
	FooterBorder template.CSS
}

// WebServer starts the token board web server on the given port.
func WebServer(ctx context.Context, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleForm)
	mux.HandleFunc("POST /preview", handlePreview)
	mux.HandleFunc("POST /generate", handleGenerate)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	fmt.Printf("Token Board Creator listening on http://localhost%s\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server: %w", err)
	}
	return nil
}

func handleForm(w http.ResponseWriter, r *http.Request) {
	tokens, _ := strconv.Atoi(r.URL.Query().Get("tokens"))
	if tokens == 0 {
		tokens = 5
	}
	tokenStyle := r.URL.Query().Get("token_style")
	if tokenStyle == "" {
		tokenStyle = "star"
	}
	theme := r.URL.Query().Get("theme")
	if theme == "" {
		theme = "default"
	}
	pageSize := r.URL.Query().Get("page_size")
	if pageSize == "" {
		pageSize = "letter"
	}
	fd := formData{
		Name:       r.URL.Query().Get("name"),
		Reward:     r.URL.Query().Get("reward"),
		Tokens:     tokens,
		TokenStyle: tokenStyle,
		Theme:      theme,
		PageSize:   pageSize,
		Title:      r.URL.Query().Get("title"),
	}
	var buf bytes.Buffer
	if err := formTmpl.Execute(&buf, fd); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	cfg, err := configFromForm(r)
	if err != nil {
		http.Error(w, "Invalid form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	themeData, err := loadPreviewTheme(cfg.Theme)
	if err != nil {
		http.Error(w, "Theme error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	emoji := tokenEmoji[cfg.TokenStyle]
	if emoji == "" {
		emoji = "⬜"
	}
	tokens := make([]string, cfg.TokenCount)
	for i := range tokens {
		tokens[i] = emoji
	}

	backParams := url.Values{}
	backParams.Set("name", cfg.ChildName)
	backParams.Set("reward", cfg.RewardText)
	backParams.Set("tokens", strconv.Itoa(cfg.TokenCount))
	backParams.Set("token_style", cfg.TokenStyle)
	backParams.Set("theme", cfg.Theme)
	backParams.Set("page_size", cfg.PageSize)
	backParams.Set("title", cfg.Title)

	// 648px usable width (680px board - 16px left+right padding).
	// N slots with 12px gaps: slot = floor((648 - 12*(N-1)) / N).
	n := cfg.TokenCount
	slotSize := (648 - 12*(n-1)) / n

	data := previewData{
		Title:      cfg.Title,
		RewardText: cfg.RewardText,
		ChildName:  cfg.ChildName,
		HasName:    cfg.ChildName != "",
		Tokens:     tokens,
		TokenCount: cfg.TokenCount,
		SlotSize:   slotSize,
		TokenStyle: cfg.TokenStyle,
		ThemeName:  cfg.Theme,
		PageSize:   cfg.PageSize,
		Theme:      themeData,
		BackURL:    "/?" + backParams.Encode(),
	}

	var buf bytes.Buffer
	if err := previewTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	cfg, err := configFromForm(r)
	if err != nil {
		http.Error(w, "Invalid form data: "+err.Error(), http.StatusBadRequest)
		return
	}

	tmp, err := os.CreateTemp("", "tokenboard_*.pdf")
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	cfg.Output = tmp.Name()
	if err := PDF(r.Context(), cfg); err != nil {
		http.Error(w, "PDF generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=tokenboard.pdf")
	http.ServeFile(w, r, tmp.Name())
}

// configFromForm parses and validates a Config from an HTTP form submission.
func configFromForm(r *http.Request) (board.Config, error) {
	if err := r.ParseForm(); err != nil {
		return board.Config{}, fmt.Errorf("parsing form: %w", err)
	}

	tokens, _ := strconv.Atoi(r.FormValue("tokens"))
	cfg := board.Config{
		ChildName:  r.FormValue("name"),
		RewardText: r.FormValue("reward"),
		TokenCount: tokens,
		TokenStyle: r.FormValue("token_style"),
		Theme:      r.FormValue("theme"),
		PageSize:   r.FormValue("page_size"),
		Title:      r.FormValue("title"),
	}
	if err := cfg.Validate(); err != nil {
		return board.Config{}, err
	}
	return cfg, nil
}

// loadPreviewTheme converts an assets.Theme into template-safe CSS strings.
func loadPreviewTheme(name string) (previewTheme, error) {
	t, err := assets.LoadTheme(name)
	if err != nil {
		return previewTheme{}, err
	}
	return previewTheme{
		HeaderBg:     template.CSS(t.HeaderBg),
		HeaderText:   template.CSS(t.HeaderText),
		NameBg:       template.CSS(t.NameBg),
		NameText:     template.CSS(t.NameText),
		TokenBg:      template.CSS(t.TokenBg),
		TokenBorder:  template.CSS(t.TokenBorder),
		TokenFill:    template.CSS(t.TokenFill),
		FooterBg:     template.CSS(t.FooterBg),
		FooterBorder: template.CSS(t.FooterBorder),
	}, nil
}
