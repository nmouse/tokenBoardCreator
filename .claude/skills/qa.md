---
name: qa
description: >
  Full QA workflow for the tokenBoardCreator web UI: build and run the app, drive the browser
  through every form control and flow, verify PDF generation, document findings, spawn agents
  for approved fixes, review diffs with risk annotations, and merge to main when clean.
  Triggers on "qa", "run qa", "qa workflow", "test the app", "quality check", or /qa.
allowed-tools: Bash, Write, Read, Task, AskUserQuestion, mcp__chrome-devtools__navigate_page, mcp__chrome-devtools__take_screenshot, mcp__chrome-devtools__take_snapshot, mcp__chrome-devtools__click, mcp__chrome-devtools__fill, mcp__chrome-devtools__select_page, mcp__chrome-devtools__list_pages, mcp__chrome-devtools__new_page, mcp__chrome-devtools__list_network_requests, mcp__chrome-devtools__evaluate_script, mcp__chrome-devtools__wait_for, mcp__chrome-devtools__close_page
---

# QA Workflow — tokenBoardCreator

End-to-end QA for the tokenBoardCreator web UI. Builds and runs the app, drives Chrome through
every form control, validates preview rendering and PDF download, documents all findings,
iterates fixes through agents with diff review, and merges to main when no Major risks remain.

## Arguments

None required. The app always runs on `localhost:8080`.

---

## Phase 1 — Build and Run

### Step 1.1 — Build

```bash
cd /home/owner/GolandProjects/tokenBoardCreator && go build .
```

If the build fails, report the error and stop.

### Step 1.2 — Run in web mode

```bash
cd /home/owner/GolandProjects/tokenBoardCreator && ./tokenBoardCreator --web --port 8080 &
sleep 1
```

Watch for:
```
Token Board Creator listening on http://localhost:8080
```

If it does not appear within 10 seconds, report a timeout and stop.

### Step 1.3 — Open the browser

Use `mcp__chrome-devtools__new_page` with URL `http://localhost:8080`. If a page already exists,
use `mcp__chrome-devtools__navigate_page` instead.

Take a screenshot and note the initial state.

---

## Phase 2 — UI Testing

Work through each test below in order. For each test, take a screenshot after the interaction
and record: **PASS**, **FAIL**, or **NOTE** with a one-line observation.

Use `mcp__chrome-devtools__take_snapshot` to get element UIDs before filling or clicking.

---

### 2.1 — Form loads with correct defaults

Navigate to `http://localhost:8080/`. Verify:

| Field | Expected default |
|---|---|
| Child Name | empty |
| Reward Text | empty |
| Number of Tokens | `5` |
| Token Style | `Star` |
| Theme | `Default` |
| Page Size | `Letter` |
| Custom Title | empty (placeholder: "I am working for:") |

Use `mcp__chrome-devtools__evaluate_script` to read values:
```javascript
({
  name:       document.querySelector('[name=name]').value,
  reward:     document.querySelector('[name=reward]').value,
  tokens:     document.querySelector('[name=tokens]').value,
  tokenStyle: document.querySelector('[name=token_style]').value,
  theme:      document.querySelector('[name=theme]').value,
  pageSize:   document.querySelector('[name=page_size]').value,
  title:      document.querySelector('[name=title]').value,
})
```

Record PASS if all match, FAIL with specifics if any diverge.

---

### 2.2 — Required field: empty reward text

Without filling any fields, click **Preview**. Verify:
- The browser blocks submission and shows a "Please fill out this field." tooltip on Reward Text.
- The page does NOT navigate to `/preview`.

Record PASS if the form stays on `/` with the validation tooltip visible.

---

### 2.3 — Token count: below minimum

Fill Reward Text with any value. Set Number of Tokens to `2`. Click **Preview**. Verify:
- The browser blocks submission and shows "Value must be greater than or equal to 3."
- The page does NOT navigate.

---

### 2.4 — Token count: above maximum

Set Number of Tokens to `11`. Click **Preview**. Verify:
- The browser blocks submission and shows "Value must be less than or equal to 10."
- The page does NOT navigate.

---

### 2.5 — Happy path: preview renders all fields

Fill the form:
- Child Name: `Alex`
- Reward Text: `iPad time`
- Tokens: `5`
- Token Style: `Star`
- Theme: `Default`
- Page Size: `Letter`
- Custom Title: *(leave empty)*

Click **Preview**. Verify the preview page shows:
- Title "I am working for:" in the header left
- "iPad time" in the header right
- Name band with "Alex"
- Exactly 5 token slots
- Footer with dashed border
- **Download PDF** button
- **← Back to form** link

Record PASS if all elements are present and correctly placed.

---

### 2.6 — Happy path: Download PDF from preview

From the preview page (after 2.5), click **Download PDF**. Verify:
- The page does NOT navigate away (stays on `/preview`)
- A POST to `/generate` returns HTTP 200

Check with `mcp__chrome-devtools__list_network_requests` and confirm the `/generate` response
has status 200. Record PASS if 200, FAIL otherwise.

---

### 2.7 — No child name: name band hides

Fill form with Reward Text only (leave Child Name empty), tokens 5, click **Preview**. Verify:
- No name band is rendered between the header and token row.

Use `evaluate_script`:
```javascript
getComputedStyle(document.querySelector('.name-band')).display
```

Record PASS if `display` is `none` or the element is absent.

---

### 2.8 — Custom title renders

Fill Child Name: `Sam`, Reward Text: `Movie night`, Custom Title: `You are doing great!`. Click **Preview**. Verify:
- Header left shows "You are doing great!" (not the default "I am working for:")

---

### 2.9 — All four themes apply colors

For each theme (`Default`, `Blue`, `Green`, `Pink`):
1. Navigate back to `/`
2. Fill Reward Text with any value
3. Select the theme
4. Click **Preview**
5. Take a screenshot

Verify the header and footer visually reflect distinct theme colors (Blue and Pink are the
easiest to spot). Record a NOTE if any theme looks identical to Default unexpectedly.

---

### 2.10 — All token styles render in preview

For each token style (`Star`, `Circle`, `Smiley`, `Thumbs Up`, `PNG Star`, `PNG Smiley`, `PNG Thumbs Up`):
1. Navigate to `/`
2. Fill Reward Text with any value
3. Select the style
4. Click **Preview**

Use `mcp__chrome-devtools__take_snapshot` to confirm the token slot text matches the expected emoji
for each style (Star/PNG Star → ⭐, Circle → ⬤, Smiley/PNG Smiley → 😊, Thumbs Up/PNG Thumbs Up → 👍).

Record PASS if all 7 styles produce slots with non-empty content. Record FAIL for any that show ⬜.

---

### 2.11 — Max tokens (10): single-row vs wrap

Fill Reward Text with any value, set Tokens to `10`, click **Preview**.

Verify whether the 10 token slots appear in a single row or wrap to multiple rows. The PDF
always renders a single row; record a NOTE if the HTML preview wraps.

---

### 2.12 — Page size: A4

Fill Reward Text, select Page Size `A4`, click **Preview**, then **Download PDF**. Verify the
`/generate` POST returns 200. The PDF content-type check is sufficient here — we cannot
verify A4 dimensions without opening the PDF.

---

### 2.13 — Download PDF button on main form (known bug candidate)

On the main form at `/`, without filling any fields, click **Download PDF** (the green button
below Preview). Verify:
- Does it navigate to `/generate`?
- Does it show a raw error page?
- Is there a way for the user to recover (back button aside)?

Record FAIL if it shows a raw error page with no navigation UI.

---

### 2.14 — Back to form: field retention

Fill Child Name `Alex`, Reward Text `Cookie`, Tokens `7`, Theme `Pink`. Click **Preview**.
On the preview page, click **← Back to form**. Verify:
- Do the form fields retain the values entered before clicking Preview?

Record PASS if all values are preserved, NOTE if the form resets to defaults.

---

## Phase 3 — Document Findings

Write all findings to `QA_FINDINGS.md` in the project root:

```markdown
# QA Findings — tokenBoardCreator Web UI

**Run date:** <today>

## Summary

| Test | Result | Notes |
|------|--------|-------|
| 2.1 Form defaults | PASS/FAIL | |
| 2.2 Empty reward validation | PASS/FAIL | |
| 2.3 Token count below min | PASS/FAIL | |
| 2.4 Token count above max | PASS/FAIL | |
| 2.5 Happy path preview | PASS/FAIL | |
| 2.6 Download PDF from preview | PASS/FAIL | |
| 2.7 No name hides name band | PASS/FAIL | |
| 2.8 Custom title | PASS/FAIL | |
| 2.9 All themes | PASS/FAIL/NOTE | |
| 2.10 All token styles | PASS/FAIL | |
| 2.11 Max tokens wrap | PASS/NOTE | |
| 2.12 A4 page size | PASS/FAIL | |
| 2.13 Download PDF from main form | PASS/FAIL | |
| 2.14 Back to form field retention | PASS/NOTE | |

## Detailed Findings

### <Test Name>

**Result:** PASS | FAIL | NOTE

**Observed:** <what actually happened>

**Expected:** <what should have happened>

**Evidence:** <screenshot filename or evaluate_script output>

---

(repeat for each non-PASS result)

## Recommended Changes

For each FAIL or NOTE, one line on what needs to be fixed.
```

---

## Phase 4 — Decision Gate

Present the findings to the user. Use `AskUserQuestion`:

```
QA is complete. Here's a summary:

<paste the Summary table>

Full findings are in QA_FINDINGS.md.

Which issues would you like me to fix? Say "all", "none", name specific tests, or "done" to skip to merge.
```

- If "none" or "done" and no FAILs: skip to Phase 8.
- If "none" or "done" and there ARE FAILs: confirm they want to skip before proceeding.
- Otherwise: proceed to Phase 5 for each approved fix.

---

## Phase 5 — Code Changes via Agents

### 5.1 — Create a feature branch

```bash
cd /home/owner/GolandProjects/tokenBoardCreator && git checkout -b qa/web-fixes
```

### 5.2 — Spawn one agent per approved fix

For each fix, spawn a sub-agent via the Task tool:

```
You are working in /home/owner/GolandProjects/tokenBoardCreator.

**Problem:** <describe the bug — what's broken, where it manifests, what the user experiences>

**Context:** See QA_FINDINGS.md for supporting detail on this finding.

**Desired outcome:** <describe the correct behavior>

Do not prescribe an implementation. Read the relevant source files first.
Commit your changes as the final step.

End your summary with:
## Risks
- Major: <one line, or omit>
- Minor: <one line, or omit>
- Minimal: <one line, or omit>
```

Wait for each agent to complete before running Phase 6.

---

## Phase 6 — Diff Review

After each agent completes, invoke `/diff-view` with `HEAD~<n>..HEAD` (where n = commits the
agent made) and pass the agent's risks.

### Risk categories

- **Major** — can break the app, corrupt PDF output, hang the server, or cause data loss.
  **Auto-fix immediately** by spawning another agent. Do not ask the user.
- **Minor** — degrades a specific scenario. Flag to user before fixing.
- **Minimal** — cosmetic or theoretical. Document only.

Repeat until no Major risks remain. Report after each round:
```
Round <N> complete.
- Major: <count> — <auto-fixed | none>
- Minor: <count> — <list>
- Minimal: <count>
```

---

## Phase 7 — User Approval

When all agents are done and no Major risks remain:

```
AskUserQuestion: "All fixes are committed and no Major risks remain.

<final diff summary>

Minor risks: <list, or 'none'>

Ready to merge to main. Approve? (yes / no / fix minor risks first)"
```

- "fix minor risks first" → spawn agents, repeat Phase 6.
- "no" → ask what to change, loop to Phase 5.
- "yes" → proceed to Phase 8.

---

## Phase 8 — Merge to Main

### 8.1 — Final checks

```bash
cd /home/owner/GolandProjects/tokenBoardCreator && go build . && go test ./... -race && go vet ./...
```

If any check fails, report to the user. Do not merge.

### 8.2 — Merge

```bash
git checkout main && git merge --no-ff qa/web-fixes
```

### 8.3 — Confirm

```bash
git log --oneline -5
```

Report:
```
Merged to main. Final commits:
<git log output>

QA_FINDINGS.md documents all findings from this session.
```

---

## Checkpoints

```
Phase 1: [ ] Built  [ ] Running on :8080  [ ] Browser open
Phase 2: [ ] 2.1 defaults  [ ] 2.2 required  [ ] 2.3 min  [ ] 2.4 max
         [ ] 2.5 happy path  [ ] 2.6 download  [ ] 2.7 no name  [ ] 2.8 title
         [ ] 2.9 themes  [ ] 2.10 styles  [ ] 2.11 wrap  [ ] 2.12 A4
         [ ] 2.13 main form download  [ ] 2.14 back retention
Phase 3: [ ] QA_FINDINGS.md written
Phase 4: [ ] User decisions received
Phase 5: [ ] Agents spawned and complete
Phase 6: [ ] Diff reviewed  [ ] No Major risks
Phase 7: [ ] User approved
Phase 8: [ ] Merged to main
```

---

## Error Handling

- **Build fails**: Report error and stop — do not run.
- **Port 8080 in use**: `lsof -ti:8080 | xargs kill` then retry once.
- **Browser unavailable**: Document what was tested manually, note limitation in findings.
- **Agent fails to commit**: Note in findings; manually commit before diff review.
- **Merge conflict**: Report to user. Do not force-merge.

---

## Notes

- Never make code edits directly — all changes go through spawned agents.
- Kill the server when done: `lsof -ti:8080 | xargs kill`
- `QA_FINDINGS.md` persists after the session — do not delete it.
- The app binary is `./tokenBoardCreator`, not `./teetime`.
