<div align="center">

# 📌 PickMem

### You pick what your AI remembers.

**A local-first memory layer for LLMs.** Your memory is a folder of Markdown you own — and the default is *nothing*. For each task you open a small local web app, pick the slice that reaches the model, and hand it over. No database, no account, no silent injection.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](go.mod)
[![Local-first](https://img.shields.io/badge/local--first-no%20telemetry-2E3440.svg)](#-design-principles)
[![Release](https://img.shields.io/github/v/release/kaiagaoo/PickMem?color=success&label=release)](https://github.com/kaiagaoo/PickMem/releases/latest)
[![MCP](https://img.shields.io/badge/MCP-compatible-5A45FF.svg)](#-how-it-reaches-a-model)

[**Install**](#-install) · [**Quick start**](#-quick-start) · [**User Guide**](USER_GUIDE.md) · [**How it compares**](#-how-it-compares)

</div>

---

## 👀 What it looks like

`pickmem web` opens a local app in your browser — a calm three-zone workspace over your own vault:

```
┌───────────────┬─────────────────────────────────────┬───────────────────────┐
│ 📌 My memory ▾│  All memories / finance              │  ACTIVE MEMORY        │
│               │  ─────────────────────────────────   │  Lens: Advice ▾       │
│  ▸ about      │  ▸ income   ▸ bills   ▸ goals        │  ● Salary             │
│  ▾ finance    │                                      │  ● Anemia             │
│    · income   │  ┌───────────────────────────────┐   │                       │
│    · bills    │  │ ● Salary                fact  │   │  2 items · ~28 tokens │
│  ▸ work       │  │   Monthly base $8k + bonus    │   │                       │
│  ─────────    │  └───────────────────────────────┘   │  text · md · json     │
│  ✦ Lenses     │                                      │  [ Copy context ]     │
│  Inbox (0)    │                        [+ Memory]    │  [ Save as lens ]     │
└───────────────┴─────────────────────────────────────┴───────────────────────┘
   navigate            browse & pick               the slice the model gets
```

Toggle items on the left or in the center; the **Active Memory** tray on the right is *exactly* what the model receives — copy it out, or let a connected assistant read it. Nothing else from your vault goes along:

```
--- pickmem: selected memory (lens: Advice) ---
Salary (finance/income): monthly base $8k + quarterly bonus

Anemia (about/health): chronic anemia, on iron supplements
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

<sub>Built on top: a **local web app** to build and pick memory (folder tree, drill-down browsing, groups you rename/reorganize, multiple vaults you switch between), a **Chrome extension** to insert your pick into any chat box, **lenses** for recurring picks, **note types** (fact / idea / thought / reference), and **Claude can stage memories for you** (it extracts; you approve). All local, all reviewed by you.</sub>

---

## 🧠 How it works

```
              your memory  (a folder of Markdown notes you own)
                                   │
        pickmem web  ──▶  you pick a slice  ──▶  pickmem/active.json
       (local app in                                     │
        your browser)              ┌─────────────────────┴─────────────────────┐
                                   ▼                                           ▼
                          Chrome extension                            MCP server (stdio)
                     ChatGPT · Claude.ai · Gemini              Claude Desktop · Cursor · Cline
                        (+ copy anywhere)                       (for assistants / AI agents)
```

You curate and pick in the **web app**. That writes one small file (`active.json`). Both delivery channels read it and produce the *same* context block, so what the model sees never depends on which channel you used.

---

## 🚀 Install

**One-liner (macOS / Linux):**

```bash
curl -fsSL https://raw.githubusercontent.com/kaiagaoo/PickMem/main/install.sh | sh
```

Detects your OS/arch, downloads the right binary from the [latest release](https://github.com/kaiagaoo/PickMem/releases/latest), verifies its checksum, and installs to `/usr/local/bin` (or `~/.local/bin` without sudo). The web UI is embedded in the binary — no separate frontend to install.

<sub>Windows: download the `_windows_amd64.zip` asset from the [releases page](https://github.com/kaiagaoo/PickMem/releases/latest) and unzip.</sub>

**From source** (needs **Go 1.26+**):

```bash
git clone https://github.com/kaiagaoo/PickMem
cd PickMem
go install ./cmd/pickmem      # → $(go env GOPATH)/bin; make sure it's on PATH
```

> The Chrome extension is separate and needs Node 20+ / Chrome — see [How it reaches a model](#-how-it-reaches-a-model).

---

## ⚡ Quick start

```bash
# 1. Create a vault (a plain folder — this is your only store)
pickmem init ~/PickMemVault

# 2. Open the web app and build your memory there
pickmem web
```

That's it. The app opens at `http://127.0.0.1:4577`. A brand-new empty vault greets you with a short **setup guide** (pick a few areas, fill in a starter memory or two); a vault with notes drops you straight into the workspace. From there you **add, organize, and pick** — all in the browser. When you want an assistant to use your pick, wire up a [delivery channel](#-how-it-reaches-a-model).

**Let Claude fill your memory:** once the MCP server is connected, just say *"remember that I'm allergic to penicillin."* Claude extracts the fact, picks a group from your folders, and stages it to your **Inbox** — review and accept it in the web app (or `pickmem review`). Nothing goes live until you say so.

Full walkthrough: **[USER_GUIDE.md](USER_GUIDE.md)**.

---

## 🔌 How it reaches a model

You pick in the web app; these channels deliver that pick:

**Chrome extension.** For ChatGPT, Claude.ai, and Gemini, with a clipboard fallback everywhere else. Grant your vault folder once (File System Access API), then **Insert** the assembled block into the chat box on a supported site, or **Copy** to paste anywhere.

```bash
cd extension && npm install && npm run build
#   then load extension/dist/ via chrome://extensions → Load unpacked
```

**MCP server (for assistants / AI agents).** `pickmem serve` is a stdio MCP server that native clients launch. It exposes a `pickmem://active` resource and tools — `get_active_memory`, `list_lenses` / `use_lens`, `list_groups`, and `stage_memories` (Claude extracts facts and stages them to your inbox; nothing activates without your review).

```bash
pickmem install claude-desktop      # or: cursor   (writes the client config for you)
```

<sub>The CLI (`pickmem add`, `pick`, `import`, `review`, `context --copy`, …) still does everything headless — handy for scripting and for AI agents driving PickMem programmatically. See the [User Guide](USER_GUIDE.md#13-command-reference).</sub>

---

## 🔒 Design principles

These are enforced invariants, not aspirations:

- **Local-first, no exceptions.** Your vaults stay on disk; PickMem makes no network calls. The web app is a local server bound to `127.0.0.1`.
- **Create-only.** PickMem creates notes and moves inbox items into folders. It **never rewrites a note you authored** without a guard — it verifies on-disk content before touching any file it owns, and refuses if you changed it out from under it.
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
  web/             the web app: HTTP+JSON API over the store + embedded SPA
  userconf/        machine-level config: current vault + recent-vaults list
  mcp/             MCP server exposing the picked slice + save tools
  picker/          Bubble Tea TUI (headless picker + inbox review)
  ingest/          import parsers + staging pipeline
  routing/         keyword group-routing rules
  install/         client config writers (Claude Desktop, Cursor)
webapp/            React + Vite + TypeScript source for the web app
templates/         starter taxonomy (embedded)
extension/         MV3 TypeScript Chrome extension
```

## 🛠️ Development

```bash
go test ./...        # Go unit tests
go vet ./...
gofmt -l .           # should be empty

cd webapp            # the web app
npm install
npm run typecheck
npm run build        # → internal/web/static/ (embedded into the binary)

cd ../extension      # the browser extension
npm run typecheck && npm test && npm run build   # → extension/dist/
```

> After changing `webapp/`, run `npm run build` there **before** `go build` — the SPA is embedded into the binary from `internal/web/static/`.

## 📈 Status

The web app (three-zone workspace, drill-down browsing, group management, multi-vault switching, onboarding), the Chrome extension, the MCP server (with model-driven `stage_memories`), and the CLI + import/review pipeline are all working and tested. This is an actively evolving personal project; the vault format — Markdown-with-frontmatter notes plus a few small JSON files — is the stable contract shared across every surface.

## 📄 License

[MIT](LICENSE) © 2026 Kaia Gao
