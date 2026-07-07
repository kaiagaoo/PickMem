<div align="center">

# 📌 PickMem

### You pick what your AI remembers.

**A local-first memory layer for LLMs.** Your memory is a folder of Markdown you own — and the default is *nothing*. For each task you choose the slice that reaches the model. No database, no account, no silent injection.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](go.mod)
[![Local-first](https://img.shields.io/badge/local--first-no%20telemetry-2E3440.svg)](#-design-principles)
[![Release](https://img.shields.io/github/v/release/kaiagaoo/PickMem?color=success&label=release)](https://github.com/kaiagaoo/PickMem/releases/latest)
[![MCP](https://img.shields.io/badge/MCP-compatible-5A45FF.svg)](#-delivery-channels)

[**Install**](#-install) · [**Quick start**](#-quick-start) · [**User Guide**](USER_GUIDE.md) · [**How it compares**](#-how-it-compares)

</div>

---

## 👀 What it looks like

Pick from a tree of your *own* memory — group headers are selectable, so toggling one grabs everything under it:

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

## 🤔 Why PickMem?

Every major assistant now has "memory," and it sometimes makes answers *worse* — pulling in details irrelevant to the question, or quietly leaning on private information you never meant to raise. The root problem is **who holds the controls**: the system decides what's relevant and injects it silently.

Nowhere does this matter more than when you ask for advice. With auto-memory on, an assistant can factor in whatever it has stored about you — your budget, your past purchases, your health — and you have no way to see what tilted the answer.

PickMem is the alternative that lets you turn auto-memory off *without going context-blind*. The model weighs only the facts you deliberately chose, so you can trust a response on a basis you can actually see. **You decide what's in scope.**

---

## ✨ Why it's different

Three things set PickMem apart — everything else is in service of these:

- **🎯 You decide what the model sees.** The default is nothing; you pick the slice, per task. No system guessing, no silent injection — the one axis no auto-memory product gives you.
- **📂 Your memory is plain Markdown you own.** A folder of notes on your disk, Obsidian-native. No database, no account, no lock-in — readable and portable ten years from now.
- **🔌 One vault, any model.** The same memory reaches Claude, ChatGPT, Gemini, Cursor, and Cline. Switch assistants without leaving your memory behind — something no built-in memory can do.

<sub>Built on top: **Claude can save memories for you** (it extracts and stages; you approve), **capture** any page selection, **lenses** for recurring picks, **note types** (fact / idea / thought / reference), and a **folder-defined taxonomy** with `_private` folders. All local, all reviewed by you.</sub>

---

## 🧠 How it works

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

## 🚀 Install

**One-liner (macOS / Linux):**

```bash
curl -fsSL https://raw.githubusercontent.com/kaiagaoo/PickMem/main/install.sh | sh
```

Detects your OS/arch, downloads the right binary from the [latest release](https://github.com/kaiagaoo/PickMem/releases/latest), verifies its checksum, and installs to `/usr/local/bin` (or `~/.local/bin` without sudo). Pin a version with `PICKMEM_VERSION=v0.1.1`, change the target with `PICKMEM_INSTALL_DIR=…`.

<sub>Windows: download the `_windows_amd64.zip` asset from the [releases page](https://github.com/kaiagaoo/PickMem/releases/latest) and unzip.</sub>

**From source** (needs **Go 1.26+**):

```bash
git clone https://github.com/kaiagaoo/PickMem
cd PickMem
go install ./cmd/pickmem      # → $(go env GOPATH)/bin; make sure it's on PATH
```

> The browser extension is separate and needs Node 20+ / Chrome — see [Delivery channels](#-delivery-channels).

---

## ⚡ Quick start

```bash
# 1. Create a vault (starter taxonomy + routing rules + one
#    fill-in-the-blank note per group — the vault starts as a form)
pickmem init ~/PickMemVault

# 2. Fill in the blanks (in Obsidian or $EDITOR)… 
pickmem edit <id>       # ids are printed by `pickmem list`

#    …and/or add your own memories
pickmem add --label "salary" --group finance/income --body "monthly base \$8k + quarterly bonus"

# 3. Pick what this session should see
pickmem pick

# 4a. Wire it into Claude Desktop, then restart the app
pickmem install claude-desktop

# 4b. …or use it in the browser
cd extension && npm install && npm run build
#     then load extension/dist/ via chrome://extensions → Load unpacked
```

**Let Claude fill your memory:** once connected, just say *"remember that I'm allergic to penicillin."* Claude extracts the fact, picks a group from your folders, and stages it to your inbox — then run `pickmem review` and press `A` to accept. Nothing goes live until you say so.

Full walkthrough: **[USER_GUIDE.md](USER_GUIDE.md)**.

---

## 🔌 Delivery channels

**MCP (native clients).** `pickmem serve` is a stdio MCP server. It exposes a `pickmem://active` resource and six tools:

| Tool | What it does |
|------|--------------|
| `get_active_memory` | Return the slice you picked for this session |
| `list_lenses` / `use_lens` | List and activate saved selections |
| `list_groups` | Report your folder taxonomy (so the model classifies into *your* categories; `_`-prefixed folders stay private) |
| `stage_memories` | Claude stages facts it extracted — pending, never activated |
| `propose_memories` | Bulk-stage raw text as a fallback |

`pickmem install claude-desktop|cursor` writes the config for you; Cline is a one-line manual setup.

**Chrome extension.** Grant your vault folder once (File System Access API), then: **pick** items in the popup and **Insert** the block into the chat box on supported sites (or **Copy** to paste anywhere); **add** a memory from the popup; or **capture** any page selection into your inbox via the right-click menu / keyboard shortcut. It only ever *creates* notes and writes `active.json` / `lenses.json` — editing existing notes stays in Obsidian or the CLI.

---

## 🔒 Design principles

These are enforced invariants, not aspirations:

- **Local-first, no exceptions.** Your vault stays on disk; PickMem makes no network calls.
- **Create-only.** PickMem creates notes and moves inbox items into folders. It **never rewrites a note you authored** — it verifies on-disk content before touching any file it owns, and refuses if you changed it.
- **The user decides relevance.** No silent auto-injection. Auto-extraction only ever *proposes* into an inbox; nothing goes live without your review.
- **Deterministic lookup, not RAG.** A picked item is fetched by id — an exact read, not a similarity guess.
- **You own the taxonomy.** Your folder tree defines your categories, and you choose which reach the model.

---

## 🆚 How it compares

- **Agent-memory backends** (Mem0, Zep, Letta) and **built-in memory** (ChatGPT/Claude/Gemini) bet on *automatic* retrieval — the system chooses. PickMem hands the choice back to you.
- **Auto-organizing "second brain" tools** build and curate an Obsidian vault for you (retrieval-first, AI writes). PickMem is *curation-first* and complementary: it adds the per-session **pick** that scopes what the model sees this time.

The single differentiating axis: **system-decides-relevance vs. user-decides-relevance.**

---

## 🗂️ Project layout

```
cmd/pickmem/       CLI entry point (cobra)
internal/
  vault/           the store: notes, groups, taxonomy, inbox, lenses, active
  picker/          Bubble Tea TUI (grouped tree picker + inbox review)
  mcp/             MCP server exposing the picked slice + save tools
  ingest/          import parsers + staging pipeline
  routing/         keyword group-routing rules
  install/         client config writers (Claude Desktop, Cursor)
templates/         starter taxonomy (embedded)
extension/         MV3 TypeScript Chrome extension
demo/              VHS tape (pick.tape)
```

## 🛠️ Development

```bash
go test ./...        # Go unit tests
go vet ./...
gofmt -l .           # should be empty

cd extension
npm run typecheck
npm test
npm run build        # → extension/dist/
```

## 📈 Status

The CLI, TUI picker, MCP server (with model-driven `stage_memories`), import/review pipeline, and Chrome extension are all working and tested. This is an actively evolving personal project; the vault format — Markdown-with-frontmatter notes plus a few small JSON files — is the stable contract shared between the Go binary and the extension.

## 📄 License

[MIT](LICENSE) © 2026 Kaia Gao
