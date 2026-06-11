---
name: diff-view
description: >
  Generate a styled dark-theme HTML diff page from git diff output and open it in the browser.
  Triggers on "diff view", "show diff", "open diff", "view diff", "diff page", "render diff", or /diff-view.
  Also triggers on "show me the changes", "visualize the diff", "html diff", or "diff in browser".
allowed-tools: Bash, Write, Read
---

# Diff View

Generate a dark-theme HTML diff page (GitHub-style dark) from `git diff` output and open it in the browser at `/tmp/teetime_diff_latest.html`.

## Arguments

- **ref range** (optional): A git ref range like `HEAD~1..HEAD`, a branch name, or a commit hash. Defaults to uncommitted working tree changes (`git diff HEAD`).
- **risks** (optional): Structured risk annotations to render at the bottom of each file block. Format: `Major: <text> | Minor: <text> | Minimal: <text>`.

## Execution

### 1. Determine the diff source

Parse the invocation for a ref range argument. If none provided, default to `git diff HEAD` (working tree vs last commit).

Run the diff:
```bash
# Working tree (default)
git diff HEAD

# Specific range
git diff <ref-range>

# Specific file(s) can also be appended: git diff HEAD -- path/to/file
```

If the diff is empty, report "no changes found" and stop.

### 2. Write the Python script to /tmp

Write the following script to `/tmp/teetime_diff_gen.py`, substituting the actual diff content and any risk annotations.

The script is embedded inline — write it to `/tmp/teetime_diff_gen.py` and execute it with `python3`.

```python
#!/usr/bin/env python3
"""Generates a dark-theme GitHub-style HTML diff page."""

import sys
import re
import subprocess

DIFF_TEXT = """__DIFF_PLACEHOLDER__"""

RISKS = __RISKS_PLACEHOLDER__  # dict: {"Major": "...", "Minor": "...", "Minimal": "..."}

OUTPUT = "/tmp/teetime_diff_latest.html"

# ── colours matching GitHub dark diff ────────────────────────────────────────
BG         = "#0d1117"
FG         = "#e6edf3"
BORDER     = "#30363d"
FILE_BG    = "#161b22"
FILE_BADGE = "#1f6feb"
HUNK_BG    = "#0d1b2a"
HUNK_FG    = "#6cb6ff"
ADD_BG     = "#0d2a12"
ADD_FG     = "#aff5b4"
DEL_BG     = "#2a0d0d"
DEL_FG     = "#ffa198"
CTX_FG     = "#8b949e"
LNUM_FG    = "#484f58"
RISK_MAJOR = "#da3633"
RISK_MINOR = "#d29922"
RISK_MIN   = "#238636"
# ─────────────────────────────────────────────────────────────────────────────


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def parse_diff(text):
    """Return list of file blocks: {filename, hunks: [{header, lines: [(type, old, new, text)]}]}"""
    files = []
    cur_file = None
    cur_hunk = None
    old_ln = new_ln = 0

    for raw_line in text.splitlines():
        # new file
        m = re.match(r'^diff --git a/(.+?) b/(.+)$', raw_line)
        if m:
            if cur_file:
                if cur_hunk:
                    cur_file["hunks"].append(cur_hunk)
                files.append(cur_file)
            cur_file = {"filename": m.group(2), "hunks": []}
            cur_hunk = None
            continue

        if cur_file is None:
            continue

        # hunk header  @@ -a,b +c,d @@
        m = re.match(r'^(@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@.*)', raw_line)
        if m:
            if cur_hunk:
                cur_file["hunks"].append(cur_hunk)
            old_ln = int(m.group(2))
            new_ln = int(m.group(3))
            cur_hunk = {"header": m.group(1), "lines": []}
            continue

        if cur_hunk is None:
            continue

        if raw_line.startswith('+') and not raw_line.startswith('+++'):
            cur_hunk["lines"].append(("add", None, new_ln, raw_line[1:]))
            new_ln += 1
        elif raw_line.startswith('-') and not raw_line.startswith('---'):
            cur_hunk["lines"].append(("del", old_ln, None, raw_line[1:]))
            old_ln += 1
        elif raw_line.startswith('\\'):
            pass  # "\ No newline at end of file"
        else:
            # context
            cur_hunk["lines"].append(("ctx", old_ln, new_ln, raw_line[1:] if raw_line else ""))
            old_ln += 1
            new_ln += 1

    if cur_file:
        if cur_hunk:
            cur_file["hunks"].append(cur_hunk)
        files.append(cur_file)

    return files


def render_line(ltype, old_ln, new_ln, text):
    if ltype == "add":
        row_bg = ADD_BG
        gutter_bg = "#0e3a16"
        marker = "+"
        ln_old = ""
        ln_new = str(new_ln)
        text_color = ADD_FG
    elif ltype == "del":
        row_bg = DEL_BG
        gutter_bg = "#3a0e0e"
        marker = "−"
        ln_old = str(old_ln)
        ln_new = ""
        text_color = DEL_FG
    else:
        row_bg = "transparent"
        gutter_bg = "transparent"
        marker = " "
        ln_old = str(old_ln) if old_ln else ""
        ln_new = str(new_ln) if new_ln else ""
        text_color = CTX_FG

    return (
        f'<tr style="background:{row_bg}">'
        f'<td class="ln" style="background:{gutter_bg};color:{LNUM_FG}">{esc(ln_old)}</td>'
        f'<td class="ln" style="background:{gutter_bg};color:{LNUM_FG}">{esc(ln_new)}</td>'
        f'<td class="marker" style="background:{gutter_bg};color:{text_color}">{marker}</td>'
        f'<td class="code" style="color:{text_color}">{esc(text)}</td>'
        f'</tr>\n'
    )


def render_risks(risks):
    if not risks:
        return ""
    items = []
    for level, color in [("Major", RISK_MAJOR), ("Minor", RISK_MINOR), ("Minimal", RISK_MIN)]:
        if risks.get(level):
            items.append(
                f'<div class="risk-item">'
                f'<span class="risk-badge" style="background:{color}">{level}</span>'
                f'<span class="risk-text">{esc(risks[level])}</span>'
                f'</div>'
            )
    if not items:
        return ""
    return '<div class="risk-block">' + "".join(items) + "</div>"


def build_html(files, risks):
    stat_adds = sum(
        1 for f in files for h in f["hunks"] for ln in h["lines"] if ln[0] == "add"
    )
    stat_dels = sum(
        1 for f in files for h in f["hunks"] for ln in h["lines"] if ln[0] == "del"
    )

    css = f"""
* {{ box-sizing: border-box; margin: 0; padding: 0; }}
body {{ background: {BG}; color: {FG}; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; font-size: 13px; padding: 24px; }}
.page-header {{ margin-bottom: 20px; }}
.page-header h1 {{ font-size: 1rem; font-weight: 600; color: {FG}; margin-bottom: 4px; }}
.stats {{ font-size: .82rem; color: {CTX_FG}; }}
.stats .adds {{ color: {ADD_FG}; font-weight: 600; }}
.stats .dels {{ color: {DEL_FG}; font-weight: 600; }}
.file-block {{ border: 1px solid {BORDER}; border-radius: 6px; margin-bottom: 20px; overflow: hidden; }}
.file-header {{ background: {FILE_BG}; padding: 8px 14px; display: flex; align-items: center; gap: 10px; border-bottom: 1px solid {BORDER}; }}
.file-badge {{ background: {FILE_BADGE}; color: #fff; border-radius: 3px; padding: 1px 7px; font-size: .75rem; font-weight: 600; letter-spacing: .03em; }}
.file-name {{ font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace; font-size: .82rem; color: {FG}; }}
.diff-table {{ width: 100%; border-collapse: collapse; font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace; font-size: .8rem; tab-size: 4; }}
.diff-table td {{ padding: 1px 6px; vertical-align: top; white-space: pre; }}
td.ln {{ width: 44px; min-width: 44px; text-align: right; user-select: none; border-right: 1px solid {BORDER}; padding-right: 10px; }}
td.marker {{ width: 20px; text-align: center; user-select: none; border-right: 1px solid {BORDER}; }}
td.code {{ width: 100%; }}
.hunk-header {{ background: {HUNK_BG}; color: {HUNK_FG}; padding: 4px 12px; font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace; font-size: .78rem; border-top: 1px solid {BORDER}; border-bottom: 1px solid {BORDER}; }}
.hunk-header:first-child {{ border-top: none; }}
.risk-block {{ background: {FILE_BG}; border-top: 1px solid {BORDER}; padding: 10px 14px; display: flex; flex-direction: column; gap: 6px; }}
.risk-item {{ display: flex; align-items: flex-start; gap: 10px; }}
.risk-badge {{ border-radius: 3px; padding: 1px 7px; font-size: .72rem; font-weight: 700; color: #fff; white-space: nowrap; margin-top: 1px; }}
.risk-text {{ font-size: .82rem; color: {FG}; }}
"""

    blocks = []
    for f in files:
        rows = []
        for h in f["hunks"]:
            rows.append(f'<div class="hunk-header">{esc(h["header"])}</div>\n')
            rows.append('<table class="diff-table">\n')
            for ltype, old_ln, new_ln, text in h["lines"]:
                rows.append(render_line(ltype, old_ln, new_ln, text))
            rows.append("</table>\n")

        risk_html = render_risks(risks.get(f["filename"], risks.get("*", {})))

        blocks.append(
            f'<div class="file-block">'
            f'<div class="file-header">'
            f'<span class="file-badge">diff</span>'
            f'<span class="file-name">{esc(f["filename"])}</span>'
            f'</div>'
            + "".join(rows)
            + risk_html
            + "</div>"
        )

    adds_html = f'<span class="adds">+{stat_adds}</span>'
    dels_html = f'<span class="dels">−{stat_dels}</span>'

    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Diff — {esc(files[0]["filename"] if len(files)==1 else f"{len(files)} files")}</title>
<style>{css}</style>
</head>
<body>
<div class="page-header">
  <h1>Code Diff</h1>
  <p class="stats">{len(files)} file{"s" if len(files)!=1 else ""} changed &nbsp;·&nbsp; {adds_html} additions &nbsp;·&nbsp; {dels_html} deletions</p>
</div>
{"".join(blocks)}
</body>
</html>"""


def main():
    diff_text = DIFF_TEXT
    if not diff_text.strip():
        print("No diff content — nothing to render.", file=sys.stderr)
        sys.exit(1)

    files = parse_diff(diff_text)
    if not files:
        print("Could not parse any file diffs.", file=sys.stderr)
        sys.exit(1)

    html = build_html(files, RISKS)
    with open(OUTPUT, "w", encoding="utf-8") as fh:
        fh.write(html)
    print(f"Written to {OUTPUT}")


if __name__ == "__main__":
    main()
```

### 3. Execution steps

Follow these exact steps when the skill is invoked:

**Step 1** — Get the diff text. Run the appropriate git diff command based on the ref range argument (default: `git diff HEAD`). Capture the output.

**Step 2** — Check for emptiness. If the diff output is empty, report "No changes found for `<ref range>`" and stop.

**Step 3** — Write the Python script. Take the embedded script above and:
  - Replace `__DIFF_PLACEHOLDER__` with the actual diff text (escape any `"""` sequences in the diff as `\"""`)
  - Replace `__RISKS_PLACEHOLDER__` with a Python dict literal for the risks. Use `{"*": {"Major": "...", "Minor": "...", "Minimal": "..."}}` to apply risks to all files, or `{}` if no risks were provided. The `"*"` key applies to every file block.

  Write the result to `/tmp/teetime_diff_gen.py`.

**Step 4** — Execute: `python3 /tmp/teetime_diff_gen.py`

**Step 5** — Open: `xdg-open /tmp/teetime_diff_latest.html`

**Step 6** — Report the output path and stat summary (files changed, additions, deletions).

### 4. Risk annotations

If the user provides risk text (inline with the invocation or in a follow-up), structure it as:

```python
{"*": {
    "Major": "text for major risk",
    "Minor": "text for minor risk",
    "Minimal": "text for minimal risk"
}}
```

Only include keys that have content. An empty dict `{}` means no risk section is rendered.

Per-file risks can be provided with the filename as key instead of `"*"`:
```python
{
    "internal/display/web.go": {"Major": "Template injection surface area"},
    "main.go": {"Minimal": "Flag parsing only"}
}
```

### 5. Example invocations

```
/diff-view
/diff-view HEAD~1..HEAD
/diff-view main..feature/my-branch
/diff-view HEAD~3..HEAD risks: Major: Changes auth flow Minor: Adds new dependency
```

## Notes

- The script is always re-written to `/tmp/teetime_diff_gen.py` on each run so the diff content is current.
- Output always overwrites `/tmp/teetime_diff_latest.html` — only the latest diff is kept.
- The Python script uses only stdlib — no pip installs required.
- Tab characters in source code are preserved via `white-space: pre` and `tab-size: 4`.
- The hunk header (`@@ -a,b +c,d @@`) is rendered as a separator row between hunk groups within a file.
