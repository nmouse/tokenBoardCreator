---
name: qa
description: >
  Full QA workflow for the teetime app: build and run the app, drive the browser to test every UI control,
  cross-check data against live booking sites, document findings, present them to the user, spawn agents
  for approved changes, review diffs with risk annotations, and merge to main when clean.
  Triggers on "qa", "run qa", "qa workflow", "test the app", "quality check", or /qa.
allowed-tools: Bash, Write, Read, Task, AskUserQuestion, mcp__chrome-devtools__navigate_page, mcp__chrome-devtools__take_screenshot, mcp__chrome-devtools__click, mcp__chrome-devtools__fill, mcp__chrome-devtools__select_page, mcp__chrome-devtools__list_pages, mcp__chrome-devtools__new_page, mcp__chrome-devtools__evaluate_script, mcp__chrome-devtools__wait_for, mcp__chrome-devtools__get_network_request, mcp__chrome-devtools__list_network_requests, mcp__chrome-devtools__fill_form, mcp__chrome-devtools__hover, mcp__chrome-devtools__press_key, mcp__chrome-devtools__close_page
---

# QA Workflow

End-to-end QA for the teetime web UI. Runs the app, drives the browser against every UI control, cross-checks
tee time data against live booking sites, documents findings, iterates changes through agents with diff review,
and merges to main when no Major risks remain.

## Arguments

- **location** (required): location to pass to `--location` (e.g. `"Palo Alto, CA"`)
- **date** (optional): date in `YYYY-MM-DD` format. Defaults to tomorrow.
- **radius** (optional): search radius in miles. Defaults to `25`.
- **players** (optional): number of players. Defaults to `2`.
- **holes** (optional): 9 or 18. Defaults to `18`.
- **from** (optional): earliest time filter `HH:MM`. Defaults to `08:00`.
- **to** (optional): latest time filter `HH:MM`. Defaults to `14:00`.

Example invocation:
```
/qa location="Palo Alto, CA" date=2026-06-10 players=2 from=08:00 to=14:00
```

---

## Phase 1 — Build and Run

### Step 1.1 — Resolve arguments

Parse the invocation for the arguments above. If `location` is missing, ask:
```
AskUserQuestion: "What location should I use for the QA run? (e.g. 'Palo Alto, CA')"
```

Set defaults for any omitted arguments.

### Step 1.2 — Build

```bash
cd /home/owner/GolandProjects/teetime && go build .
```

If the build fails, report the error and stop. Do not proceed to run.

### Step 1.3 — Run the app in web mode

Run the app in the background with `--web` and all QA arguments:

```bash
cd /home/owner/GolandProjects/teetime && ./teetime \
  --location "<location>" \
  --date <date> \
  --radius <radius> \
  --players <players> \
  --holes <holes> \
  --from <from> \
  --to <to> \
  --web &
```

Capture stdout. Watch for the line:
```
Serving results at http://localhost:<PORT>  (Ctrl+C to exit)
```

Extract the URL (e.g. `http://localhost:54321`). If it does not appear within 120 seconds, report a timeout and stop.

### Step 1.4 — Open the browser

Use `mcp__chrome-devtools__navigate_page` to open the extracted URL. If no page exists, use `mcp__chrome-devtools__new_page` first.

Take a screenshot with `mcp__chrome-devtools__take_screenshot` and note the initial state.

---

## Phase 2 — UI Testing

Work through each control listed below. For each test, take a screenshot before and after the interaction and record the result (pass / fail / note).

### 2.1 — Filter bar pre-population

Verify the filter bar is pre-populated from the CLI flags used in Step 1.3:

| Control | Expected value |
|---------|----------------|
| `#f-from` (From time) | `<from>` argument (e.g. `08:00`) |
| `#f-to` (To time) | `<to>` argument (e.g. `14:00`) |
| `#f-spots` (Min spots) | `<players>` argument (e.g. `2`) |
| `#f-date` (Date) | `<date>` argument |

Use `mcp__chrome-devtools__evaluate_script` to read the actual values:
```javascript
({
  from:    document.getElementById('f-from').value,
  to:      document.getElementById('f-to').value,
  spots:   document.getElementById('f-spots').value,
  date:    document.getElementById('f-date').value
})
```

Record PASS if all match, FAIL with specifics if any diverge.

### 2.2 — Expand / collapse courses

For a course row that has tee times (has class `has-times`):
1. Click the row — verify tee time rows appear beneath it (class `teetime-row`).
2. Click again — verify the tee time rows disappear.
3. Expand two courses simultaneously — verify both sets of rows are visible at the same time.

Use `evaluate_script` to count visible `teetime-row` elements before and after each click.

### 2.3 — From / To time filter

1. Clear the From field and set it to `12:00`. Verify tee times before noon disappear.
2. Clear the To field and set it to `10:00`. Verify tee times after 10am disappear.
3. Set From and To both to the same value. Verify only exact-match times remain (or zero if none).
4. Clear both fields. Verify all tee times return.

### 2.4 — Min spots filter

Step through each option in `#f-spots` (1, 2, 3, 4). For each:
- Verify visible tee time rows all have `players >= selected value`.
- Use `evaluate_script` to confirm.

### 2.5 — Sort

Test each option in `#f-sort`:
- **Distance**: course rows should appear in ascending distance order.
- **Earliest time**: courses with the earliest first tee time should appear first.
- **Lowest price**: courses with the lowest minimum price should appear first.

For each sort, use `evaluate_script` to extract the rendered order and verify it is correct.

### 2.6 — Hide unavailable toggle

1. Note the total number of rows (including no-time rows) before toggling.
2. Check `#f-hide`. Verify all `no-time` rows disappear.
3. Uncheck `#f-hide`. Verify they return.

### 2.7 — Date change

1. Change the date picker to tomorrow + 1 (two days from today).
2. Verify a loading spinner/row appears immediately.
3. Wait for new results to load (up to 60 seconds). Verify the header date updates and the table re-renders with new data.
4. Change back to the original date. Verify results reload.

If the fetch errors, record the error message shown in the table.

### 2.8 — Book links

For each expanded course row, verify the "Book →" link:
- Is present for tee times that have a `bookURL`.
- Opens to the correct booking provider (Chronogolf or ForeUP URL pattern).
- Has `target="_blank"`.

Use `evaluate_script` to check `href` values without actually navigating away:
```javascript
Array.from(document.querySelectorAll('a.book')).map(a => ({
  text: a.textContent.trim(),
  href: a.href,
  target: a.target
}))
```

### 2.9 — No-times status messages

Locate courses displayed in the no-time row style. Verify:
- Courses with a provider but no available times show: `"no times available"`
- Courses where no online booking was found show: `"no online booking found"`
- Courses that errored show the actual error message (not a blank cell or `undefined`).

### 2.10 — Cross-check tee time data

For at least 2 courses that have tee times (one Chronogolf, one ForeUP if both present):

1. Note the times, prices, and players shown in the UI.
2. Open the live booking site in a new tab using the Book link URL:
   - Chronogolf: extract the club slug from the URL, open `https://chronogolf.ca/club/<slug>/teetimes`
   - ForeUP: open the book URL directly
3. Compare at least 3 tee times (time, players, price) between the UI and the live site.
4. Record any discrepancies (wrong price, wrong player count, missing times, extra times).

---

## Phase 3 — Document Findings

Write all findings to `QA_FINDINGS.md` in the project root using this template:

```markdown
# QA Findings — <location> — <date>

**Run date:** <today>
**Flags:** --location "<location>" --date <date> --radius <radius> --players <players> --holes <holes> --from <from> --to <to>

## Summary

| Test | Result | Notes |
|------|--------|-------|
| Filter bar pre-population | PASS/FAIL | |
| Expand/collapse | PASS/FAIL | |
| From/To filter | PASS/FAIL | |
| Min spots filter | PASS/FAIL | |
| Sort | PASS/FAIL | |
| Hide unavailable | PASS/FAIL | |
| Date change | PASS/FAIL | |
| Book links | PASS/FAIL | |
| No-times status messages | PASS/FAIL | |
| Data cross-check | PASS/FAIL | |

## Detailed Findings

### <Test Name>

**Result:** PASS | FAIL | NOTE

**Observed:** <what actually happened>

**Expected:** <what should have happened>

**Evidence:** <screenshot filename or evaluate_script output>

---

(repeat for each test)

## Courses Tested

| Course | Provider | Times in UI | Times on Site | Match? |
|--------|----------|-------------|---------------|--------|
| | | | | |

## Recommended Changes

For each FAIL or NOTE, list a brief description of what needs to be fixed.
```

---

## Phase 4 — Decision Gate

Present the findings to the user. If there are no failures or notes, say so and ask whether to proceed with merging.

Use `AskUserQuestion` with this prompt:

```
QA is complete. Here's a summary of findings:

<paste the Summary table from QA_FINDINGS.md>

Full findings are in QA_FINDINGS.md.

Which issues would you like me to fix? List them by name, or say "all", "none", or "just the majors".
You can also say "done" to skip straight to merge.
```

Wait for the user's response before continuing.

If the user says "none" or "done" and there are no FAIL findings: skip to Phase 8 (Merge).
If the user says "none" or "done" and there ARE FAIL findings: confirm they want to skip fixes before proceeding.

---

## Phase 5 — Code Changes via Agents

For each approved change:

### 5.1 — Create a feature branch (if not already on one)

```bash
git checkout -b qa/<location-slug>-<date>
```

Where `<location-slug>` is the location with spaces replaced by hyphens, lowercased.

### 5.2 — Spawn one agent per approved change

For each change, spawn a sub-agent using the Task tool with this prompt structure:

```
You are working in /home/owner/GolandProjects/teetime.

**Problem:** <describe the bug or issue observed during QA — what's broken and where>

**Context:** See QA_FINDINGS.md for full details on this finding.

**Desired outcome:** <describe the correct behavior the user should see>

Do not prescribe an implementation approach — find the best solution yourself.
Read the relevant source files before making changes.
Commit your changes as the final step.

End your summary with a risks section in this exact format:
## Risks
- Major: <one line, or omit if none>
- Minor: <one line, or omit if none>
- Minimal: <one line, or omit if none>
```

Key rules for agent prompts:
- Describe the **problem and desired outcome only** — do not specify how to implement the fix.
- Link to `QA_FINDINGS.md` for supporting context.
- Always include the `## Risks` format instruction.
- Always include "Commit your changes as the final step."

Wait for each agent to complete before running the diff review step.

---

## Phase 6 — Diff Review

After each agent completes, run `/diff-view` with the risks from the agent's summary.

### 6.1 — Extract risks from agent summary

Parse the agent's `## Risks` section. Extract Major, Minor, and Minimal text.

### 6.2 — Run diff-view

Invoke the `diff-view` skill (as `/diff-view`) with the ref range pointing to the agent's new commits and the risks from the agent summary.

Use `HEAD~<n>..HEAD` where `<n>` is the number of commits the agent made, or use the commit hash directly.

Pass the risks as: `risks: Major: <text> | Minor: <text> | Minimal: <text>`

### 6.3 — Categorize risks

After the diff opens in the browser, categorize each risk per the project rules:

- **Major** — can break the app, hang execution, cause data loss, or silently corrupt results for all users. **Auto-fix immediately** by spawning another agent (same rules: describe problem + desired outcome, commit, return risks). Do not ask the user first.
- **Minor** — degrades a specific scenario. Flag clearly in your report. Present to the user before fixing.
- **Minimal** — pre-existing behavior, cosmetic, or theoretical. Document only.

### 6.4 — Repeat until no Major risks remain

If the auto-fix agent introduces a Major risk in its own diff, spawn another fix agent for that risk. Continue until the diff review shows no Major risks.

Report to the user after each round:
```
Round <N> diff review complete.
- Major risks: <count> — <auto-fixed | none>
- Minor risks: <count> — <list>
- Minimal risks: <count>
```

---

## Phase 7 — User Approval of Final Diff

After all agents have completed and no Major risks remain, show the user the final diff summary and ask:

```
AskUserQuestion: "All changes are committed and no Major risks remain. Here's the final diff summary:

<summary from diff-view>

Minor risks identified:
<list any Minor risks>

Ready to merge to main. Approve? (yes / no / fix minor risks first)"
```

If the user says "fix minor risks first": spawn agents for each minor risk (same rules) and repeat Phase 6.

If the user says "no": ask what they'd like to change and loop back to Phase 5.

If the user says "yes": proceed to Phase 8.

---

## Phase 8 — Merge to Main

### 8.1 — Final checks

```bash
cd /home/owner/GolandProjects/teetime && go build . && go test ./... && go vet ./...
```

If any check fails, report to the user. Do not merge.

### 8.2 — Merge

```bash
git checkout main && git merge --no-ff qa/<branch-name>
```

### 8.3 — Confirm

```bash
git log --oneline -5
```

Report the final state to the user:
```
Merged to main. Final commits:
<git log output>

QA_FINDINGS.md documents all findings from this session.
```

---

## Checkpoints and State

Keep a running checklist in memory as you execute. If interrupted mid-phase, restart from the last completed phase.

```
Phase 1: [ ] Built  [ ] Running at <URL>
Phase 2: [ ] Pre-pop  [ ] Expand/collapse  [ ] From/To  [ ] Spots  [ ] Sort  [ ] Hide  [ ] Date  [ ] Book  [ ] Status msgs  [ ] Cross-check
Phase 3: [ ] QA_FINDINGS.md written
Phase 4: [ ] User decisions received
Phase 5: [ ] Agents spawned and complete
Phase 6: [ ] Diff reviewed  [ ] No Major risks
Phase 7: [ ] User approved merge
Phase 8: [ ] Merged to main
```

---

## Error Handling

- **App won't start**: Check for port conflicts, build errors, or missing flags. Report and stop.
- **Browser won't open**: Try navigating manually to the URL via `navigate_page`. If DevTools MCP is unavailable, document what was tested manually and note the limitation in findings.
- **Live booking site is unreachable**: Mark the cross-check as SKIPPED with reason; do not fail the entire QA run.
- **Agent fails to commit**: Note in findings; manually commit at the end of Phase 5 before proceeding to diff review.
- **Merge conflict**: Report to the user. Do not force-merge.

---

## Notes

- Never make code edits directly. All code changes go through spawned agents (per project rules).
- Only show new diffs in `/diff-view` — do not re-show diffs already reviewed.
- The `QA_FINDINGS.md` file persists after the session; do not delete it.
- Always kill the background server process when the QA session ends: `pkill -f 'teetime.*--web'`
