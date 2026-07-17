# PickMem — User Guide

*A local-first memory-curation layer for LLMs. You pick what the model remembers.*

PickMem keeps your memory in a plain folder of Markdown files (an Obsidian-compatible vault) on your own disk. For any given task, **you** select which memory items reach the model — nothing is sent automatically. You do that curating and picking in a small **local web app**; two channels then deliver your pick to a model.

---

## Table of contents

- [1. What is PickMem?](#1-what-is-pickmem)
- [2. Prerequisites](#2-prerequisites)
- [3. Install](#3-install)
- [4. Create your first vault](#4-create-your-first-vault)
- [5. The web app](#5-the-web-app)
  - [5.1 Launch it](#51-launch-it)
  - [5.2 First-run setup](#52-first-run-setup)
  - [5.3 The three zones](#53-the-three-zones)
  - [5.4 Browse & navigate](#54-browse--navigate)
  - [5.5 Pick what the model sees](#55-pick-what-the-model-sees)
  - [5.6 Add, edit, and open notes](#56-add-edit-and-open-notes)
  - [5.7 Organize groups](#57-organize-groups)
  - [5.8 Lenses](#58-lenses)
  - [5.9 The inbox](#59-the-inbox)
  - [5.10 Switch and manage vaults](#510-switch-and-manage-vaults)
  - [5.11 Settings](#511-settings)
- [6. Use PickMem in the browser (extension)](#6-use-pickmem-in-the-browser-extension)
- [7. Connect an assistant (MCP)](#7-connect-an-assistant-mcp)
- [8. Let Claude save memories for you](#8-let-claude-save-memories-for-you)
- [9. Import a batch of memories](#9-import-a-batch-of-memories)
- [10. Editing and organizing in Obsidian](#10-editing-and-organizing-in-obsidian)
- [11. CLI reference (scripting & agents)](#11-cli-reference-scripting--agents)
- [12. Troubleshooting](#12-troubleshooting)

---

## 1. What is PickMem?

PickMem inverts how assistant "memory" usually works. Instead of the system silently deciding what past context to inject, **the default is nothing, and you add context on purpose.** Switch the assistant's own memory off, and the model is personalized only by the items you deliberately pick for that session — never by stored context behind the scenes.

- Your memories live in a folder of Markdown notes. Each note is one memory item.
- You open the **web app**, browse your vault, and toggle the items relevant to your current task. That selection is written to a small file (`pickmem/active.json`).
- Two channels deliver *only that selection* to a model:
  - **Chrome extension** — injects your selection into the chat box on ChatGPT, Claude.ai, or Gemini (and a **Copy** button works anywhere).
  - **MCP** — a local server (`pickmem serve`) that assistants and AI agents (Claude Desktop, Cursor, Cline) connect to.

Both channels read the same vault and produce the same context block, so switching between them doesn't change what the model sees.

---

## 2. Prerequisites

- **macOS, Linux, or Windows.** Commands below use macOS paths; the equivalents work on other platforms.
- **A modern browser** — the web app runs in Chrome, Safari, Firefox, or Edge.
- **Go 1.26+** — only if you build from source (`go version`; install from [go.dev/dl](https://go.dev/dl) or `brew install go`). The one-liner installer needs no Go.
- **Obsidian** (optional) — for browsing/editing the vault visually. PickMem works without it; the vault is just files.
- **Node.js 20+ and Chrome** — only if you want the Chrome extension.

You do **not** need Docker, an account, an API key, or a network connection — everything runs locally.

---

## 3. Install

**Easiest — the install script** (macOS/Linux; no Go needed). Downloads the right prebuilt binary from the latest release, verifies its checksum, and installs it. The web UI is embedded in the binary:

```bash
curl -fsSL https://raw.githubusercontent.com/kaiagaoo/PickMem/main/install.sh | sh
```

It installs to `/usr/local/bin` if writable, else `~/.local/bin`. Overrides: `PICKMEM_VERSION=v0.1.1` to pin, `PICKMEM_INSTALL_DIR=…` for a custom target. Windows: grab the `.zip` from the [releases page](https://github.com/kaiagaoo/PickMem/releases/latest).

**Or build from source** (needs Go 1.26+):

```bash
git clone https://github.com/kaiagaoo/PickMem.git
cd PickMem
go install ./cmd/pickmem      # → $(go env GOPATH)/bin — make sure that's on PATH
```

Verify:

```bash
pickmem --help
```

> **macOS note:** if a freshly built binary ever hangs on launch with no output, macOS Gatekeeper may have flagged it. Rebuild it directly (`go build -o /usr/local/bin/pickmem ./cmd/pickmem`) and confirm with `spctl -a -vv /usr/local/bin/pickmem`.

---

## 4. Create your first vault

A **vault** is just a folder. PickMem uses it as its only store — there is no database.

```bash
pickmem init ~/PickMemVault
```

By default `init` lays down a **starter taxonomy**: a tree of group folders, a `pickmem/config.json` with keyword→group routing rules, a root `README.md` describing every group, and **one fill-in-the-blank note per group** (tagged `starter`) so the vault reads as a form to complete rather than an empty tree.

The starter groups nest by life area:

| Top level | Sub-groups |
|-----------|-----------|
| `about/` | `identity`, `preferences`, `health` |
| `work/` | `role`, `projects`, `stack`, `contacts` |
| `finance/` | `income`, `bills`, `goals` |
| `home/` | `housing`, `logistics` |
| `relationships/` | `family`, `friends`, `dates` |
| `learning/` | `topics`, `resources` |
| `projects/` | *(flat)* |

Other useful forms:

```bash
pickmem init ~/PickMemVault --bare     # empty vault, no taxonomy (the web app's setup guide will greet you)
pickmem init ~/PickMemVault --force    # re-apply the starter taxonomy to an existing vault (never overwrites your notes)
```

You can point `init` at a subfolder of an existing Obsidian vault (`pickmem init ~/ObsidianVault/memory`) or at the vault root — PickMem only creates and manages files it owns and never touches your other notes.

**After `init`, the vault path is remembered** in `~/.config/pickmem/config.json` (or `$XDG_CONFIG_HOME/pickmem/config.json`), so later commands don't need `--vault`. You can also create and switch vaults from inside the web app (§5.10). The resolution order for every command except `init` is: `--vault` → `$PICKMEM_VAULT` → recorded user config.

---

## 5. The web app

The web app is where you live day-to-day: build memory, organize it, and pick.

### 5.1 Launch it

```bash
pickmem web
```

This starts a local server (bound to `127.0.0.1`, no network access) and opens your browser at `http://127.0.0.1:4577`. It uses the vault the CLI resolves (`--vault` → `$PICKMEM_VAULT` → recorded config) and records it as the current + recent vault.

| Flag | Default | Meaning |
|------|---------|---------|
| `--port` | `4577` | Port to listen on |
| `--host` | `127.0.0.1` | Interface to bind (localhost only by default) |
| `--no-open` | off | Don't open a browser automatically (just print the URL) |
| `--vault <path>` | — | Open a specific vault instead of the recorded one |

Leave the terminal running; press **Ctrl-C** to stop. The app reloads the vault from disk on every action, so edits you make in Obsidian or the CLI show up on the next click.

### 5.2 First-run setup

Open an **empty** vault (say, one made with `init --bare`, or a brand-new folder) and the app greets you with a short **setup guide**:

1. **Choose areas** — pick the life areas your memory should help with (About you, Work, Preferences, Money, Health, People, Learning). Each becomes a group.
2. **Fill in memories** — each area shows a few suggested memories with a guiding question. Type an answer, tap **use example** to prefill one, or **fill all with examples** for a whole area. Skip anything that doesn't fit.
3. **Review & create** — the memories you filled start out **picked**, saved as a lens called **Starter**, and you land in the workspace.

The setup guide is entirely optional — **Skip for now** leaves you with just the groups you chose and a friendly empty state. It only appears for a genuinely empty vault; a vault that already has notes drops you straight into the workspace. (Choose **"I already have a vault"** on the welcome screen to import a PickMem vault export instead.)

### 5.3 The three zones

```
┌───────────────┬─────────────────────────────────────┬───────────────────────┐
│  navigate     │           browse & pick             │    active memory      │
│  (left)       │           (center)                  │    (right)            │
└───────────────┴─────────────────────────────────────┴───────────────────────┘
```

- **Left — navigate.** The vault switcher (top), your **folder tree** (groups, collapsible), your **lenses**, the **Inbox** (with a count), **Settings**, and an inert **✨ Suggestions** entry for the future AI feature.
- **Center — browse & pick.** A drill-down view of wherever you are: a breadcrumb, the current group's subgroups (as folders you open) and its notes (as cards). Clicking a note opens its detail.
- **Right — active memory.** The current pick: a live list of what you selected, a rough token estimate, **Copy context**, **Save as lens**, and **Clear pick**. This tray is *exactly* what a model receives.

### 5.4 Browse & navigate

The left **tree** is your folder map — groups only, nested, collapsible. Click a group name to open it in the center; click the caret to expand/collapse.

The **center** shows one group at a time (drill-down):

- A **breadcrumb** (`All memories / finance / income`) — click any segment to jump up.
- The group's **subgroups** as folder cards — click one to go deeper.
- The group's **notes** as cards.

This keeps the center focused even for a large vault; the full structure stays browsable in the left tree.

### 5.5 Pick what the model sees

Picking is the point of PickMem, and it's one click:

- **A note** — click its card (or the circular toggle). Picked cards turn the accent color; they appear instantly in the Active Memory tray.
- **A whole group** — use the checkbox next to any group, in the left tree or on a folder card. It's tri-state (none / some / all) and toggles every note in that group and its subgroups.
- **Everything shown** — the "Pick all / Clear" controls above a group's notes toggle just the notes in view.

The **Active Memory** tray on the right shows the running selection, an item count, and a **~token estimate** so you feel the size of what you're handing over. When you're ready:

- **Copy context** — assembles your pick into clean text and copies it. A **text / markdown / json** toggle picks the format. This is the universal delivery path — paste it into any chat, no extension needed. (The plain-text form is byte-identical to what the MCP server and extension produce.)
- With nothing picked, the block reads `--- pickmem: no memory selected ---` — the "default is nothing" state.

### 5.6 Add, edit, and open notes

**Add** — click **+ Memory** (in a group's header, so it's pre-filed there) or use **Add memory here** from a group's ⋯ menu. The editor has a **label**, the **memory** text, a **group** (type a new `a/b` path or pick an existing one), and optional **tags** — with one-click **suggested tag chips** (`fact` / `idea` / `thought` / `reference` by default; customize the list in Settings). Save routes the note to **active** by default; a de-emphasized **"Send to inbox instead"** stages it for later review.

**Open** — click a note's **title** to open its detail view: the full body, tags, provenance, and **Edit / Delete** actions, plus a pick toggle.

**Edit** — the **Edit** button (on a card or in the detail view) reopens the editor. Changing the group **moves** the note to the new folder.

**Delete** — removes the note's file (with a confirm). Deleting also drops the note's id from any lens or the active selection that referenced it.

> **Tags, not types.** A note isn't only "a fact about me." Tag it `idea` or `thought` and you can later filter "my *ideas* about this project" apart from stable facts. The suggested chips are ordinary tags — nothing special on disk — so tag freely and rename the suggestions in Settings.

### 5.7 Organize groups

Your **taxonomy is your folder tree**, and you shape it right in the app. Every group — in the left tree or the center header — has a **⋯ menu**:

- **New subgroup…** — create a nested group (e.g. `work/client-acme`).
- **Add memory here** — open the editor pre-filed to this group.
- **Rename group…** — moves the group and *every note and subgroup under it* to the new path, updating each note's `group` and its routing rules.
- **Delete group…** — deletes the folder and all notes under it (a typed-`DELETE` confirm when it isn't empty).

The **+ group** button (top of the tree) and **+ Subgroup** (center header) create groups too. All of this happens in-app with a small dialog — no terminal needed.

### 5.8 Lenses

A **lens** is a named, saved selection — for a recurring task you don't want to re-pick every time (`Advice`, `Job-Hunt`, `Client-Acme`).

- **Save** — pick some items, then type a name in the tray's **save pick as lens…** field.
- **Activate** — click a lens in the left sidebar; it replaces the current pick. The tray's lens dropdown switches between lenses too.
- **Unsaved changes** — if you toggle items after activating a lens, the tray offers **Update lens** or **Save as new**.
- **Manage** — the sidebar's **Lenses → manage** opens a screen to activate, **edit membership** (jumps you to the pick), **rename**, **duplicate**, or **delete** each lens.

Lenses live in `pickmem/lenses.json`, so they travel with your vault.

### 5.9 The inbox

The **Inbox** (left sidebar, with a count badge) holds **pending** items — memories staged for review rather than added directly. This is where captured pages, imports, and memories an assistant stages for you land (§8). For each item you can:

- **file into** a group and **Accept** — files the note into that folder and makes it active,
- **Edit** first, or
- **Discard** it.

Nothing in the inbox is ever active until you accept it — the same guarantee as `pickmem review` on the CLI.

### 5.10 Switch and manage vaults

You can keep several vaults (work, personal, a shared project) and move between them. The **vault switcher** at the top-left shows the current vault and drops down:

- **Recent vaults** — click one to switch. The pick, tree, and everything else reload for that vault.
- **Open a folder…** — open an existing folder as a vault by pasting its path (e.g. `~/vaults/work`).
- **Create new vault…** — make a fresh, empty vault at a path and switch to it.
- **Import a vault…** — create a new vault from a PickMem vault export (a `pickmem-vault.json`) and switch to it.

All vaults are plain folders on your machine; paths are resolved on the computer running `pickmem web`. Switching also sets that vault as your CLI default, so the two surfaces stay in sync.

### 5.11 Settings

The **Settings** screen (left sidebar) has:

- **Vault** — a display name for the vault and its on-disk path.
- **Appearance** — System / Light / Dark theme.
- **AI extraction** — a preview of the upcoming suggestion feature (see §8 for what already works today via MCP).
- **Danger zone → Clear vault** — deletes every PickMem note and lens in the current vault (a typed-`DELETE` confirm). Your non-PickMem Obsidian notes are untouched.

---

## 6. Use PickMem in the browser (extension)

The extension inserts your pick directly into the chat box on ChatGPT, Claude.ai, and Gemini, with a clipboard fallback everywhere else. It reads the same vault as the web app.

**Build and load:**

```bash
cd extension/
npm install
npm run build          # writes extension/dist/
```

In Chrome: open `chrome://extensions`, enable **Developer mode**, click **Load unpacked**, and select `extension/dist/`.

**Connect your vault.** Click the PickMem toolbar icon. First run shows **"Choose vault folder…"** — grant access to the same folder you `init`-ed (via the browser's File System Access API). Chrome remembers the grant. The popup shows the vault name with a **switch** button and an **inbox N** badge when captures are waiting.

**Pick and deliver.** The popup shows your groups and notes as a **tree with group checkboxes** (tri-state, click a group to select its subtree). Saved lenses appear as chips; the bottom field saves the current selection as a new lens.

- **Insert** — on ChatGPT / Claude.ai / Gemini, prepends the assembled block into the chat input (your draft is preserved).
- **Copy** — puts the block on your clipboard. Works on **any** site, and even with no vault connected.

**Add a memory** with **+ Add memory** (label, group, optional tags, text) — same result as adding in the web app, and it selects the new note for you.

The extension only ever **creates** notes and writes `active.json` / `lenses.json`. Editing existing notes stays in the web app, Obsidian, or the CLI. If a site changes its markup and the input can't be found, the popup says so and **Copy** still works.

---

## 7. Connect an assistant (MCP)

MCP is how assistants and AI agents read your pick. Install the server entry into a client's config (non-destructive — it merges, preserving other servers):

```bash
pickmem install claude-desktop            # ~/Library/Application Support/Claude/claude_desktop_config.json (macOS)
pickmem install claude-desktop --dry-run  # preview the entry without writing
pickmem install cursor                    # ~/.cursor/mcp.json
pickmem uninstall claude-desktop          # remove the entry
```

Supported for `install`: **`claude-desktop`** and **`cursor`**. For **Cline**, add it by hand — command: your `pickmem` binary, args: `serve`. Then fully quit and reopen the client. The server it launches is `pickmem serve` (stdio).

**What the server exposes:**

| Resource / Tool | Purpose |
|-----------------|---------|
| resource `pickmem://active` | The assembled context block for your current pick |
| `get_active_memory` | Returns the same block via a tool call |
| `list_lenses` | Lists your saved lenses (`name`, item count) |
| `use_lens(name)` | Activates a lens — rewrites `active.json` and returns the new block |
| `list_groups` | Lists your vault's groups (your folder tree + note groups + routing targets) so the model classifies into your real taxonomy. Folders prefixed with `_` are private and never sent. |
| `stage_memories(items)` | **The main save path.** Claude extracts memory-worthy facts itself — label + body + `suggested_group` per item — and stages them to the inbox as **pending**. Never activates; you review (§8). |
| `propose_memories(chat_text)` | Fallback bulk-stage for raw text. Prefer `stage_memories`. |

**Recommended Claude Desktop settings.** Two client-side settings remove friction (neither adds anything beyond what you picked):

1. **Tool permissions → Always allow.** Settings → Connectors → pickmem → Tool access: set the tools to *Always allow* so Claude doesn't stop to confirm each read.
2. **A custom instruction.** Settings → Profile → personal preferences:
   > *Before answering anything that might depend on my personal context or preferences, check my PickMem active memory first (`get_active_memory` or the `pickmem://active` resource). If it's empty or unrelated, say so rather than guessing.*

**Testing it.** Pick a few items in the web app, then in a new conversation ask something that depends on your context. The server reloads the vault on every call, so a fresh pick (or an Obsidian edit) is visible without restarting.

The assembled block looks like this (same on every channel):

```
--- pickmem: selected memory ---
salary (finance/income): monthly base $8k plus quarterly bonus

anemia (about/health): chronic anemia, on iron supplements
--- end pickmem memory ---
```

---

## 8. Let Claude save memories for you

With the MCP server connected, saving stops being manual sentence-picking. Say *"remember that…"* or just share something durable, and — steered by the server's instructions — Claude calls `stage_memories`: it extracts each fact, condenses it to a label + body, picks a `suggested_group` from your existing taxonomy, and stages everything to your **inbox** as pending. Invalid groups are rejected (staging can't invent taxonomy); duplicates of existing vault content are skipped.

Your part shrinks to **review**: open the **Inbox** in the web app (§5.9) — file each item and **Accept**, or **Discard** — or sweep them on the CLI:

```bash
pickmem review
```

Nothing an assistant stages ever goes live until you accept it.

> Fully automatic extraction (PickMem noticing memories as you work and proposing them itself) is the planned next step — the inert **Suggestions** entry in the web app reserves its place. Today, extraction is assistant-driven through `stage_memories`.

---

## 9. Import a batch of memories

To bring in many items at once — a memory export from another assistant, or your own list — write them to a file and import from the CLI. Each item is staged to the **inbox** as pending; nothing goes active until you review.

```bash
pickmem import memories.txt
```

The parser auto-detects the file shape (JSON array of strings; JSON objects with a `memory`/`text`/`content`/`body` field; a `{"memories": [...]}` wrapper; a Markdown bullet/numbered list; or blank-line-separated paragraphs) and takes each chunk as one item. Override with `--format json|bullets|paragraphs|auto`. Items are routed with your vault's keyword rules and de-duplicated on a content hash.

```
Parsed:    47      # chunks recognized
Staged:    47      # staged to the inbox as pending
Routed:    18      # of those, how many got a suggested_group
Duplicate: 0       # skipped: already in the vault
```

Then review the pending items in the web app's **Inbox**, or with `pickmem review`.

**Undoing an import** (only pending items are ever affected — active notes can't be touched):

```bash
pickmem inbox clear --source import --yes   # delete only import-staged items
pickmem inbox clear --yes                   # …or everything pending
```

Run it without `--yes` first to preview.

---

## 10. Editing and organizing in Obsidian

Every memory note is a normal Markdown file with a YAML frontmatter block. Open the vault in Obsidian and edit freely — with two things to know:

**Create-only.** PickMem only ever creates notes and moves inbox notes into group folders. When it *does* edit a note (via the web app or CLI), it first checks that the on-disk content still matches what it last wrote and refuses if you changed it out from under it. So your Obsidian edits are never silently clobbered.

**Ids are assigned by PickMem.** Each note needs a stable `id` in its frontmatter, and every add path (web app, `pickmem add`, `import`, the extension) generates it for you. Practical consequences:

- A note with **no `---` frontmatter block** is ignored by PickMem — a normal Obsidian note.
- A note with a **frontmatter block missing its `id`** will cause vault loads to warn until you fix or delete it. So: create memory items through PickMem, not by typing a half-frontmatter file.

**Your taxonomy is your folder tree.** Make a folder in Obsidian and it becomes a category — the same as creating a group in the web app. Every directory under the vault (except `pickmem/`) is a group PickMem knows about.

**Keeping a category private.** Because folder names go to the model via `list_groups`, prefix a folder with `_` (`_medical/`, `_finance/`) to keep it *out* of the shared list. You can still file notes there by hand; the name never leaves your disk. (Dot-folders like `.obsidian` are excluded too.)

**Don't hand-edit** files under `pickmem/` (`config.json`, `lenses.json`, `active.json`, `inbox/`) — let the tools manage those.

---

## 11. CLI reference (scripting & agents)

Everything the web app does is also available headless — useful for scripting, automation, and for AI agents driving PickMem programmatically. `pickmem pick` and `pickmem review` are interactive terminal UIs; the web app is the recommended surface for humans, but both remain fully supported.

The vault path for every command except `init` resolves as `--vault` → `$PICKMEM_VAULT` → recorded user config.

```
pickmem init <path> [--bare] [--force]         # create a vault; applies the starter taxonomy unless --bare
pickmem web [--port 4577] [--host 127.0.0.1] [--no-open]   # launch the web app

pickmem add --label "…" [--group …] [--tags idea,money]
            [--body "…" | --file <path|-> | (stdin) | ($EDITOR)] [--inbox]
pickmem list [--group <prefix>] [--pending] [--all]                     # alias: ls
pickmem show <id-or-suffix> [--raw]
pickmem edit <id-or-suffix>                    # opens $EDITOR
pickmem rm <id-or-suffix> --yes

pickmem pick                                   # terminal picker → writes active.json
pickmem status                                 # vault summary + current selection
pickmem context [--copy]                       # print (or copy) the assembled block
pickmem lens list | use <name> | rm <name>     # manage saved lenses
pickmem inbox clear [--source import|extract|manual] --yes   # bulk-delete pending items

pickmem serve                                  # MCP stdio server (clients launch this)
pickmem install <claude-desktop|cursor> [--dry-run] [--name <n>] [--bin <path>]
pickmem uninstall <claude-desktop|cursor> [--name <n>]

pickmem import <file> [--format auto|json|bullets|paragraphs]   # stages parsed items to the inbox
pickmem review                                 # terminal UI to accept/reject/reassign inbox items

--vault <path>                                 # global override on any command
```

---

## 12. Troubleshooting

**"no vault path set"** — you haven't `init`-ed, or you're running a command before init. Run `pickmem init <path>`, set `$PICKMEM_VAULT`, or pass `--vault`.

**`pickmem web` says "address already in use"** — another server (or a stale one) holds the port. Start on another with `pickmem web --port 8080`, or stop the other instance.

**The web app won't load / "can't reach the PickMem server"** — the `pickmem web` process stopped. Restart it; the terminal running it must stay open.

**"warning: skipped <file>: …"** — a `.md` file has a `---` block with a missing/malformed `id`, or a duplicate id (usually a half-typed note in Obsidian). It's skipped, everything else works; fix or delete it. (Files with no frontmatter are ignored silently.)

**"refusing to overwrite … create-only"** — a file changed on disk (e.g. in Obsidian) between PickMem reading and writing it. That's the guard protecting your edit; re-run and it'll re-read your version.

**Claude Desktop doesn't call PickMem tools** — confirm the connector is listed and connected (Settings → Connectors), set its tools to *Always allow*, and add the custom instruction from §7. Fully quit and relaunch the app after `pickmem install`, not just close the window.

**`pickmem import` staged 0 items** — the parser didn't recognize the file shape. Try `--format bullets` or `--format paragraphs`, and check the file isn't empty.

**Extension shows "input not found"** — the target site changed its markup. Insert is disabled but **Copy** still works; the adapter selector is a one-line fix in `extension/src/adapters/index.ts`.

---

*PickMem in one line: the model knows exactly what you chose to tell it, this time.*
