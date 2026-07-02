# PickMem — User Guide

*A local-first memory-curation layer for LLMs. You pick what the model remembers.*

This guide walks you from "I found this repo on GitHub" to "I'm actively using PickMem with ChatGPT, Claude Desktop, and my Obsidian vault." Follow it top to bottom the first time — later sections assume you've done the earlier ones.

---

## Table of contents

- [PickMem — User Guide](#pickmem--user-guide)
  - [Table of contents](#table-of-contents)
  - [1. What is PickMem?](#1-what-is-pickmem)
  - [2. Prerequisites](#2-prerequisites)
  - [3. Clone and build](#3-clone-and-build)
  - [4. Create your first vault](#4-create-your-first-vault)
    - [Option A — Use PickMem alongside your existing Obsidian vault](#option-a--use-pickmem-alongside-your-existing-obsidian-vault)
    - [Option B — Dedicate a whole vault to PickMem](#option-b--dedicate-a-whole-vault-to-pickmem)
    - [Option C — Just use your whole existing vault](#option-c--just-use-your-whole-existing-vault)
    - [About templates](#about-templates)
  - [5. Add your first memory](#5-add-your-first-memory)
    - [See what you've got](#see-what-youve-got)
    - [Edit or delete a memory](#edit-or-delete-a-memory)
  - [6. Pick which memories the model sees](#6-pick-which-memories-the-model-sees)
  - [7. Use PickMem with Claude Desktop (MCP)](#7-use-pickmem-with-claude-desktop-mcp)
    - [Recommended: two settings that make this actually reliable](#recommended-two-settings-that-make-this-actually-reliable)
  - [8. Use PickMem in your browser (extension)](#8-use-pickmem-in-your-browser-extension)
    - [Build and load](#build-and-load)
    - [Grant your vault](#grant-your-vault)
    - [Insert into a chat](#insert-into-a-chat)
    - [Copy anywhere](#copy-anywhere)
    - [Byte parity with MCP](#byte-parity-with-mcp)
  - [9. Import from ChatGPT / Claude memory exports (still under development)](#9-import-from-chatgpt--claude-memory-exports-still-under-development)
    - [AI-assisted import — split, classify, and merge](#ai-assisted-import--split-classify-and-merge)
  - [10. Review the inbox](#10-review-the-inbox)
  - [11. Lenses — save a selection for a recurring task](#11-lenses--save-a-selection-for-a-recurring-task)
    - [Save from the CLI TUI](#save-from-the-cli-tui)
    - [Save from the extension](#save-from-the-extension)
    - [Activate a lens](#activate-a-lens)
    - [Suggested lenses](#suggested-lenses)
  - [12. Edit and organize in Obsidian](#12-edit-and-organize-in-obsidian)
  - [13. Command reference](#13-command-reference)
    - [Vault + memory](#vault--memory)
    - [Picking + delivery](#picking--delivery)
    - [Ingestion](#ingestion)
    - [Global flags](#global-flags)
  - [14. Troubleshooting](#14-troubleshooting)
  - [Where to go next](#where-to-go-next)

---

## 1. What is PickMem?

Every major AI assistant now has "memory." In practice, that memory often makes answers *worse* — it imports irrelevant details and bends the model toward agreeing with you.

PickMem inverts that: **the default is nothing, and you add context on purpose.** Your memories live in an Obsidian vault on your disk. You select which slice reaches the model, per session.

Two ways to deliver that slice:

- **MCP** — a local server that Claude Desktop, Cursor, Cline, and Claude Code connect to. The model sees only what you picked.
- **Chrome extension** — reads your vault, injects your selection into the chat box on ChatGPT / Claude.ai / Gemini. A **Copy** button works on every other site as a fallback.

Both channels read the same vault. Your brain lives in one place; you choose what leaves.

---

## 2. Prerequisites

You should have:

- **macOS, Linux, or Windows** (macOS instructions are shown; Windows/Linux equivalents work the same way).
- **A terminal you're comfortable with** — Terminal.app, iTerm, Windows Terminal, whatever.
- **Obsidian** installed (you said you already do — good).
- **Go 1.22 or newer** — check with `go version`. If missing: `brew install go` on macOS, or [go.dev/dl](https://go.dev/dl).
- **Node.js 20+ and npm** — only if you want the Chrome extension. Check with `node -v`. If missing: `brew install node` or [nodejs.org](https://nodejs.org).
- **Chrome or a Chromium-based browser** — only for the extension.

You do **not** need: Docker, an API key, a cloud account, or anything else. PickMem is local-first.

---

## 3. Clone and build

Open a terminal and clone the repo somewhere sensible (`~/code/` or `~/projects/` — wherever you keep code):

```bash
cd ~/code
git clone https://github.com/qwgao/pickmem   # substitute the actual URL
cd pickmem
```

Build the CLI binary:

```bash
go build -o pickmem ./cmd/pickmem
```

That produces a `pickmem` binary in the current directory. Put it on your PATH so you can run it from anywhere:

```bash
# Option A — install into $GOPATH/bin (usually already on your PATH)
go install ./cmd/pickmem

# Option B — copy manually
sudo cp pickmem /usr/local/bin/
```

Verify it works:

```bash
pickmem --help
```

You should see the list of subcommands: `init`, `add`, `list`, `show`, `pick`, `serve`, `install`, `import`, `review`, and a few more.

---

## 4. Create your first vault

Your **vault** is a folder on your disk. PickMem uses it as its only source of truth — no database. You have two choices:

### Option A — Use PickMem alongside your existing Obsidian vault

Recommended if you already have an Obsidian vault you love and want to keep it separate.

```bash
pickmem init ~/ObsidianVault/pickmem-memory --template personal
```

This creates a `pickmem-memory/` subfolder inside your existing vault. Everything else in your vault stays untouched.

### Option B — Dedicate a whole vault to PickMem

Cleaner if you're starting fresh.

```bash
pickmem init ~/PickMemVault --template personal
```

Then in Obsidian: **File → Open folder as vault** → select `~/PickMemVault`.

### Option C — Just use your whole existing vault

If you're not worried about organization mixing:

```bash
pickmem init ~/ObsidianVault --template personal
```

PickMem drops a `pickmem/` metadata folder at the root and adds group folders (`financial/`, `home/`, etc.) alongside your existing notes. **PickMem never edits notes it didn't create**, so your existing notes are safe.

### About templates

The `--template` flag seeds a starter taxonomy — folder structure + a `pickmem/config.json` with routing rules. Three ship in the box:

- **`personal`** — `personal/`, `health/`, `home/`, `relationships/`. Rules for `doctor`, `mortgage`, `rent`, `gift`, `birthday`.
- **`developer`** — `projects/`, `stack/`, `learning/`, `tools/`. Rules for `python`, `docker`, `kubernetes`, `postgres`, etc.
- **`researcher`** — `courses/`, `papers/`, `advisors/`, `deadlines/`. Rules for `paper`, `arxiv`, `advisor`.

Pick the one that best matches how you think about your memory. You can rename or reorganize folders later — **frontmatter's `group:` field is what PickMem actually reads**, not the folder name.

You can skip the template entirely if you want empty group folders: `pickmem init ~/PickMemVault`.

After `init`, PickMem remembers the vault path in a config file (`~/.config/pickmem/config.json`), so most subsequent commands don't need `--vault`.

---

## 5. Add your first memory

The simplest way:

```bash
pickmem add --label "salary" --group financial --body "monthly base \$8k plus quarterly bonus"
```

Or with tags:

```bash
pickmem add --label "prefers vim over vscode" --group tools --tags editor,productivity --body "hard preference — don't suggest vscode extensions for solving vim problems"
```

Or pipe a longer body from a file or stdin:

```bash
echo "Loves plants, especially trailing pothos. Also enamel pins with animals." \
  | pickmem add --label "sister gift ideas" --group relationships
```

Or open your `$EDITOR` for a longer note (leave `--body` off, don't pipe stdin):

```bash
EDITOR=vim pickmem add --label "kickoff notes for Client Acme" --group "work/Client-Acme"
```

That last example shows two things:
- Groups can nest with `/`. `work/Client-Acme` means "work" is a parent group, "Client-Acme" is nested inside.
- `$EDITOR` opens a scratch file; save + quit to add the memory.

### See what you've got

```bash
pickmem list                          # everything, grouped
pickmem list --group financial        # just one group
pickmem show <id-or-suffix>           # e.g. show 9TW0KK (last 3+ chars of the ULID)
pickmem show <id> --raw               # print the raw file (frontmatter + body)
```

If you look inside your vault now, you'll see:

```
financial/salary.md
tools/prefers-vim-over-vscode.md
relationships/sister-gift-ideas.md
work/Client-Acme/kickoff-notes-for-client-acme.md
pickmem/
  active.json    (currently empty — nothing picked yet)
  lenses.json    (currently empty)
  config.json
```

Each `.md` is a normal Obsidian note. Open one in Obsidian — you'll see the frontmatter in the Properties pane and the body renders normally.

### Edit or delete a memory

```bash
pickmem edit <id>            # opens $EDITOR on the file — you edit, PickMem doesn't touch it
pickmem rm <id> --yes        # deletes the note (--yes required)
```

`edit` launches your editor because PickMem never rewrites bytes itself. If you edit in Obsidian instead, that also works — PickMem re-reads on every operation.

---

## 6. Pick which memories the model sees

This is the heart of PickMem. Run:

```bash
pickmem pick
```

A full-screen TUI opens showing your memories grouped by folder. Keys:

| Key | Action |
|-----|--------|
| `↑` / `↓` or `k` / `j` | Move (skips group headers) |
| `space` | Toggle the item at cursor |
| `/` | Filter — types match label + body + tags (fuzzy) |
| `l` | Lens overlay (activate a saved lens) |
| `s` | Save current selection as a new lens |
| `enter` | Confirm — writes `pickmem/active.json` and exits |
| `q` or `esc` | Cancel — active.json unchanged |

Toggle a couple of items, then press `enter`. You'll see:

```
Active: custom · 2 items
```

Look at `pickmem/active.json` — the ids you picked are in there. That's the file every downstream channel (MCP server, extension) reads.

**Confirming with nothing selected clears active.json.** That's the "default is nothing" thesis in action — a deliberate way to reset.

---

## 7. Use PickMem with Claude Desktop (MCP)

Assuming you have [Claude Desktop](https://claude.ai/download) installed:

```bash
pickmem install claude-desktop
```

This writes a `pickmem` entry into `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS), merging with any other MCP servers you already had — nothing else is touched. Preview first with `--dry-run`.

Restart Claude Desktop. When it comes back up, PickMem is connected.

**Test it:**

1. In the terminal, `pickmem pick` and select some memories. Confirm with `enter`.
2. Open Claude Desktop, start a new conversation.
3. Ask something that would benefit from your context. Example: if you picked your "salary" memory, ask *"Given what you know about my finances, what should I set as my monthly savings target?"*
4. Claude will consult the `pickmem://active` resource. It sees only the items you picked — nothing else from your vault.

**Tools Claude can call:**

- `get_active_memory` — same block as the resource, via a tool call.
- `list_lenses` — see your saved lenses.
- `use_lens("Job-Hunt")` — switch to a lens mid-conversation. Writes back to `active.json`.
- `propose_memories(chat_text)` — extract candidate memories from a chunk of chat and stage them in your inbox as *pending*. **Nothing goes active without your review.**

Cursor works the same way: `pickmem install cursor`.

For Cline (VS Code), add the MCP server manually from Cline's Settings → MCP Servers. Command: your `pickmem` binary; args: `serve`.

### Recommended: two settings that make this actually reliable

Out of the box, Claude *can* call PickMem's tools, but two default behaviors get in the way of it doing so smoothly. Neither of these adds anything to what the model sees beyond what you've already picked — they just remove friction around using it. Worth setting up once.

**1. Set tool permissions to "Always allow."**

By default, Claude Desktop asks for per-call confirmation the first time it wants to use a new tool. For a memory layer you're using in most conversations, that gets old fast.

Go to **Settings → Connectors → pickmem → Tool access**, and for all four tools (`Get active memory`, `List lenses`, `Propose memories`, `Use lens`), set the permission to **Always allow**.

This doesn't change what's exposed — it's still only the slice you picked with `pickmem pick`. It just means Claude doesn't have to stop and ask before reading it.

**2. Add a Custom Instruction telling Claude to check memory proactively.**

Without this, Claude often won't call `get_active_memory` unless you explicitly say "check my memory first" — the tool's presence alone isn't enough of a signal. Go to **Settings → Profile → "What personal preferences should Claude take into account?"** and add:

> Before answering anything that might depend on my personal context, preferences, or facts about my life, check my PickMem active memory first — call `get_active_memory` or read the `pickmem://active` resource. If it comes back empty or unrelated to the question, say so rather than guessing.

This is a Claude Desktop application setting, not something PickMem's `install` command writes for you — and deliberately so. PickMem writing to your global instructions on your behalf would be exactly the kind of silent auto-injection this project exists to avoid (see [PROPOSAL.md §1](PROPOSAL.md)). You turning this on yourself, once, is you extending your own "the user decides relevance" choice to *how* the model uses what you already picked — not the system deciding for you.

With both set, a fresh "what should I budget for rent?" question (no explicit "check my memory" prompt) should pull from your active selection unprompted.

---

## 8. Use PickMem in your browser (extension)

For ChatGPT, Claude.ai, Gemini (and clipboard-fallback everywhere else).

### Build and load

```bash
cd extension/
npm install
npm run build       # writes extension/dist/
```

In Chrome:

1. Open `chrome://extensions`.
2. Enable **Developer mode** (top-right toggle).
3. Click **Load unpacked**. Select `extension/dist/`.

The PickMem icon appears in your toolbar. Pin it if it hides in the puzzle-piece menu.

### Grant your vault

Click the PickMem icon. First time, you'll see **Choose vault folder…**. Click it and pick the same folder you `pickmem init`-ed. Chrome persists the grant across sessions.

The popup now shows your groups, lenses, and items. Toggle to select. Save-as-lens with the field at the bottom.

### Insert into a chat

1. Go to `chatgpt.com` (or `claude.ai`, or `gemini.google.com`).
2. Click the PickMem icon. Header should say `ChatGPT · ready` (or similar).
3. Type your question in the chat box first if you want.
4. Click **Insert**. The assembled block prepends into the composer above your question.

If a site's DOM has changed and PickMem can't find the input, you'll get a specific message. Click **Copy** instead and paste manually — that always works.

### Copy anywhere

On any site (Perplexity, poe.com, `example.com`, whatever), the **Copy** button assembles the block and puts it on your clipboard. Paste anywhere.

### Byte parity with MCP

The extension and the MCP server produce the same context block for the same selection. This is by design — switching between Claude Desktop and ChatGPT doesn't change what the model sees.

---

## 9. Import from ChatGPT / Claude memory exports (still under development)

If you've been using another assistant's memory feature and have an export, PickMem can ingest it.

**ChatGPT** → Settings → Personalization → Manage memories → export (or copy-paste the list). Save it as `chatgpt-export.txt` or `.json`.

**Claude** → similarly, export or copy your saved memories.

Then:

```bash
pickmem import chatgpt-export.json
```

You'll see something like:

```
Parsed:    47
Staged:    47
Routed:    18 (with a suggested_group)
Duplicate: 0 (already in vault, skipped)

Review + accept with: pickmem review
```

- **Parsed** — how many memory items the parser recognized (JSON, bullets, or paragraphs — auto-detected).
- **Staged** — how many landed in your inbox as *pending*.
- **Routed** — how many got a suggested group from the vault's routing rules.
- **Duplicate** — content-hash matches something already in the vault.

**Nothing is active yet.** Everything sits in `pickmem/inbox/` awaiting your review.

### AI-assisted import — split, classify, and merge

Rules-only routing is a blunt instrument: it matches keywords, nothing more, and every imported item becomes exactly one new note. If you want more than that, PickMem can call Claude to do three things beyond what the rules do:

1. **Split each item into atomic claims.** A ChatGPT memory export often bundles several unrelated facts into one entry — *"I moved to Portland in 2024 and I prefer vim over vscode"* becomes two separate staged notes instead of one lumpy one.
2. **Propose a brand-new group when nothing existing fits**, instead of leaving the item unrouted. This is flagged distinctly (`→ NEW: <group>`) so you always see it coming before it lands — it never silently creates a folder.
3. **Suggest merging a claim into a note you already have**, instead of always creating a new one. If you already have a note titled "my cat" and you import *"the cat is now two years old,"* PickMem can suggest folding that sentence into the existing note rather than starting a duplicate.

**This is opt-in, and stays that way even when you turn it on.** Nothing about it changes the core rule: everything still lands in the inbox as a *suggestion*, and nothing is applied to your vault until you accept it in `pickmem review`. A "new group" suggestion doesn't create a folder — it's a proposal you can accept, redirect, or ignore. A "merge" suggestion doesn't touch an existing note — it's a proposal you have to press a key to apply.

**Turning it on** — two ways:

```bash
# Explicit flag, no prompt, good for scripts:
export ANTHROPIC_API_KEY=sk-ant-…
pickmem import chatgpt-export.json --allow-ai

# Or just leave the flag off — if your API key is set and you're at a
# real terminal, PickMem asks once:
pickmem import chatgpt-export.json
# AI-assisted import is available (uses $ANTHROPIC_API_KEY) — split
# memories into finer claims and suggest merges into existing notes? [Y/n]
```

The prompt only appears when `$ANTHROPIC_API_KEY` is set *and* you didn't already pass `--allow-ai` *and* you're running interactively — piped or scripted invocations never see it and silently stay rules-only, so nothing in a script or CI job starts making network calls by surprise.

With AI assistance on, the import summary has a few extra lines:

```
Parsed:    12
Split:     3 source item(s) decomposed into finer claims
Staged:    15
Routed:    11 (with a suggested_group)
New group: 2 suggestion(s) propose a group that doesn't exist yet
Merge:     4 suggestion(s) to fold into an existing note
Duplicate: 0 (already in vault, skipped)

Review + accept with: pickmem review
```

If the API is down, rate-limited, or your key is wrong, PickMem quietly falls back to rules-only for whatever it couldn't classify — an AI outage never fails the import, it just degrades to the same behavior you'd get without `--allow-ai`.

---

## 10. Review the inbox

```bash
pickmem review
```

A TUI opens with every pending item. Keys:

| Key | Action |
|-----|--------|
| `space` | Select at cursor |
| `a` | Accept selected (or cursor row) — moves to group folder, flips to active |
| `A` | Accept every remaining item that has a `suggested_group` (never merges — see below) |
| `m` | Accept as a **merge** into the AI-suggested existing note — no-op on rows without a merge suggestion |
| `r` | Reject selected (or cursor) — deletes the inbox file |
| `g` | Reassign group — overlay: type new, or `tab`/`↓` to pick from existing. Always wins over a prior merge decision. |
| `/` | Filter |
| `enter` | Apply the decisions |
| `q` or `esc` | Cancel — inbox unchanged |

Each row's right-hand side tells you what's about to happen to it:

- `→ financial` — routes into an existing group, plain and ordinary.
- `→ NEW: pets` — the AI is proposing a group that doesn't exist in your vault yet. Accepting this (`a`) creates that folder for the first time; press `g` instead if you'd rather redirect it into something that already exists.
- `→ merge? "my cat"` — the AI thinks this claim belongs inside an existing note called "my cat." Press `m` to fold it in, or `a` to ignore the suggestion and create a separate new note instead.

**`A` (accept-all) deliberately never merges**, even on rows with a merge suggestion — it always creates a fresh note. Merging is a one-row-at-a-time decision you make with `m`, on purpose: it's the one action here that changes an existing note's content, so it doesn't get swept up in a bulk command.

Typical flow for a fresh import:

1. Press `A` to bulk-accept everything the router matched into an existing group. Most of your inbox clears.
2. Walk through the rows still pending. For a `→ merge? "..."` row, press `m` if the suggestion looks right. For a `→ NEW: ...` row, press `a` to accept the new group (or `g` to redirect it somewhere existing). For anything unrouted, press `g`, type a group, press `enter`.
3. Anything you don't want? Select with `space` and press `r`.
4. Press `enter` to apply.

You'll see:

```
Accepted: 38  Merged: 4  Rejected: 3  Left pending: 2
```

Accepted items are now active notes in their group folders. Merged items are gone from the inbox — their text is now part of the note you merged them into. Rejected ones are gone entirely. The two you left pending stay in the inbox for later.

**One thing worth knowing about merges:** they're a plain append — the claim's text gets added to the end of the existing note, separated by a blank line. The AI decides *whether* to merge and *which* note, but it never rewrites or rephrases the existing note's own wording. If you'd hand-edited that note (in Obsidian, say) since the import ran, the merge is refused rather than clobbering your edit — you'll see an error in the terminal and the item just stays in the inbox for you to handle manually.

---

## 11. Lenses — save a selection for a recurring task

A **lens** is a named selection. Instead of re-picking every time you do a task, you save a lens and activate it in one keystroke.

### Save from the CLI TUI

1. `pickmem pick`
2. Toggle the items you'd use for, say, job hunting: your resume, work preferences, current interests.
3. Press `s`, type `Job-Hunt`, press `enter`.
4. Press `enter` again to also make that the active selection.

Look at `pickmem/lenses.json` — your lens is in there. Obsidian sync (or iCloud, Dropbox, git — whatever you use) syncs it across machines because it's just a file.

### Save from the extension

Same idea: toggle items in the popup, type a name in the "lens name" field, click **Save**. Written to the same `lenses.json`.

### Activate a lens

**CLI:** `pickmem pick` → press `l` → pick a lens → `enter`.

**Extension:** click the lens chip at the top of the popup.

**MCP (from inside Claude):** Claude calls `use_lens("Job-Hunt")`.

Activating a lens replaces the current selection with the lens's items and stamps `active_lens` in `active.json`.

### Suggested lenses

- **`Job-Hunt`** — resume, career preferences, target companies.
- **`Client-Acme`** — everything about one client (their stack, personalities, deadlines).
- **`Gift-Sister`** — her preferences, past gifts, budget.
- **`Doctor-Visit`** — health history, current meds, symptoms notebook.
- **`Meal-Plan`** — dietary preferences, allergies, budget.

The theme: recurring tasks that pull from the same slice of your brain.

---

## 12. Edit and organize in Obsidian

You can edit memories directly in Obsidian. Two ways this is safe:

- **PickMem re-reads on every operation.** Your edit in Obsidian shows up next time you run `pickmem list` / `pick` / a Claude query.
- **PickMem's `Store.Update` refuses to overwrite files the user has modified.** It hashes what it last wrote; if that hash doesn't match on disk, it refuses. So you can't accidentally clobber your own edits.

**Reorganizing:**

- **Rename a group folder** in Obsidian — PickMem will get confused because frontmatter still says the old group. Fix: edit each note's frontmatter `group:` field to match the new folder name. Or (easier) update `group:` first, then move the file. Frontmatter is truth.
- **Add tags in Obsidian's tag pane** — PickMem reads the `tags:` frontmatter field. Obsidian's inline `#tag` syntax in the body isn't picked up.
- **Delete a note in Obsidian** — PickMem notices on next reload, quietly removes it from any lens or active selection referencing it.

**Do NOT:**
- Manually edit files under `pickmem/` (inbox, config, lenses, active) — let the tools manage those.
- Create files inside `pickmem/inbox/` yourself; use `pickmem add --inbox` or `pickmem import` instead.

---

## 13. Command reference

Quick reference — the full `--help` on any command has more detail.

### Vault + memory
```
pickmem init <path> [--template personal|developer|researcher] [--force]
pickmem add --label "…" --group … [--body "…" | --file … | stdin | $EDITOR] [--tags a,b] [--inbox]
pickmem list [--group prefix] [--pending] [--all]
pickmem show <id-or-suffix> [--raw]
pickmem edit <id-or-suffix>          # launches $EDITOR
pickmem rm <id-or-suffix> --yes
```

### Picking + delivery
```
pickmem pick                          # TUI to build active.json
pickmem serve                         # MCP server on stdio (Claude Desktop launches this)
pickmem install <client> [--dry-run --bin PATH --name NAME]
pickmem uninstall <client>
```

Clients supported by `install`: `claude-desktop`, `cursor`.

### Ingestion
```
pickmem import <file> [--format auto|json|bullets|paragraphs] [--allow-ai] [--ai-model MODEL]
pickmem review                        # TUI to bulk-accept/reject the inbox
```

### Global flags
```
--vault <path>                        # override vault path (also $PICKMEM_VAULT env)
```

Vault path priority: `--vault` → `$PICKMEM_VAULT` → `~/.config/pickmem/config.json` (recorded by `init`).

---

## 14. Troubleshooting

**"no vault path set"** — you haven't run `pickmem init`, or you're passing `--vault` to something before init. Either init first, or set `$PICKMEM_VAULT`.

**Claude Desktop can't find PickMem after `install`** — restart Claude Desktop. Check `~/Library/Application Support/Claude/claude_desktop_config.json` — the `pickmem` entry should be under `mcpServers`. If it's not, run `pickmem install claude-desktop --dry-run` to see what would be written.

**Extension shows "No vault connected" after granting** — Chrome sometimes doesn't persist handles in incognito/guest profiles. Try a regular profile. If it's a regular profile and still forgetting, `chrome://extensions` → PickMem → Details → Site permissions.

**Extension shows "input not found"** — the target site changed its DOM. Copy fallback still works. The adapter selector is a one-line fix in `extension/src/adapters/index.ts`; file an issue or send a PR.

**`pickmem import` staged 0 items** — the parser didn't recognize the shape. Try `--format bullets` or `--format paragraphs` explicitly. Check the file has non-empty content.

**AI classifier isn't classifying anything** — needs both `--allow-ai` and `$ANTHROPIC_API_KEY`. It also skips the call if your vault has zero groups yet (nothing to classify into) — accept a few items manually first, then re-run import.

**"refusing to overwrite … create-only invariant"** — you edited a file (probably in Obsidian) between PickMem loading it and trying to write it. Not a bug — the guard is protecting your edit. Re-run the operation; PickMem's next load will pick up your changes.

**Test failures after pulling changes** — `go test ./...` and `cd extension && npm test`. If a test fails, the failing test's message tells you which invariant broke.

---

## Where to go next

- **Watch what you use.** After a week of picking, you'll notice which lenses come up repeatedly. Save them.
- **Trim.** Regularly `pickmem list --pending` and clean the inbox — a growing pile is a signal your routing rules need adjusting.
- **Contribute.** Missing an adapter for your favorite chat site? Add one entry to `extension/src/adapters/index.ts`, run `npm run build`, and file a PR.

The thesis in one line: *the model knows exactly what you chose to tell it, this time.*
