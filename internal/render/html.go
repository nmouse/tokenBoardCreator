package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&display=swap" rel="stylesheet">
<style>
  *{box-sizing:border-box}
  body{font-family:'Nunito','Segoe UI',Arial,sans-serif;max-width:640px;margin:0 auto;padding:0 0 48px;background:#F3F0FF}
  .page-header{background:linear-gradient(135deg,#5C6BC0,#7E57C2);color:white;padding:18px 24px;margin-bottom:20px;display:flex;align-items:center;justify-content:space-between}
  .page-header h1{margin:0;font-size:20px;font-weight:700}
  .page-header a{color:rgba(255,255,255,.85);font-size:13px;text-decoration:none}
  .card{background:white;border-radius:12px;padding:20px;margin:0 12px 16px;box-shadow:0 1px 6px rgba(92,107,192,.10)}
  .card-title{font-size:14px;font-weight:800;color:#5C6BC0;margin:0 0 16px;padding-bottom:10px;border-bottom:2px solid #EDE7F6}
  label{display:block;margin-top:16px;font-weight:600;font-size:14px;color:#37474F}
  .card>label:first-of-type,.card-first-label{margin-top:0}
  .hint{font-weight:400;color:#9E9E9E;font-size:13px}
  input[type=text],input[type=number],select{width:100%;padding:10px 12px;margin-top:6px;border:1.5px solid #D1C4E9;border-radius:8px;font-size:15px;color:#37474F;background:#FAFAFA}
  input[type=text]:focus,input[type=number]:focus,select:focus{outline:none;border-color:#7E57C2;background:white}
  input[type=file]{margin-top:6px;font-size:14px;color:#555}
  input[type=file]::file-selector-button{padding:6px 12px;background:#EDE7F6;border:1px solid #D1C4E9;border-radius:6px;color:#5C6BC0;font-size:13px;cursor:pointer;margin-right:8px}
  input[type=file]::file-selector-button:hover{background:#D1C4E9}
  .page-tagline{font-size:14px;color:rgba(255,255,255,.9);margin-top:4px}
  .mode-toggle{display:flex;gap:16px;margin:8px 0 6px}
  .mode-toggle label{display:flex;align-items:center;gap:6px;font-weight:500;font-size:14px;color:#555;cursor:pointer;margin:0}
  .mode-toggle input[type=radio]{width:auto;margin:0;accent-color:#7E57C2}
  .checkbox-row{display:flex;align-items:center;gap:10px;margin-top:16px;cursor:pointer}
  .checkbox-row input[type=checkbox]{width:17px;height:17px;margin:0;accent-color:#7E57C2;cursor:pointer;flex-shrink:0}
  .checkbox-row span{font-weight:600;font-size:14px;color:#37474F}
  .custom-token-box{border:1.5px solid #D1C4E9;border-radius:8px;padding:12px 14px;margin-top:8px;background:#F9F7FF;display:none}
  .show-custom-link{display:block;margin-top:10px;font-size:13px;color:#5C6BC0;text-decoration:none}
  .show-custom-link:hover{text-decoration:underline}
  .submit-card{background:none;box-shadow:none;padding:4px 12px}
  .btn-preview{width:100%;padding:15px;background:linear-gradient(135deg,#5C6BC0 0%,#7E57C2 100%);color:white;border:none;border-radius:10px;font-size:16px;font-weight:700;cursor:pointer;letter-spacing:.01em}
  .btn-preview:hover{opacity:.92}
</style>
</head>
<body>
<div class="page-header">
  <div>
    <h1>Token Board Creator</h1>
    <div class="page-tagline">Make printable reward boards for ABA therapy</div>
  </div>
  <a href="/settings">&#9881; Settings</a>
</div>
<script>var _restoredTokens={{.CustomTokensJSON}};var _hasRewardPrompt={{if .RewardImagePrompt}}true{{else}}false{{end}};var _restoredTokenStyle={{.TokenStyleJSON}};</script>
<form method="POST" action="/preview" enctype="multipart/form-data">

  <div class="card">
    <div class="card-title">Board Content</div>
    <label class="card-first-label">Child Name <span class="hint">(optional)</span>
      <input type="text" name="name" placeholder="e.g. Alex" value="{{.Name}}">
    </label>
    <label>Reward Text <span class="hint">(or use a reward image below) <span style="color:#c62828;">*</span></span>
      <input type="text" name="reward" placeholder="e.g. iPad time" value="{{.Reward}}" id="reward-input">
    </label>
    <div id="reward-error" style="display:none;color:#c62828;font-size:13px;margin-top:6px;padding:8px 12px;background:#FFF5F5;border:1px solid #FFCDD2;border-radius:6px;">Please enter a Reward Text, or upload / generate a Reward Image below.</div>
    <label>Reward Image <span class="hint">(optional — replaces reward text)</span></label>
    <div style="margin-top:4px;">
      <div class="mode-toggle">
        <label><input type="radio" name="reward_image_mode" value="upload" id="ri_mode_upload" onchange="rewardModeChange()" checked> Upload</label>
        <label><input type="radio" name="reward_image_mode" value="ai" id="ri_mode_ai" onchange="rewardModeChange()"> AI Generate</label>
      </div>
      <div id="ri_upload_section">
        {{if .RewardImageSrc}}<div style="margin-bottom:4px;"><img src="{{.RewardImageSrc}}" style="max-height:60px;border:1px solid #ccc;border-radius:4px;"><br><small class="hint">Uploaded — choose a new file to replace</small></div>{{end}}
        <input type="file" name="reward_image" accept="image/*">
      </div>
      <div id="ri_ai_section" style="display:none;">
        <input type="text" name="reward_image_prompt" placeholder="e.g. ice cream cone, golden trophy" value="{{.RewardImagePrompt}}">
        {{if .RewardImageSrc}}<div style="margin-top:4px;"><img src="{{.RewardImageSrc}}" style="max-height:60px;border:1px solid #ccc;border-radius:4px;"></div>{{end}}
        <small class="hint">Requires Hugging Face token — <a href="/settings">Settings</a></small>
      </div>
      <input type="hidden" name="reward_image_data" value="{{.RewardImageData}}">
    </div>
    <label>Number of Tokens (3–10)
      <input type="number" name="tokens" min="3" max="10" value="{{.Tokens}}" required>
    </label>
  </div>

  <div class="card">
    <div class="card-title">Token Style</div>
    <label id="global-token-style-label" class="card-first-label">Style
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
    <div class="checkbox-row">
      <input type="checkbox" id="individual_styles" name="individual_styles"{{if .IndividualStyles}} checked{{end}}>
      <span>Customize each slot individually <span class="hint">— pick a different style per slot</span></span>
    </div>
    <div id="slot-styles-container"></div>
    <input type="hidden" id="token_styles_data" value="{{.TokenStylesData}}">
    <hr style="border:none;border-top:1px solid #EDE7F6;margin:14px 0 0;">
    <a href="#" class="show-custom-link" id="show-custom-link" onclick="toggleCustomToken(event)">+ Add a custom token image</a>
    <div id="add-custom-token-section" class="custom-token-box">
      <strong style="font-size:13px;color:#5C6BC0;">Custom Token Image</strong>
      <div class="mode-toggle" style="margin-top:6px;">
        <label><input type="radio" name="add_token_mode" value="upload" id="at_mode_upload" onchange="addCustomTokenMode()" checked> Upload File</label>
        <label><input type="radio" name="add_token_mode" value="ai" id="at_mode_ai" onchange="addCustomTokenMode()"> AI Generate</label>
      </div>
      <div id="at_upload_section">
        <input type="file" id="at_file" accept="image/*">
      </div>
      <div id="at_ai_section" style="display:none;">
        <div style="display:flex;gap:8px;align-items:center;margin-top:6px;">
          <input type="text" id="at_prompt" placeholder="e.g. golden star, rocket ship" style="flex:1;margin:0;">
          <button type="button" id="at_gen_btn" onclick="generateCustomToken()" style="padding:9px 12px;background:#5C6BC0;color:white;border:none;border-radius:8px;cursor:pointer;white-space:nowrap;font-size:13px;font-weight:600;">Generate</button>
        </div>
        <span id="at_gen_status" style="font-size:12px;color:#888;margin-top:4px;display:block;"></span>
        <small class="hint">Requires Hugging Face token — <a href="/settings">Settings</a></small>
      </div>
      <div id="at_preview" style="margin-top:6px;"></div>
      <button type="button" onclick="addCustomToken()" style="margin-top:8px;padding:7px 14px;background:#5C6BC0;color:white;border:none;border-radius:8px;cursor:pointer;font-size:13px;font-weight:600;">+ Add to Token List</button>
      <div id="custom-token-hidden-fields"></div>
    </div>
  </div>

  <div class="card">
    <div class="card-title">Appearance</div>
    <label class="card-first-label">Theme
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
        <option value="letter-half"{{if eq .PageSize "letter-half"}} selected{{end}}>Letter — Half Page</option>
        <option value="a4-half"{{if eq .PageSize "a4-half"}} selected{{end}}>A4 — Half Page</option>
      </select>
    </label>
    <label>Custom Title <span class="hint">(default: "I am working for:")</span>
      <input type="text" name="title" placeholder="I am working for:" value="{{.Title}}">
    </label>
    <label>Background Scene <span class="hint">(optional)</span></label>
    <div style="margin-top:4px;">
      <div class="mode-toggle">
        <label><input type="radio" name="background_mode" value="ai" id="bg_mode_ai" onchange="bgModeChange()" checked> AI Generate</label>
        <label><input type="radio" name="background_mode" value="upload" id="bg_mode_upload" onchange="bgModeChange()"> Upload File</label>
      </div>
      <div id="bg_ai_section">
        <input type="text" name="background_prompt" placeholder="e.g. dinosaurs in space, rainbow forest" value="{{.BackgroundPrompt}}">
        <small class="hint">Requires Hugging Face token — <a href="/settings">Settings</a> &middot; ~10&ndash;30 sec</small>
      </div>
      <div id="bg_upload_section" style="display:none;">
        {{if .BackgroundImageSrc}}<div style="margin-bottom:4px;"><img src="{{.BackgroundImageSrc}}" style="max-height:60px;border:1px solid #ccc;border-radius:4px;"><br><small class="hint">Uploaded — choose a new file to replace</small></div>{{end}}
        <input type="file" name="background_image" accept="image/*">
      </div>
      <input type="hidden" name="background_image_data" value="{{.BackgroundImageData}}">
    </div>
  </div>

  <div class="submit-card card">
    <button type="submit" class="btn-preview">Preview Board &#8594;</button>
  </div>

</form>
<div id="ai-loading" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,.55);z-index:999;flex-direction:column;align-items:center;justify-content:center;color:#fff;font-family:Arial,sans-serif;font-size:18px;gap:16px;">
  <div style="width:48px;height:48px;border:5px solid rgba(255,255,255,.3);border-top-color:#fff;border-radius:50%;animation:spin 1s linear infinite;"></div>
  Generating AI image&hellip; (~10&ndash;30 sec)
</div>
<style>@keyframes spin{to{transform:rotate(360deg)}}</style>
<script>
(function() {
  var customTokens = [];
  var pendingCustomImage = null;

  function buildStyleOptions() {
    var base = [
      {value:'star', label:'Star'},
      {value:'circle', label:'Circle'},
      {value:'smiley', label:'Smiley'},
      {value:'thumbsup', label:'Thumbs Up'},
      {value:'png:star', label:'PNG Star'},
      {value:'png:smiley', label:'PNG Smiley'},
      {value:'png:thumbsup', label:'PNG Thumbs Up'},
    ];
    customTokens.forEach(function(_, i) {
      base.push({value: 'custom:' + i, label: 'Custom ' + (i + 1)});
    });
    return base;
  }

  function updateSlotDropdowns() {
    var checked = document.getElementById('individual_styles').checked;
    document.getElementById('global-token-style-label').style.display = checked ? 'none' : '';
    var showLink = document.getElementById('show-custom-link');
    if (checked) {
      document.getElementById('add-custom-token-section').style.display = 'none';
      if (showLink) showLink.style.display = 'none';
    } else {
      if (showLink) showLink.style.display = '';
    }

    var container = document.getElementById('slot-styles-container');
    container.innerHTML = '';
    if (!checked) return;

    var n = parseInt(document.querySelector('[name="tokens"]').value) || 5;
    var existingRaw = document.getElementById('token_styles_data').value;
    var existing = existingRaw ? existingRaw.split(',') : [];
    var opts = buildStyleOptions();

    for (var i = 0; i < n; i++) {
      var val = existing[i] || opts[0].value;
      var wrapper = document.createElement('div');
      wrapper.style.cssText = 'display:flex;align-items:center;gap:8px;margin-top:6px;';
      var lbl = document.createElement('span');
      lbl.style.cssText = 'min-width:52px;color:#555;font-size:14px;';
      lbl.textContent = 'Slot ' + (i + 1) + ':';
      var sel = document.createElement('select');
      sel.name = 'token_style_' + (i + 1);
      sel.style.cssText = 'flex:1;margin-top:0;';
      opts.forEach(function(opt) {
        var o = document.createElement('option');
        o.value = opt.value;
        o.text = opt.label;
        if (opt.value === val) o.selected = true;
        sel.appendChild(o);
      });
      wrapper.appendChild(lbl);
      wrapper.appendChild(sel);
      container.appendChild(wrapper);
    }
  }

  function refreshAllDropdowns() {
    var globalSel = document.querySelector('[name="token_style"]');
    if (globalSel) {
      var currentVal = globalSel.value;
      var toRemove = [];
      for (var i = 0; i < globalSel.options.length; i++) {
        if (globalSel.options[i].value.indexOf('custom:') === 0) toRemove.push(i);
      }
      for (var j = toRemove.length - 1; j >= 0; j--) globalSel.remove(toRemove[j]);
      customTokens.forEach(function(_, i) {
        var o = document.createElement('option');
        o.value = 'custom:' + i;
        o.text = 'Custom ' + (i + 1);
        globalSel.appendChild(o);
      });
      globalSel.value = currentVal;
    }
    if (document.getElementById('individual_styles').checked) updateSlotDropdowns();
  }

  function serializeCustomTokens() {
    var container = document.getElementById('custom-token-hidden-fields');
    container.innerHTML = '';
    customTokens.forEach(function(t, i) {
      var inp = document.createElement('input');
      inp.type = 'hidden';
      inp.name = 'custom_token_data_' + i;
      inp.value = t.base64;
      container.appendChild(inp);
    });
  }

  function showPendingPreview(dataURL) {
    document.getElementById('at_preview').innerHTML =
      '<img src="' + dataURL + '" style="max-height:60px;border:1px solid #ccc;border-radius:4px;margin-top:4px;">';
  }

  window.addCustomToken = function() {
    if (!pendingCustomImage) { alert('Choose a file or generate an image first.'); return; }
    customTokens.push(pendingCustomImage);
    pendingCustomImage = null;
    document.getElementById('at_preview').innerHTML = '';
    var f = document.getElementById('at_file');
    if (f) f.value = '';
    refreshAllDropdowns();
    serializeCustomTokens();
    var globalSel = document.querySelector('[name="token_style"]');
    if (globalSel) globalSel.value = 'custom:' + (customTokens.length - 1);
  };

  window.generateCustomToken = async function() {
    var prompt = document.getElementById('at_prompt').value.trim();
    if (!prompt) { alert('Enter a prompt first.'); return; }
    var btn = document.getElementById('at_gen_btn');
    var status = document.getElementById('at_gen_status');
    btn.disabled = true;
    status.textContent = 'Generating… (~10–30 sec)';
    try {
      var resp = await fetch('/api/generate-image', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({prompt: prompt})
      });
      if (!resp.ok) {
        var txt = await resp.text();
        throw new Error(txt || String(resp.status));
      }
      var contentType = resp.headers.get('Content-Type') || 'image/png';
      var buf = await resp.arrayBuffer();
      var bytes = new Uint8Array(buf);
      var binary = '';
      bytes.forEach(function(b) { binary += String.fromCharCode(b); });
      var b64 = btoa(binary);
      var urlB64 = b64.replace(/\+/g, '-').replace(/\//g, '_');
      var dataURL = 'data:' + contentType + ';base64,' + b64;
      pendingCustomImage = {dataURL: dataURL, base64: urlB64};
      showPendingPreview(dataURL);
      status.textContent = 'Done! Click “+ Add to Token List”.';
    } catch(e) {
      status.textContent = 'Error: ' + e.message;
    } finally {
      btn.disabled = false;
    }
  };

  window.rewardModeChange = function() {
    var upload = document.getElementById('ri_mode_upload').checked;
    document.getElementById('ri_upload_section').style.display = upload ? '' : 'none';
    document.getElementById('ri_ai_section').style.display = upload ? 'none' : '';
  };

  window.bgModeChange = function() {
    var ai = document.getElementById('bg_mode_ai').checked;
    document.getElementById('bg_ai_section').style.display = ai ? '' : 'none';
    document.getElementById('bg_upload_section').style.display = ai ? 'none' : '';
    if (!ai) {
      var p = document.querySelector('[name="background_prompt"]');
      if (p) p.value = '';
    }
  };

  window.addCustomTokenMode = function() {
    var upload = document.getElementById('at_mode_upload').checked;
    document.getElementById('at_upload_section').style.display = upload ? '' : 'none';
    document.getElementById('at_ai_section').style.display = upload ? 'none' : '';
  };

  window.toggleCustomToken = function(e) {
    e.preventDefault();
    var sec = document.getElementById('add-custom-token-section');
    sec.style.display = (sec.style.display === 'block') ? 'none' : 'block';
  };

  document.getElementById('individual_styles').addEventListener('change', updateSlotDropdowns);
  document.querySelector('[name="tokens"]').addEventListener('input', function() {
    if (document.getElementById('individual_styles').checked) updateSlotDropdowns();
  });

  document.getElementById('at_file').addEventListener('change', function(e) {
    var file = e.target.files[0];
    if (!file) return;
    var reader = new FileReader();
    reader.onload = function(ev) {
      var dataURL = ev.target.result;
      var b64Part = dataURL.split(',')[1];
      var urlB64 = b64Part.replace(/\+/g, '-').replace(/\//g, '_');
      pendingCustomImage = {dataURL: dataURL, base64: urlB64};
      showPendingPreview(dataURL);
    };
    reader.readAsDataURL(file);
  });

  document.querySelector('[name="reward"]').addEventListener('input', function() {
    document.getElementById('reward-error').style.display = 'none';
  });

  document.querySelector('form').addEventListener('submit', function(e) {
    var rewardText = document.querySelector('[name="reward"]').value.trim();
    var rewardData = document.querySelector('[name="reward_image_data"]').value;
    var riUpload = document.getElementById('ri_mode_upload');
    var rewardFile = riUpload && riUpload.checked && document.querySelector('[name="reward_image"]').files.length > 0;
    var riAI = document.getElementById('ri_mode_ai');
    var riPrompt = document.querySelector('[name="reward_image_prompt"]');
    var hasRewardAI = riAI && riAI.checked && riPrompt && riPrompt.value.trim();

    if (!rewardText && !rewardData && !rewardFile && !hasRewardAI) {
      e.preventDefault();
      document.getElementById('reward-error').style.display = '';
      document.getElementById('reward-input').focus();
      return;
    }

    var hasBgPrompt = document.querySelector('[name="background_prompt"]').value.trim();
    if (hasBgPrompt || hasRewardAI) {
      document.getElementById('ai-loading').style.display = 'flex';
    }
  });

  // Page-load initialization.
  (function init() {
    if (typeof _restoredTokens !== 'undefined' && _restoredTokens && _restoredTokens.length) {
      _restoredTokens.forEach(function(b64) {
        var stdB64 = b64.replace(/-/g, '+').replace(/_/g, '/');
        customTokens.push({dataURL: 'data:image/png;base64,' + stdB64, base64: b64});
      });
      refreshAllDropdowns();
      serializeCustomTokens();
      // Re-apply the server-side selected style now that custom:N options exist.
      if (typeof _restoredTokenStyle !== 'undefined' && _restoredTokenStyle) {
        var globalSel = document.querySelector('[name="token_style"]');
        if (globalSel) globalSel.value = _restoredTokenStyle;
      }
    }
    if (document.getElementById('individual_styles').checked) updateSlotDropdowns();

    // Restore background upload mode when there is image data but no prompt.
    var bgData = document.querySelector('[name="background_image_data"]');
    var bgPrompt = document.querySelector('[name="background_prompt"]');
    if (bgData && bgData.value && bgPrompt && !bgPrompt.value) {
      document.getElementById('bg_mode_upload').checked = true;
      window.bgModeChange();
    }

    // Restore reward AI gen mode.
    if (typeof _hasRewardPrompt !== 'undefined' && _hasRewardPrompt) {
      document.getElementById('ri_mode_ai').checked = true;
      window.rewardModeChange();
    }
  })();
})();
</script>
</body>
</html>`

// formData holds values for pre-populating the form template.
type formData struct {
	Name                string
	Reward              string
	Tokens              int
	TokenStyle          string
	IndividualStyles    bool   // whether per-slot customization is enabled
	TokenStylesData     string // comma-joined per-slot styles for JS pre-population
	Theme               string
	PageSize            string
	Title               string
	BackgroundPrompt    string
	BackgroundImageData string       // URL-safe base64 background image for hidden field passthrough
	BackgroundImageSrc  template.URL // data: URL for thumbnail preview
	RewardImageData     string       // URL-safe base64 reward image for hidden field passthrough
	RewardImageSrc      template.URL // data: URL for thumbnail preview
	RewardImagePrompt   string       // AI generation prompt for reward image
	CustomTokensJSON    template.JS  // JSON array of URL-safe base64 strings for JS restoration
	TokenStyleJSON      template.JS  // JS string literal of the current token style for restoration
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
<p>A Hugging Face API token is needed for AI-generated images. It is stored in memory for this session — you will need to re-enter it if you restart the app.</p>
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
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&display=swap" rel="stylesheet">
<style>
  *{box-sizing:border-box}
  body{font-family:'Nunito','Segoe UI',Arial,sans-serif;background:#F3F0FF;margin:0}
  .preview-header{background:linear-gradient(135deg,#5C6BC0,#7E57C2);color:white;padding:18px 24px;margin-bottom:20px;display:flex;align-items:center;justify-content:space-between}
  .preview-header h1{margin:0;font-size:20px;font-weight:800}
  .preview-header a{color:rgba(255,255,255,.85);font-size:13px;text-decoration:none}
  .preview-tagline{font-size:14px;color:rgba(255,255,255,.9);margin-top:4px}
  .board-wrap{display:flex;flex-direction:column;align-items:center;padding:0 12px 48px}
  .board{width:680px;max-width:100%;background:white;border:2px solid #aaa;box-shadow:0 4px 20px rgba(92,107,192,.20);overflow:hidden;border-radius:4px}
  .header{background:{{.Theme.HeaderBg}};color:{{.Theme.HeaderText}};padding:16px 20px;display:flex;align-items:center;justify-content:space-between;min-height:80px}
  .header-left{font-size:20px;font-weight:bold}
  .header-right{font-size:24px;font-weight:bold}
  .name-band{background:{{.Theme.NameBg}};color:{{.Theme.NameText}};text-align:center;padding:10px;font-size:22px;font-weight:bold}
  .name-band-placeholder{background:{{.Theme.NameBg}};text-align:center;padding:10px;font-size:14px;font-style:italic;color:{{.Theme.NameText}};opacity:.45}
  .token-row{background:{{.Theme.TokenBg}};display:flex;justify-content:center;align-items:center;gap:12px;padding:24px 16px}
  .token-slot{border:2px solid {{.Theme.TokenBorder}};border-radius:8px;background:{{.Theme.TokenBg}};display:flex;align-items:center;justify-content:center;min-width:44px;min-height:44px;flex-shrink:1}
  .footer{background:{{.Theme.FooterBg}};border-top:2px solid {{.Theme.FooterBorder}};height:80px;display:flex;align-items:center;justify-content:center}
  .footer-inner{border:2px dashed {{.Theme.FooterBorder}};width:calc(100% - 20px);height:60px;margin:0 10px;border-radius:4px;display:flex;align-items:center;justify-content:center}
  .footer-label{font-size:12px;color:{{.Theme.FooterBorder}};font-style:italic;opacity:.8}
  .token-note{font-size:12px;color:#7E57C2;text-align:center;margin-top:10px;opacity:.75}
  .action-card{background:white;border-radius:12px;padding:20px;margin-top:20px;width:680px;max-width:100%;box-shadow:0 1px 6px rgba(92,107,192,.10)}
  .action-bar{display:flex;gap:12px}
  .form-download{flex:1}
  .dl-btn{width:100%;padding:14px;background:linear-gradient(135deg,#2E7D32,#43A047);color:white;border:none;border-radius:10px;font-size:15px;font-weight:700;cursor:pointer;font-family:inherit}
  .dl-btn:hover{opacity:.92}
  .back-btn{width:100%;padding:14px 20px;background:white;color:#5C6BC0;border:2px solid #5C6BC0;border-radius:10px;font-size:15px;font-weight:700;cursor:pointer;white-space:nowrap;font-family:inherit}
  .back-btn:hover{background:#EDE7F6}
  .print-tip{margin-top:14px;font-size:13px;color:#5C6BC0;text-align:center;font-weight:600}
  @media(max-width:560px){.action-bar{flex-direction:column}.back-btn{white-space:normal}.board-wrap{padding:0 0 48px}.action-card{border-radius:0}}
  .action-bar{flex-wrap:wrap}
  .dl-loading{display:none;align-items:center;justify-content:center;gap:10px;margin-top:12px;color:#5C6BC0;font-size:13px;font-weight:600}
  @keyframes spin{to{transform:rotate(360deg)}}
  .dl-spinner{width:16px;height:16px;border:2px solid #D1C4E9;border-top-color:#5C6BC0;border-radius:50%;animation:spin .8s linear infinite}
</style>
<script>
  document.addEventListener('DOMContentLoaded', function() {
    document.querySelector('.form-download').addEventListener('submit', function() {
      var btn = this.querySelector('.dl-btn');
      btn.disabled = true;
      btn.textContent = 'Generating PDF…';
      document.querySelector('.dl-loading').style.display = 'flex';
    });
  });
</script>
</head>
<body>
<div class="preview-header">
  <div>
    <h1>Token Board Preview</h1>
    <div class="preview-tagline">Your board is ready — review below, then download</div>
  </div>
</div>
<div class="board-wrap">
  <div class="board"{{if .BackgroundImageSrc}} style="background-image:url({{.BackgroundImageSrc}});background-size:cover;background-position:center;"{{end}}>
    <div class="header"{{if .BackgroundImageSrc}} style="opacity:0.82;"{{end}}>
      <div class="header-left">{{.Title}}</div>
      <div class="header-right">{{if .RewardImageSrc}}<img src="{{.RewardImageSrc}}" style="max-height:80px;max-width:200px;object-fit:contain;">{{else}}{{.RewardText}}{{end}}</div>
    </div>
    {{if .HasName}}<div class="name-band"{{if $.BackgroundImageSrc}} style="opacity:0.82;"{{end}}>{{.ChildName}}</div>{{else}}<div class="name-band-placeholder">Child name — optional, add it in the form</div>{{end}}
    <div class="token-row"{{if .BackgroundImageSrc}} style="opacity:0.82;"{{end}}>
      {{range .Tokens}}<div class="token-slot" style="width:{{$.SlotSize}}px;height:{{$.SlotSize}}px;">{{.}}</div>{{end}}
    </div>
    <div class="footer"{{if .BackgroundImageSrc}} style="opacity:0.82;"{{end}}><div class="footer-inner"><span class="footer-label">Reward image or teacher sign-off</span></div></div>
  </div>
  <div class="action-card">
  <p class="token-note" style="margin:0 0 16px;font-size:14px;color:#5C6BC0;font-weight:600;">Tip: each slot shows the token style that will print — children add stickers or stamps as they earn each token.</p>
  <div class="action-bar">
    <form method="POST" action="/generate" class="form-download">
      <input type="hidden" name="name" value="{{.ChildName}}">
      <input type="hidden" name="reward" value="{{.RewardText}}">
      <input type="hidden" name="tokens" value="{{.TokenCount}}">
      <input type="hidden" name="token_style" value="{{.TokenStyle}}">
      {{if .IndividualStyles}}<input type="hidden" name="individual_styles" value="on">
      <input type="hidden" name="token_styles" value="{{.TokenStylesCSV}}">{{end}}
      <input type="hidden" name="theme" value="{{.ThemeName}}">
      <input type="hidden" name="page_size" value="{{.PageSize}}">
      <input type="hidden" name="title" value="{{.Title}}">
      <input type="hidden" name="background_prompt" value="{{.BackgroundPrompt}}">
      <input type="hidden" name="reward_image_data" value="{{.RewardImageData}}">
      <input type="hidden" name="token_image_data" value="{{.TokenImageData}}">
      <input type="hidden" name="background_image_data" value="{{.BackgroundImageData}}">
      {{range $i, $v := .CustomTokenDataFields}}<input type="hidden" name="custom_token_data_{{$i}}" value="{{$v}}">{{end}}
      <button type="submit" class="dl-btn">&#8595; Download PDF</button>
    </form>
    <form method="POST" action="/">
      <input type="hidden" name="name" value="{{.ChildName}}">
      <input type="hidden" name="reward" value="{{.RewardText}}">
      <input type="hidden" name="tokens" value="{{.TokenCount}}">
      <input type="hidden" name="token_style" value="{{.TokenStyle}}">
      {{if .IndividualStyles}}<input type="hidden" name="individual_styles" value="on">
      <input type="hidden" name="token_styles" value="{{.TokenStylesCSV}}">{{end}}
      <input type="hidden" name="theme" value="{{.ThemeName}}">
      <input type="hidden" name="page_size" value="{{.PageSize}}">
      <input type="hidden" name="title" value="{{.Title}}">
      <input type="hidden" name="background_prompt" value="{{.BackgroundPrompt}}">
      <input type="hidden" name="reward_image_data" value="{{.RewardImageData}}">
      <input type="hidden" name="token_image_data" value="{{.TokenImageData}}">
      <input type="hidden" name="background_image_data" value="{{.BackgroundImageData}}">
      <input type="hidden" name="reward_image_prompt" value="{{.RewardImagePrompt}}">
      {{range $i, $v := .CustomTokenDataFields}}<input type="hidden" name="custom_token_data_{{$i}}" value="{{$v}}">{{end}}
      <button type="submit" class="back-btn">&#8592; Edit Board</button>
    </form>
  </div>
  <div class="dl-loading"><div class="dl-spinner"></div>Generating your PDF — this takes a moment…</div>
  <p class="print-tip">&#10003; Board ready — click Download PDF to save, then print at home or school.</p>
  </div>
</div>
</body>
</html>`

var previewTmpl = template.Must(template.New("preview").Parse(previewHTML))

// tokenEmoji maps builtin styles to a display emoji for the HTML preview.
var tokenSVG = map[string]template.HTML{
	"star":         `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg"><polygon points="12,2 15.09,8.26 22,9.27 17,14.14 18.18,21.02 12,17.77 5.82,21.02 7,14.14 2,9.27 8.91,8.26" fill="#FFD700" stroke="#F9A825" stroke-width="1" stroke-linejoin="round"/></svg>`,
	"circle":       `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg"><circle cx="12" cy="12" r="10" fill="#4FC3F7" stroke="#0288D1" stroke-width="1.5"/></svg>`,
	"smiley":       `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg"><circle cx="12" cy="12" r="10" fill="#FFD54F" stroke="#F57F17" stroke-width="1"/><circle cx="9" cy="10" r="1.5" fill="#4A3728"/><circle cx="15" cy="10" r="1.5" fill="#4A3728"/><path d="M8.5 14 Q12 17.5 15.5 14" stroke="#4A3728" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg>`,
	"thumbsup":     `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg" fill="none"><path d="M14 9V5a3 3 0 0 0-3-3l-4 9v11h11.28a2 2 0 0 0 2-1.7l1.38-9a2 2 0 0 0-2-2.3H14z" fill="#66BB6A" stroke="#388E3C" stroke-width="1.2"/><path d="M7 22H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2h3" stroke="#388E3C" stroke-width="1.2" fill="#A5D6A7"/></svg>`,
	"png:star":     `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg"><polygon points="12,2 15.09,8.26 22,9.27 17,14.14 18.18,21.02 12,17.77 5.82,21.02 7,14.14 2,9.27 8.91,8.26" fill="#FFD700" stroke="#F9A825" stroke-width="1" stroke-linejoin="round"/></svg>`,
	"png:smiley":   `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg"><circle cx="12" cy="12" r="10" fill="#FFD54F" stroke="#F57F17" stroke-width="1"/><circle cx="9" cy="10" r="1.5" fill="#4A3728"/><circle cx="15" cy="10" r="1.5" fill="#4A3728"/><path d="M8.5 14 Q12 17.5 15.5 14" stroke="#4A3728" stroke-width="1.5" fill="none" stroke-linecap="round"/></svg>`,
	"png:thumbsup": `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg" fill="none"><path d="M14 9V5a3 3 0 0 0-3-3l-4 9v11h11.28a2 2 0 0 0 2-1.7l1.38-9a2 2 0 0 0-2-2.3H14z" fill="#66BB6A" stroke="#388E3C" stroke-width="1.2"/><path d="M7 22H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2h3" stroke="#388E3C" stroke-width="1.2" fill="#A5D6A7"/></svg>`,
}

// previewData is the data model for the HTML preview template.
type previewData struct {
	Title                 string
	RewardText            string
	ChildName             string
	HasName               bool
	Tokens                []template.HTML
	TokenCount            int
	SlotSize              int
	TokenStyle            string
	IndividualStyles      bool   // whether per-slot customization is enabled
	TokenStylesCSV        string // comma-joined per-slot styles for hidden field passthrough
	ThemeName             string
	PageSize              string
	Theme                 previewTheme
	BackURL               string
	BackgroundPrompt      string
	RewardImageData       string       // URL-safe base64 reward image (for hidden field passthrough)
	RewardImageSrc        template.URL // data: URL for <img src> preview
	RewardImagePrompt     string       // AI generation prompt for reward image (for round-trip)
	TokenImageData        string       // URL-safe base64 custom token image (for hidden field passthrough)
	BackgroundImageData   string       // URL-safe base64 background image (for hidden field passthrough)
	BackgroundImageSrc    template.URL // data: URL for CSS background display in preview
	CustomTokenDataFields []string     // URL-safe base64 strings, one per custom:N token
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
	mux.HandleFunc("POST /api/generate-image", handleGenerateImage)

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

// handleGenerateImage generates an image from a text prompt via the Hugging Face API.
func handleGenerateImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "prompt required", http.StatusBadRequest)
		return
	}
	apiToken := os.Getenv("HF_TOKEN")
	if apiToken == "" {
		apiToken = getStoredHFToken()
	}
	if apiToken == "" {
		http.Error(w, "no Hugging Face API token configured — go to Settings", http.StatusUnauthorized)
		return
	}
	imgBytes, err := imagegen.Generate(r.Context(), req.Prompt, apiToken)
	if err != nil {
		http.Error(w, "image generation failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	mime := http.DetectContentType(imgBytes)
	w.Header().Set("Content-Type", mime)
	w.Write(imgBytes)
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
		theme = "blue"
	}
	pageSize := r.FormValue("page_size")
	if pageSize == "" {
		pageSize = "letter"
	}
	fd := formData{
		Name:              r.FormValue("name"),
		Reward:            r.FormValue("reward"),
		Tokens:            tokens,
		TokenStyle:        tokenStyle,
		IndividualStyles:  r.FormValue("individual_styles") == "on",
		TokenStylesData:   r.FormValue("token_styles"),
		Theme:             theme,
		PageSize:          pageSize,
		Title:             r.FormValue("title"),
		BackgroundPrompt:  r.FormValue("background_prompt"),
		RewardImagePrompt: r.FormValue("reward_image_prompt"),
	}
	// Restore uploaded/generated images when navigating back from preview.
	fd.RewardImageData = r.FormValue("reward_image_data")
	if fd.RewardImageData != "" {
		if b, err := base64.URLEncoding.DecodeString(fd.RewardImageData); err == nil && len(b) > 0 {
			fd.RewardImageSrc = imageDataURL(b)
		}
	}
	fd.BackgroundImageData = r.FormValue("background_image_data")
	if fd.BackgroundImageData != "" {
		if b, err := base64.URLEncoding.DecodeString(fd.BackgroundImageData); err == nil && len(b) > 0 {
			fd.BackgroundImageSrc = imageDataURL(b)
		}
	}
	// Restore custom token images for JS re-initialization.
	var customTokens []string
	for i := 0; ; i++ {
		v := r.FormValue(fmt.Sprintf("custom_token_data_%d", i))
		if v == "" {
			break
		}
		customTokens = append(customTokens, v)
	}
	b, _ := json.Marshal(customTokens)
	fd.CustomTokensJSON = template.JS(b)
	tsJSON, _ := json.Marshal(fd.TokenStyle)
	fd.TokenStyleJSON = template.JS(tsJSON)

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

	// Build token display — custom:N styles show as inline images.
	tokens := make([]template.HTML, cfg.TokenCount)
	for i := range tokens {
		style := cfg.TokenStyle
		if len(cfg.TokenStyles) == cfg.TokenCount {
			style = cfg.TokenStyles[i]
		}
		if idx, ok := board.CustomStyleIndex(style); ok {
			if idx < len(cfg.CustomTokenImages) && len(cfg.CustomTokenImages[idx]) > 0 {
				src := imageDataURL(cfg.CustomTokenImages[idx])
				tokens[i] = template.HTML(`<img src="` + string(src) + `" style="max-width:100%;max-height:100%;">`)
			} else {
				tokens[i] = template.HTML("⬜")
			}
		} else {
			svg, ok := tokenSVG[style]
			if !ok {
				svg = `<svg viewBox="0 0 24 24" width="58%" height="58%" xmlns="http://www.w3.org/2000/svg"><rect x="3" y="3" width="18" height="18" rx="3" fill="none" stroke="#ccc" stroke-width="2"/></svg>`
			}
			tokens[i] = svg
		}
	}

	indStyles := len(cfg.TokenStyles) == cfg.TokenCount
	var tokenStylesCSV string
	if indStyles {
		tokenStylesCSV = strings.Join(cfg.TokenStyles, ",")
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

	// Resolve reward image — upload, cached data, or AI generation.
	rewardImgBytes, _ := resolveImageData(r, "reward_image", "reward_image_data")
	if len(rewardImgBytes) == 0 {
		if prompt := strings.TrimSpace(r.FormValue("reward_image_prompt")); prompt != "" {
			apiToken := os.Getenv("HF_TOKEN")
			if apiToken == "" {
				apiToken = getStoredHFToken()
			}
			if apiToken == "" {
				writeHTMLError(w, http.StatusBadRequest,
					"No Hugging Face API token configured.\nGo to Settings (top-right) to enter your token.\nGet a free token at https://huggingface.co/settings/tokens")
				return
			}
			generated, err := imagegen.Generate(r.Context(), prompt, apiToken)
			if err != nil {
				writeHTMLError(w, http.StatusBadGateway, "Reward image generation failed: "+err.Error())
				return
			}
			rewardImgBytes = generated
		}
	}
	rewardImgData := base64.URLEncoding.EncodeToString(rewardImgBytes)
	var rewardImgSrc template.URL
	if len(rewardImgBytes) > 0 {
		rewardImgSrc = imageDataURL(rewardImgBytes)
	}

	tokenImgBytes, _ := resolveImageData(r, "token_image", "token_image_data")
	tokenImgData := base64.URLEncoding.EncodeToString(tokenImgBytes)

	// Resolve background image — upload, cached data, or AI generation.
	var bgImgData string
	var bgImgSrc template.URL
	bgBytes, _ := resolveImageData(r, "background_image", "background_image_data")
	if len(bgBytes) == 0 && cfg.BackgroundPrompt != "" {
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
	if len(bgBytes) > 0 {
		bgImgData = base64.URLEncoding.EncodeToString(bgBytes)
		bgImgSrc = imageDataURL(bgBytes)
	}

	// Build CustomTokenDataFields for hidden field passthrough.
	var customTokenDataFields []string
	for _, b := range cfg.CustomTokenImages {
		customTokenDataFields = append(customTokenDataFields, base64.URLEncoding.EncodeToString(b))
	}

	data := previewData{
		Title:                 cfg.Title,
		RewardText:            cfg.RewardText,
		ChildName:             cfg.ChildName,
		HasName:               cfg.ChildName != "",
		Tokens:                tokens,
		TokenCount:            cfg.TokenCount,
		SlotSize:              slotSize,
		TokenStyle:            cfg.TokenStyle,
		IndividualStyles:      indStyles,
		TokenStylesCSV:        tokenStylesCSV,
		ThemeName:             cfg.Theme,
		PageSize:              cfg.PageSize,
		Theme:                 themeData,
		BackURL:               "/?" + backParams.Encode(),
		BackgroundPrompt:      cfg.BackgroundPrompt,
		RewardImageData:       rewardImgData,
		RewardImageSrc:        rewardImgSrc,
		RewardImagePrompt:     r.FormValue("reward_image_prompt"),
		TokenImageData:        tokenImgData,
		BackgroundImageData:   bgImgData,
		BackgroundImageSrc:    bgImgSrc,
		CustomTokenDataFields: customTokenDataFields,
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

	if r.FormValue("background_image_data") != "" {
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
	if err := r.ParseMultipartForm(32 << 20); err != nil && err != http.ErrNotMultipart {
		return board.Config{}, fmt.Errorf("parsing form: %w", err)
	}

	tokens, _ := strconv.Atoi(r.FormValue("tokens"))
	individualStyles := r.FormValue("individual_styles") == "on"
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

	// Sentinels let Validate() pass when images are provided via upload or AI gen prompt.
	if r.FormValue("reward_image_data") != "" || hasUpload(r, "reward_image") || r.FormValue("reward_image_prompt") != "" {
		cfg.RewardImage = "uploaded"
	}
	// Only apply custom token image when individual styles are not active (legacy "custom" path).
	if !individualStyles && (hasUpload(r, "token_image") || (r.FormValue("token_style") == "custom" && r.FormValue("token_image_data") != "")) {
		cfg.TokenStyle = "custom"
	}

	// Parse custom token images from hidden fields (indexed by position = custom:N index).
	for i := 0; ; i++ {
		val := r.FormValue(fmt.Sprintf("custom_token_data_%d", i))
		if val == "" {
			break
		}
		b, err := base64.URLEncoding.DecodeString(val)
		if err == nil && len(b) > 0 {
			cfg.CustomTokenImages = append(cfg.CustomTokenImages, b)
		} else {
			cfg.CustomTokenImages = append(cfg.CustomTokenImages, nil)
		}
	}

	// Read per-slot styles when individual customization is enabled.
	if individualStyles {
		n := tokens
		if r.FormValue("token_style_1") != "" {
			// Individual slot fields from main form submission.
			styles := make([]string, n)
			for i := range styles {
				s := r.FormValue(fmt.Sprintf("token_style_%d", i+1))
				if s == "" {
					s = cfg.TokenStyle
				}
				styles[i] = s
			}
			cfg.TokenStyles = styles
		} else if ts := r.FormValue("token_styles"); ts != "" {
			// Comma-joined passthrough from preview hidden field.
			parts := strings.Split(ts, ",")
			if len(parts) == n {
				cfg.TokenStyles = parts
			}
		}
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
