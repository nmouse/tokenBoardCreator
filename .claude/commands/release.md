---
name: release
description: >
  Cut a tagged release for tokenBoardCreator. Runs pre-flight checks (clean tree, tests pass),
  confirms the version number, creates and pushes the git tag, and reports the GitHub Actions
  URL to watch the build. Triggers on "release", "cut a release", "tag a release", or /release.
allowed-tools: Bash, AskUserQuestion
---

# Release Workflow — tokenBoardCreator

Cuts a versioned release by pushing a `v*` tag, which triggers GitHub Actions to build
binaries for Windows, macOS (amd64 + arm64), and Linux, then publish a GitHub Release.

## Arguments

Optionally accepts a version as the first argument (e.g. `/release v1.3.0`).
If omitted, the workflow will determine it interactively.

---

## Phase 1 — Pre-flight checks

### 1.1 — Confirm clean working tree

```bash
cd /home/owner/GolandProjects/tokenBoardCreator && git status --short
```

If there are any uncommitted changes or untracked files (excluding `board.pdf`), stop and report:
> "Working tree is not clean — commit or stash changes before releasing."

### 1.2 — Confirm on main and up to date

```bash
git branch --show-current && git fetch origin && git rev-list --count HEAD..origin/main
```

If not on `main`, stop and report the current branch.
If behind `origin/main` (count > 0), stop and report how many commits behind.

### 1.3 — Build and test

```bash
go build . && go test ./... -race && go vet ./...
```

If any step fails, stop and report the error. Do not tag a broken build.

---

## Phase 2 — Determine version

### 2.1 — Show recent tags

```bash
git tag --sort=-version:refname | head -5
```

### 2.2 — Resolve the version

If a version argument was passed (e.g. `v1.3.0`), use it directly — skip to Phase 3.

Otherwise, use `AskUserQuestion`:

> "Pre-flight checks passed. The most recent tags are:
> <tag list>
>
> What version should this release be? (e.g. v1.3.0)"

Validate the answer matches the pattern `v<major>.<minor>.<patch>` (e.g. `v1.3.0`).
If it doesn't match, ask again with:
> "Version must follow semver format: v<major>.<minor>.<patch>"

Also confirm the tag doesn't already exist:
```bash
git tag --list "<version>"
```

If it already exists, stop and report: "Tag `<version>` already exists."

---

## Phase 3 — Tag and push

### 3.1 — Create the tag

```bash
git tag <version>
```

### 3.2 — Push the tag

```bash
git push origin <version>
```

If the push fails, delete the local tag and report the error:
```bash
git tag -d <version>
```

---

## Phase 4 — Report

Report success:

```
Release <version> tagged and pushed.

GitHub Actions is now building binaries for Windows, macOS (amd64 + arm64), and Linux.
Watch the build at: https://github.com/nmouse/tokenBoardCreator/actions

Once the workflow completes, the release will appear at:
https://github.com/nmouse/tokenBoardCreator/releases/tag/<version>
```

---

## Error handling

- **Dirty tree**: Stop immediately, list the dirty files, suggest committing or stashing.
- **Not on main**: Stop, tell the user which branch they're on.
- **Behind remote**: Stop, tell the user to `git pull` first.
- **Build/test failure**: Stop, show the error output in full.
- **Tag already exists**: Stop, suggest bumping the version.
- **Push fails**: Delete the local tag, show the git error.
