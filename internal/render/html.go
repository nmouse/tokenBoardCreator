package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/owner/tokenBoardCreator/internal/assets"
	"github.com/owner/tokenBoardCreator/internal/board"
	"github.com/owner/tokenBoardCreator/internal/imagegen"
)

var (
	hfTokenMu     sync.RWMutex
	storedHFToken string
)

func getStoredHFToken() string {
	hfTokenMu.RLock()
	defer hfTokenMu.RUnlock()
	return storedHFToken
}

func setStoredHFToken(t string) {
	hfTokenMu.Lock()
	defer hfTokenMu.Unlock()
	storedHFToken = t
}

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
<div style="display:flex;align-items:baseline;justify-content:space-between;">
  <h1>Token Board Creator</h1>
  <a href="/settings" style="color:#888;font-size:13px;text-decoration:none;">&#9881; Settings</a>
</div>
<form method="POST" action="/preview" enctype="multipart/form-data">
  <label>Child Name (optional)
    <input type="text" name="name" placeholder="e.g. Alex" value="{{.Name}}">
  </label>
  <label>Reward Text <span style="font-weight:normal;color:#888">(optional if uploading image below)</span>
    <input type="text" name="reward" placeholder="e.g. iPad time" value="{{.Reward}}">
  </label>
  <label>Reward Image <span style="font-weight:normal;color:#888">(optional — replaces reward text)</span>
    {{if .RewardImageSrc}}<div style="margin-top:4px"><img src="{{.RewardImageSrc}}" style="max-height:60px;border:1px solid #ccc;border-radius:4px;"><br><small style="color:#888">Uploaded — choose a new file to replace</small></div>{{end}}
    <input type="file" name="reward_image" accept="image/*" style="padding:4px 0;">
    <input type="hidden" name="reward_image_data" value="{{.RewardImageData}}">
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
      {{if eq .TokenStyle "custom"}}<option value="custom" selected>Custom Image (uploaded)</option>{{end}}
    </select>
  </label>
  <label>Custom Token Image <span style="font-weight:normal;color:#888">(optional — overrides token style)</span>
    {{if .TokenImageSrc}}<div style="margin-top:4px"><img src="{{.TokenImageSrc}}" style="max-height:60px;border:1px solid #ccc;border-radius:4px;"><br><small style="color:#888">Uploaded — choose a new file to replace, or select a different style above</small></div>{{end}}
    <input type="file" name="token_image" accept="image/*" style="padding:4px 0;">
    <input type="hidden" name="token_image_data" value="{{.TokenImageData}}">
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
  <label>Background Scene <span style="font-weight:normal;color:#888">(optional — requires Hugging Face token; set in <a href="/settings">Settings</a>; ~10–30 sec)</span>
    <input type="text" name="background_prompt" placeholder="e.g. dinosaurs in space, rainbow forest" value="{{.BackgroundPrompt}}">
  </label>
  <div class="btn-row">
    <button type="submit" class="btn-preview">Preview</button>
  </div>
</form>
<div id="ai-loading" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:999;flex-direction:column;align-items:center;justify-content:center;color:#fff;font-family:Arial,sans-serif;font-size:18px;gap:16px;">
  <div style="width:48px;height:48px;border:5px solid rgba(255,255,255,.3);border-top-color:#fff;border-radius:50%;animation:spin 1s linear infinite;"></div>
  Generating AI background&hellip; (~10&ndash;30 sec)
</div>
<style>@keyframes spin{to{transform:rotate(360deg)}}</style>
<script>
document.querySelector('form').addEventListener('submit', function() {
  if (document.querySelector('[name="background_prompt"]').value.trim()) {
    document.getElementById('ai-loading').style.display = 'flex';
  }
});
</script>
</body>
</html>`

// formData holds values for pre-populating the form template.
type formData struct {
	Name             string
	Reward           string
	Tokens           int
	TokenStyle       string
	Theme            string
	PageSize         string
	Title            string
	BackgroundPrompt string
	RewardImageData  string       // URL-safe base64 reward image for hidden field passthrough
	RewardImageSrc   template.URL // data: URL for thumbnail preview
	TokenImageData   string       // URL-safe base64 token image for hidden field passthrough
	TokenImageSrc    template.URL // data: URL for thumbnail preview
}

var formTmpl = template.Must(template.New("form").Parse(formHTML))

const settingsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Settings — Token Board Creator</title>
<style>
  body { font-family: Arial, sans-serif; max-width: 600px; margin: 40px auto; padding: 0 20px; background: #f9f9f9; }
  h1 { color: #333; }
  label { display: block; margin-top: 12px; font-weight: bold; color: #555; }
  input[type=password] { width: 100%; padding: 8px; margin-top: 4px; box-sizing: border-box; border: 1px solid #ccc; border-radius: 4px; font-family: monospace; }
  .hint { font-size: 13px; color: #777; margin-top: 6px; }
  .badge { display: inline-block; padding: 2px 10px; border-radius: 12px; font-size: 13px; font-weight: bold; }
  .badge-set { background: #d4edda; color: #155724; }
  .badge-unset { background: #fff3cd; color: #856404; }
  .btn-row { margin-top: 20px; display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
  button { padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
  .btn-save { background: #1565C0; color: white; }
  .btn-clear { background: #c62828; color: white; }
  .back { color: #1565C0; font-size: 14px; text-decoration: none; }
  button:hover { opacity: 0.85; }
  p { color: #555; line-height: 1.5; }
</style>
</head>
<body>
<h1>Settings</h1>
<p>A Hugging Face API token is only needed for AI-generated backgrounds. It is stored in memory for this session — you will need to re-enter it if you restart the app.</p>
<p>Status: {{if .TokenSet}}<span class="badge badge-set">&#10003; Token saved for this session</span>{{else}}<span class="badge badge-unset">Not set</span>{{end}}</p>
<form method="POST" action="/settings">
  <label>Hugging Face API Token
    <input type="password" name="hf_token" placeholder="hf_..." autocomplete="off">
  </label>
  <p class="hint">Get a free token at <a href="https://huggingface.co/settings/tokens" target="_blank">huggingface.co/settings/tokens</a> — read access is sufficient.</p>
  <p id="save-msg" style="display:none;color:#c62828;font-size:13px;margin:8px 0 0;"></p>
  <div class="btn-row">
    <button type="submit" name="action" value="save" class="btn-save">Save</button>
    {{if .TokenSet}}<button type="submit" name="action" value="clear" class="btn-clear">Clear</button>{{end}}
    <a href="/" class="back">&#8592; Back to form</a>
  </div>
</form>
<script>
  var tokenInput = document.querySelector('input[name="hf_token"]');
  var saveMsg = document.getElementById('save-msg');
  document.querySelector('form').addEventListener('submit', function(e) {
    if (e.submitter && e.submitter.value === 'save' && tokenInput.value.trim() === '') {
      e.preventDefault();
      saveMsg.textContent = 'Please enter a token before saving.';
      saveMsg.style.display = 'block';
      tokenInput.focus();
    }
  });
  tokenInput.addEventListener('input', function() {
    saveMsg.style.display = 'none';
  });
</script>
</body>
</html>`

var settingsTmpl = template.Must(template.New("settings").Parse(settingsHTML))

const previewHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Token Board Preview</title>
<style>
  body { font-family: Arial, sans-serif; background: #eee; margin: 0; padding: 20px; }
  .board { width: 680px; margin: 0 auto; background: white; border: 2px solid #999; box-shadow: 2px 2px 8px rgba(0,0,0,0.2); overflow: hidden; }
  .header { background: {{.Theme.HeaderBg}}; color: {{.Theme.HeaderText}}; padding: 16px 20px; display: flex; align-items: center; justify-content: space-between; min-height: 80px; }
  .header-left { font-size: 20px; font-weight: bold; }
  .header-right { font-size: 24px; font-weight: bold; }
  .name-band { background: {{.Theme.NameBg}}; color: {{.Theme.NameText}}; text-align: center; padding: 10px; font-size: 22px; font-weight: bold; {{if not .HasName}}display:none;{{end}} }
  .token-row { background: {{.Theme.TokenBg}}; display: flex; justify-content: center; align-items: center; gap: 12px; padding: 24px 16px; }
  .token-slot { border: 2px solid {{.Theme.TokenBorder}}; border-radius: 8px; background: {{.Theme.TokenBg}}; display: flex; align-items: center; justify-content: center; font-size: 36px; }
  .footer { background: {{.Theme.FooterBg}}; border-top: 2px solid {{.Theme.FooterBorder}}; height: 80px; display: flex; align-items: center; justify-content: center; }
  .footer-inner { border: 2px dashed {{.Theme.FooterBorder}}; width: calc(100% - 20px); height: 60px; margin: 0 10px; border-radius: 4px; }
  .back-form { text-align: center; margin: 20px auto; }
  .back { background: none; border: none; cursor: pointer; color: #1565C0; font-size: 16px; }
  .dl-form { text-align: center; margin: 16px; }
  .dl-btn { padding: 10px 24px; background: #2E7D32; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 15px; }
</style>
</head>
<body>
<div class="board"{{if .BackgroundImageSrc}} style="background-image:url({{.BackgroundImageSrc}});background-size:cover;background-position:center;"{{end}}>
  <div class="header"{{if .BackgroundImageSrc}} style="opacity:0.82;"{{end}}>
    <div class="header-left">{{.Title}}</div>
    <div class="header-right">{{if .RewardImageSrc}}<img src="{{.RewardImageSrc}}" style="max-height:80px;max-width:200px;object-fit:contain;">{{else}}{{.RewardText}}{{end}}</div>
  </div>
  {{if .HasName}}<div class="name-band"{{if $.BackgroundImageSrc}} style="opacity:0.82;"{{end}}>{{.ChildName}}</div>{{end}}
  <div class="token-row"{{if .BackgroundImageSrc}} style="opacity:0.82;"{{end}}>
    {{range .Tokens}}<div class="token-slot" style="width:{{$.SlotSize}}px;height:{{$.SlotSize}}px;">{{.}}</div>{{end}}
  </div>
  <div class="footer"{{if .BackgroundImageSrc}} style="opacity:0.82;"{{end}}><div class="footer-inner"></div></div>
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
    <input type="hidden" name="background_prompt" value="{{.BackgroundPrompt}}">
    <input type="hidden" name="reward_image_data" value="{{.RewardImageData}}">
    <input type="hidden" name="token_image_data" value="{{.TokenImageData}}">
    <input type="hidden" name="background_image_data" value="{{.BackgroundImageData}}">
    <button type="submit" class="dl-btn">{{if .BackgroundPrompt}}Download PDF (with AI background){{else}}Download PDF{{end}}</button>
  </form>
</div>
<form method="POST" action="/" class="back-form">
  <input type="hidden" name="name" value="{{.ChildName}}">
  <input type="hidden" name="reward" value="{{.RewardText}}">
  <input type="hidden" name="tokens" value="{{.TokenCount}}">
  <input type="hidden" name="token_style" value="{{.TokenStyle}}">
  <input type="hidden" name="theme" value="{{.ThemeName}}">
  <input type="hidden" name="page_size" value="{{.PageSize}}">
  <input type="hidden" name="title" value="{{.Title}}">
  <input type="hidden" name="background_prompt" value="{{.BackgroundPrompt}}">
  <input type="hidden" name="reward_image_data" value="{{.RewardImageData}}">
  <input type="hidden" name="token_image_data" value="{{.TokenImageData}}">
  <input type="hidden" name="background_image_data" value="{{.BackgroundImageData}}">
  <button type="submit" class="back">&#8592; Back to form</button>
</form>
</body>
</html>`

var previewTmpl = template.Must(template.New("preview").Parse(previewHTML))

// tokenEmoji maps builtin styles to a display emoji for the HTML preview.
var tokenEmoji = map[string]string{
	"star":         "⭐",
	"circle":       "⬤",
	"smiley":       "😊",
	"thumbsup":     "👍",
	"png:star":     "⭐",
	"png:smiley":   "😊",
	"png:thumbsup": "👍",
}

// previewData is the data model for the HTML preview template.
type previewData struct {
	Title               string
	RewardText          string
	ChildName           string
	HasName             bool
	Tokens              []string
	TokenCount          int
	SlotSize            int
	TokenStyle          string
	ThemeName           string
	PageSize            string
	Theme               previewTheme
	BackURL             string
	BackgroundPrompt    string
	RewardImageData     string       // URL-safe base64 reward image (for hidden field passthrough)
	RewardImageSrc      template.URL // data: URL for <img src> preview
	TokenImageData      string       // URL-safe base64 custom token image (for hidden field passthrough)
	BackgroundImageData string       // URL-safe base64 AI background image (for hidden field passthrough)
	BackgroundImageSrc  template.URL // data: URL for CSS background display in preview
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
	mux.HandleFunc("POST /", handleForm)
	mux.HandleFunc("POST /preview", handlePreview)
	mux.HandleFunc("POST /generate", handleGenerate)
	mux.HandleFunc("GET /settings", handleSettings)
	mux.HandleFunc("POST /settings", handleSettings)

	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	url := fmt.Sprintf("http://localhost%s", addr)
	fmt.Printf("Token Board Creator listening on %s\n", url)
	go openBrowser(url)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server: %w", err)
	}
	return nil
}

// openBrowser opens the default system browser to the given URL.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.FormValue("action") == "clear" {
			setStoredHFToken("")
		} else if token := strings.TrimSpace(r.FormValue("hf_token")); token != "" {
			setStoredHFToken(token)
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	data := struct{ TokenSet bool }{TokenSet: getStoredHFToken() != ""}
	var buf bytes.Buffer
	if err := settingsTmpl.Execute(&buf, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func handleForm(w http.ResponseWriter, r *http.Request) {
	tokens, _ := strconv.Atoi(r.FormValue("tokens"))
	if tokens == 0 {
		tokens = 5
	}
	tokenStyle := r.FormValue("token_style")
	if tokenStyle == "" {
		tokenStyle = "star"
	}
	theme := r.FormValue("theme")
	if theme == "" {
		theme = "default"
	}
	pageSize := r.FormValue("page_size")
	if pageSize == "" {
		pageSize = "letter"
	}
	fd := formData{
		Name:             r.FormValue("name"),
		Reward:           r.FormValue("reward"),
		Tokens:           tokens,
		TokenStyle:       tokenStyle,
		Theme:            theme,
		PageSize:         pageSize,
		Title:            r.FormValue("title"),
		BackgroundPrompt: r.FormValue("background_prompt"),
	}
	// Restore uploaded images when navigating back from preview.
	fd.RewardImageData = r.FormValue("reward_image_data")
	if fd.RewardImageData != "" {
		if b, err := base64.URLEncoding.DecodeString(fd.RewardImageData); err == nil && len(b) > 0 {
			fd.RewardImageSrc = imageDataURL(b)
		}
	}
	fd.TokenImageData = r.FormValue("token_image_data")
	if fd.TokenImageData != "" {
		if b, err := base64.URLEncoding.DecodeString(fd.TokenImageData); err == nil && len(b) > 0 {
			fd.TokenImageSrc = imageDataURL(b)
		}
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
		writeHTMLError(w, http.StatusBadRequest, "Invalid form data: "+err.Error())
		return
	}

	themeData, err := loadPreviewTheme(cfg.Theme)
	if err != nil {
		writeHTMLError(w, http.StatusInternalServerError, "Theme error: "+err.Error())
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
	backParams.Set("title", r.FormValue("title"))
	if cfg.BackgroundPrompt != "" {
		backParams.Set("background_prompt", cfg.BackgroundPrompt)
	}

	// 648px usable width (680px board - 16px left+right padding).
	// N slots with 12px gaps: slot = floor((648 - 12*(N-1)) / N).
	n := cfg.TokenCount
	slotSize := (648 - 12*(n-1)) / n

	rewardImgBytes, _ := resolveImageData(r, "reward_image", "reward_image_data")
	rewardImgData := base64.URLEncoding.EncodeToString(rewardImgBytes)
	var rewardImgSrc template.URL
	if len(rewardImgBytes) > 0 {
		rewardImgSrc = imageDataURL(rewardImgBytes)
	}
	tokenImgBytes, _ := resolveImageData(r, "token_image", "token_image_data")
	tokenImgData := base64.URLEncoding.EncodeToString(tokenImgBytes)

	// Generate (or reuse cached) AI background image.
	var bgImgData string
	var bgImgSrc template.URL
	if cfg.BackgroundPrompt != "" {
		bgBytes, _ := resolveImageData(r, "", "background_image_data")
		if len(bgBytes) == 0 {
			// No cached image — generate one now.
			apiToken := os.Getenv("HF_TOKEN")
			if apiToken == "" {
				apiToken = getStoredHFToken()
			}
			if apiToken == "" {
				writeHTMLError(w, http.StatusBadRequest,
					"No Hugging Face API token configured.\nGo to Settings (top-right) to enter your token.\nGet a free token at https://huggingface.co/settings/tokens")
				return
			}
			generated, err := imagegen.Generate(r.Context(), cfg.BackgroundPrompt, apiToken)
			if err != nil {
				writeHTMLError(w, http.StatusBadGateway, "Background image generation failed: "+err.Error())
				return
			}
			bgBytes = generated
		}
		bgImgData = base64.URLEncoding.EncodeToString(bgBytes)
		bgImgSrc = imageDataURL(bgBytes)
	}

	data := previewData{
		Title:               cfg.Title,
		RewardText:          cfg.RewardText,
		ChildName:           cfg.ChildName,
		HasName:             cfg.ChildName != "",
		Tokens:              tokens,
		TokenCount:          cfg.TokenCount,
		SlotSize:            slotSize,
		TokenStyle:          cfg.TokenStyle,
		ThemeName:           cfg.Theme,
		PageSize:            cfg.PageSize,
		Theme:               themeData,
		BackURL:             "/?" + backParams.Encode(),
		BackgroundPrompt:    cfg.BackgroundPrompt,
		RewardImageData:     rewardImgData,
		RewardImageSrc:      rewardImgSrc,
		TokenImageData:      tokenImgData,
		BackgroundImageData: bgImgData,
		BackgroundImageSrc:  bgImgSrc,
	}

	var buf bytes.Buffer
	if err := previewTmpl.Execute(&buf, data); err != nil {
		writeHTMLError(w, http.StatusInternalServerError, "Template error: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

// writeHTMLError renders a styled error page with a back button.
func writeHTMLError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8">
<title>Error</title>
<style>body{font-family:Arial,sans-serif;max-width:600px;margin:60px auto;padding:0 20px;background:#f9f9f9}
.box{background:#fff;border:1px solid #f5c6cb;border-radius:6px;padding:20px 24px;color:#721c24;background-color:#f8d7da}
h2{margin-top:0}pre{white-space:pre-wrap;word-break:break-word;margin:8px 0 0}
.back{display:inline-block;margin-top:20px;padding:8px 18px;background:#1565C0;color:#fff;border:none;border-radius:4px;cursor:pointer;font-size:14px;text-decoration:none}
</style></head><body>
<div class="box"><h2>Something went wrong</h2><pre>%s</pre></div>
<a class="back" href="javascript:history.back()">&#8592; Go back</a>
</body></html>`, template.HTMLEscapeString(msg))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	cfg, err := configFromForm(r)
	if err != nil {
		writeHTMLError(w, http.StatusBadRequest, "Invalid form data: "+err.Error())
		return
	}

	if cfg.RewardImage == "uploaded" {
		imgBytes, err := resolveImageData(r, "reward_image", "reward_image_data")
		if err != nil || len(imgBytes) == 0 {
			writeHTMLError(w, http.StatusBadRequest, "Reward image data missing or invalid")
			return
		}
		tmpPath, err := writeTempImage(imgBytes, "reward_")
		if err != nil {
			writeHTMLError(w, http.StatusInternalServerError, "Server error")
			return
		}
		defer os.Remove(tmpPath)
		cfg.RewardImage = tmpPath
	}

	if cfg.TokenStyle == "custom" {
		imgBytes, err := resolveImageData(r, "token_image", "token_image_data")
		if err != nil || len(imgBytes) == 0 {
			writeHTMLError(w, http.StatusBadRequest, "Token image data missing or invalid")
			return
		}
		tmpPath, err := writeTempImage(imgBytes, "token_")
		if err != nil {
			writeHTMLError(w, http.StatusInternalServerError, "Server error")
			return
		}
		defer os.Remove(tmpPath)
		cfg.TokenStyle = tmpPath
	}

	if cfg.BackgroundPrompt != "" {
		imgBytes, err := resolveImageData(r, "", "background_image_data")
		if err != nil || len(imgBytes) == 0 {
			writeHTMLError(w, http.StatusBadRequest, "Background image data missing — go back and regenerate the preview.")
			return
		}
		cfg.BackgroundImageBytes = imgBytes
	}

	tmp, err := os.CreateTemp("", "tokenboard_*.pdf")
	if err != nil {
		writeHTMLError(w, http.StatusInternalServerError, "Server error")
		return
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	cfg.Output = tmp.Name()
	if err := PDF(r.Context(), cfg); err != nil {
		writeHTMLError(w, http.StatusInternalServerError, "PDF generation failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=tokenboard.pdf")
	http.ServeFile(w, r, tmp.Name())
}

// configFromForm parses and validates a Config from an HTTP form submission.
func configFromForm(r *http.Request) (board.Config, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil && err != http.ErrNotMultipart {
		return board.Config{}, fmt.Errorf("parsing form: %w", err)
	}

	tokens, _ := strconv.Atoi(r.FormValue("tokens"))
	cfg := board.Config{
		ChildName:        r.FormValue("name"),
		RewardText:       r.FormValue("reward"),
		TokenCount:       tokens,
		TokenStyle:       r.FormValue("token_style"),
		Theme:            r.FormValue("theme"),
		PageSize:         r.FormValue("page_size"),
		Title:            r.FormValue("title"),
		BackgroundPrompt: r.FormValue("background_prompt"),
	}

	// Sentinels let Validate() pass when images are provided via upload.
	if r.FormValue("reward_image_data") != "" || hasUpload(r, "reward_image") {
		cfg.RewardImage = "uploaded"
	}
	if hasUpload(r, "token_image") || (r.FormValue("token_style") == "custom" && r.FormValue("token_image_data") != "") {
		cfg.TokenStyle = "custom"
	}

	if err := cfg.Validate(); err != nil {
		return board.Config{}, err
	}
	return cfg, nil
}

// readUpload reads the bytes from a multipart file upload field, returning nil if absent or on error.
func readUpload(r *http.Request, field string) []byte {
	f, _, err := r.FormFile(field)
	if err != nil {
		return nil
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	return b
}

// hasUpload reports whether a non-empty file was uploaded for the given field.
func hasUpload(r *http.Request, field string) bool {
	_, fh, err := r.FormFile(field)
	return err == nil && fh.Size > 0
}

// imageDataURL returns a data: URL for image bytes, detecting the MIME type from the content.
func imageDataURL(b []byte) template.URL {
	mime := http.DetectContentType(b)
	return template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b))
}

// imageExt returns the file extension for image bytes based on magic bytes.
func imageExt(b []byte) string {
	switch {
	case len(b) >= 4 && b[0] == 0x89 && b[1] == 'P' && b[2] == 'N' && b[3] == 'G':
		return ".png"
	case len(b) >= 2 && b[0] == 0xff && b[1] == 0xd8:
		return ".jpg"
	default:
		return ".png"
	}
}

// writeTempImage writes image bytes to a temp file with the correct extension and returns the path.
func writeTempImage(imgBytes []byte, prefix string) (string, error) {
	ext := imageExt(imgBytes)
	tmp, err := os.CreateTemp("", prefix+"*"+ext)
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if _, err := tmp.Write(imgBytes); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// resolveImageData returns image bytes from a direct upload field or a URL-safe base64-encoded form field.
func resolveImageData(r *http.Request, uploadField, dataField string) ([]byte, error) {
	if b := readUpload(r, uploadField); len(b) > 0 {
		return b, nil
	}
	encoded := r.FormValue(dataField)
	if encoded == "" {
		return nil, nil
	}
	return base64.URLEncoding.DecodeString(encoded)
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
