# PickMem

**A local-first memory-curation layer for LLMs — you pick what the model remembers, per session.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)
![Local-first](https://img.shields.io/badge/local--first-no%20telemetry-2E3440.svg)

Every major assistant now has "memory," and it sometimes makes answers *worse* — importing details irrelevant to the question and nudging the model toward agreeing with you or taking advantage of those private personal information without your notice. The root problem is *who holds the controls*: the system decides what's relevant and injects it silently.

PickMem inverts that. Your memory lives in a plain folder of Markdown notes on your own disk. **The default is nothing** — for each task you open a picker, select the items that matter, and only that slice reaches the model.

Nowhere does this matter more than when you ask for advice. With auto-memory on, an assistant can quietly factor in whatever it has stored about you — your budget, your past purchases, details you never meant to raise — and you have no way to see what tilted the answer. PickMem is the alternative that lets you turn auto-memory off without going context-blind: the model weighs only the facts you deliberately chose. You decide what's in scope, so you can trust a reponse on a basis you can actually see.

---

## What it looks like

Pick from a tree of your own memory (group headers are selectable — toggling one selects everything under it):

```
[ ] about
    [ ] health
        [x] chronic anemia, on iron supplements
    [ ] preferences
        [x] uses vim; don't suggest vscode
[ ] finance
    [ ] income
        [x] monthly base $8k + quarterly bonus

Active: custom · 3 selected · ~28 tokens
space toggle · / filter · l lens · s save-lens · enter confirm · q cancel
```

Confirm, and *exactly that* is what the model gets — nothing else from your vault:

```
--- pickmem: selected memory ---
chronic anemia (about/health): chronic anemia, on iron supplements

editor preference (about/preferences): uses vim; don't suggest vscode
--- end pickmem memory ---
```

---

## Features

- **You curate; the system doesn't guess.** Selection is deliberate and per-session. Nothing is auto-injected.
- **Your vault is the store.** Notes are Markdown with YAML frontmatter — readable, editable, and Obsidian-compatible. No database, no lock-in.
- **Two delivery channels, one vault:**
  - **MCP server** for Claude Desktop, Cursor, and Cline.
  - **Chrome extension** that injects into ChatGPT, Claude.ai, and Gemini — plus a clipboard fallback that works anywhere.
- **Grouped, nested picker** (TUI and browser) with fuzzy filter, tri-state group checkboxes, and saved **lenses** for recurring tasks.
- **Bulk ingest.** Import a memory export or a plain list; items stage to an inbox for review before anything goes live.
- **Local-first and private.** No account, no telemetry, no network calls — everything runs on your machine.

---

## How it works

```
          your memory (Obsidian-compatible Markdown vault)
                              │
                    pickmem pick  →  active.json   (the slice you chose)
                              │
             ┌────────────────┴─────────────────┐
             ▼                                   ▼
     MCP server (stdio)                   Chrome extension
   Claude Desktop · Cursor · Cline    ChatGPT · Claude.ai · Gemini
                                        (+ clipboard anywhere)
```

Both channels read the same `active.json` and produce the same context block, so switching between them never changes what the model sees.

---

## Install

**Prebuilt binary (recommended — no Go needed).** Grab the archive for your OS/arch
from the [latest release](https://github.com/qwgao/pickmem/releases/latest), unpack
it, and put `pickmem` on your `PATH`. On macOS/Linux:

```bash
# pick the asset matching your platform: darwin/linux, arm64/amd64
curl -sSL https://github.com/qwgao/pickmem/releases/latest/download/pickmem_darwin_arm64.tar.gz | tar xz
sudo mv pickmem /usr/local/bin/
pickmem --version
```

(Windows: download the `_windows_amd64.zip` asset and unzip.)

**From source** (needs **Go 1.26+**):

```bash
git clone https://github.com/qwgao/pickmem
cd pickmem
go build -o pickmem ./cmd/pickmem
go install ./cmd/pickmem      # or: cp pickmem /usr/local/bin/
```

> The browser extension is separate and needs Node 20+ / Chrome — see below.

---

## Quick start

```bash
# 1. Create a vault (lays down a starter taxonomy + routing rules)
pickmem init ~/PickMemVault

# 2. Add a couple of memories
pickmem add --label "salary" --group finance/income --body "monthly base \$8k + quarterly bonus"
pickmem add --label "editor" --group about/preferences --body "uses vim; don't suggest vscode"

# 3. Pick what this session should see
pickmem pick

# 4a. Wire it into Claude Desktop, then restart the app
pickmem install claude-desktop

# 4b. …or use it in the browser
cd extension && npm install && npm run build
#     then load extension/dist/ via chrome://extensions → Load unpacked
```

Full walkthrough: **[USER_GUIDE.md](USER_GUIDE.md)**.

---

## Delivery channels

**MCP (native clients).** `pickmem serve` is a stdio MCP server exposing a `pickmem://active` resource plus tools `get_active_memory`, `list_lenses`, `use_lens`, and `propose_memories` (which stages candidate memories to your inbox — it never activates anything). `pickmem install claude-desktop|cursor` writes the config entry for you; Cline is a one-line manual setup.

**Chrome extension.** Grant it your vault folder once (File System Access API), pick items in the popup, and **Insert** prepends the assembled block into the chat box on supported sites — or **Copy** to paste anywhere. You can also **add** a new memory from the popup. It only ever creates notes (and writes `active.json` / `lenses.json`) — editing existing notes stays in Obsidian or the CLI.

---

## Design principles

These are enforced invariants, not aspirations:

- **Local-first, no exceptions.** Your vault stays on disk and PickMem makes no network calls.
- **Create-only.** PickMem creates notes and moves inbox items into folders. It **never rewrites a note you authored** — it verifies on-disk content before touching any file it owns, and refuses if you changed it.
- **The user decides relevance.** No silent auto-injection. Auto-extraction only ever *proposes* into an inbox; nothing goes live without your review.
- **Deterministic lookup, not RAG.** A picked item is fetched by id — an exact read, not a similarity guess.
- **Frontmatter is grouping truth.** A note's `group:` field, not its folder, is authoritative — reorganize freely.

---

## How it compares

- **Agent-memory backends** (Mem0, Zep, Letta) and **consumer memory** (ChatGPT/Claude/Gemini) bet on *automatic* retrieval — the system chooses. PickMem hands the choice back to you.
- **claude-obsidian**-style tools *build and auto-organize* an Obsidian brain (retrieval-first, AI writes). PickMem is *curation-first* and complementary: it adds the per-session **pick** that scopes what the model sees this time.

The single differentiating axis: **system-decides-relevance vs. user-decides-relevance.**

---

## Project layout

```
cmd/pickmem/       CLI entry point (cobra)
internal/
  vault/           the store: notes, groups, inbox, lenses, active selection
  picker/          Bubble Tea TUI (grouped tree picker + inbox review)
  mcp/             MCP server exposing the picked slice
  ingest/          import parsers + staging pipeline
  routing/         keyword group-routing rules
  install/         client config writers (Claude Desktop, Cursor)
templates/         starter taxonomy (embedded)
extension/         MV3 TypeScript Chrome extension
demo/              VHS tape (pick.tape)
```

## Development

```bash
go test ./...        # Go unit tests
go vet ./...
gofmt -l .           # should be empty

cd extension
npm run typecheck
npm test
npm run build        # → extension/dist/
```

## Status

The CLI, TUI picker, MCP server, import/review pipeline, and Chrome extension are all working and tested. This basic release routes imported and proposed memories with keyword rules; AI-assisted extraction (splitting messy text into clean, atomic facts) is planned for a future update. This is an actively evolving personal project; the vault format — Markdown-with-frontmatter notes plus a few small JSON files — is the stable contract shared between the Go binary and the extension.

## Documentation

Full walkthrough — install to daily use, every command and flag — in **[USER_GUIDE.md](USER_GUIDE.md)**.

## License

[MIT](LICENSE) © 2026 Kaia Gao
