# PickMem — User Guide

*A local-first memory-curation layer for LLMs. You pick what the model remembers.*

PickMem keeps your memory in a plain folder of Markdown files (an Obsidian vault) on your own disk. For any given task, **you** select which memory items reach the model — nothing is sent automatically. This guide is written to match exactly how the tool behaves; follow it top to bottom the first time.

---

## Table of contents

- [PickMem — User Guide](#pickmem--user-guide)
  - [Table of contents](#table-of-contents)
  - [1. What is PickMem?](#1-what-is-pickmem)
  - [2. Prerequisites](#2-prerequisites)
  - [3. Install the CLI](#3-install-the-cli)
  - [4. Create your first vault](#4-create-your-first-vault)
  - [5. Add memories](#5-add-memories)
    - [Inspecting](#inspecting)
    - [Editing / deleting](#editing--deleting)
  - [6. Pick what the model sees](#6-pick-what-the-model-sees)
  - [7. Use PickMem with Claude Desktop (MCP)](#7-use-pickmem-with-claude-desktop-mcp)
    - [Recommended Claude Desktop settings](#recommended-claude-desktop-settings)
  - [8. Use PickMem in the browser (extension)](#8-use-pickmem-in-the-browser-extension)
    - [Build and load](#build-and-load)
    - [Connect your vault](#connect-your-vault)
    - [Pick and deliver](#pick-and-deliver)
    - [Add a memory from the popup](#add-a-memory-from-the-popup)
  - [9. Import a batch of memories](#9-import-a-batch-of-memories)
    - [Undoing an import](#undoing-an-import)
  - [10. Review the inbox](#10-review-the-inbox)
  - [11. Lenses](#11-lenses)
  - [12. Editing and organizing in Obsidian](#12-editing-and-organizing-in-obsidian)
  - [13. Command reference](#13-command-reference)
  - [14. Troubleshooting](#14-troubleshooting)

---

## 1. What is PickMem?

PickMem inverts how assistant "memory" usually works. Instead of the system silently deciding what past context to inject, **the default is nothing, and you add context on purpose.** Think of it as a replacement for built-in auto-memory: switch the assistant's own memory off, and the model is personalized only by the items you deliberately pick for that session — never by stored context behind scenes.

- Your memories live in a folder of Markdown notes. Each note is one memory item.
- You run a picker, select the items relevant to your current task, and confirm. That selection is written to a small file (`pickmem/active.json`).
- Two channels deliver *only that selection* to a model:
  - **MCP** — a local server (`pickmem serve`) that Claude Desktop, Cursor, and Cline connect to.
  - **Chrome extension** — reads the same vault and injects your selection into the chat box on ChatGPT, Claude.ai, or Gemini. A **Copy** button works on any other site.

Both channels read the same vault and produce the same context block, so switching between them doesn't change what the model sees.

---

## 2. Prerequisites

- **macOS, Linux, or Windows.** Commands below use macOS paths; the equivalents work on other platforms.
- **Go 1.26 or newer** — the CLI is written in Go. Check with `go version`; install from [go.dev/dl](https://go.dev/dl) or `brew install go`.
- **Obsidian** (optional but recommended) — for browsing/editing the vault visually. PickMem works without it; the vault is just files.
- **Node.js 20+ and npm** — only if you want the Chrome extension.
- **Chrome or another Chromium browser** — only for the extension.

You do **not** need Docker, an account, an API key, or a network connection — everything runs locally.

---

## 3. Install the CLI

**Easiest — the install script** (macOS/Linux; no Go needed). Downloads the right prebuilt binary from the latest release, verifies its checksum, and installs it:

```bash
curl -fsSL https://raw.githubusercontent.com/kaiagaoo/PickMem/main/install.sh | sh
```

It installs to `/usr/local/bin` if writable, else `~/.local/bin`. Env overrides: `PICKMEM_VERSION=v0.1.1` to pin, `PICKMEM_INSTALL_DIR=…` for a custom target. Windows: grab the `.zip` from the [releases page](https://github.com/kaiagaoo/PickMem/releases/latest).

**Or build from source** (needs Go 1.26+):

```bash
git clone https://github.com/kaiagaoo/PickMem.git
cd PickMem
go build -o pickmem ./cmd/pickmem
```

Put the binary on your PATH:

```bash
go install ./cmd/pickmem      # installs to $(go env GOPATH)/bin — make sure that's on PATH
# or copy it:  cp pickmem /usr/local/bin/
```

Verify:

```bash
pickmem --help
```

You should see these subcommands: `init`, `add`, `list`, `show`, `edit`, `rm`, `pick`, `status`, `context`, `lens`, `inbox`, `serve`, `install`, `uninstall`, `import`, `review`.

> **macOS note:** if you rebuild often and the binary ever hangs on launch with no output, macOS Gatekeeper may have flagged it. Rebuild it yourself directly (`go build -o /usr/local/bin/pickmem ./cmd/pickmem`) rather than through another tool, and confirm with `spctl -a -vv /usr/local/bin/pickmem`.

---

## 4. Create your first vault

A **vault** is just a folder. PickMem uses it as its only store — there is no database.

```bash
pickmem init ~/PickMemVault
```

By default `init` lays down a **starter taxonomy**: a tree of group folders, a `pickmem/config.json` seeded with keyword→group routing rules, and a `README.md` at the vault root describing every group. (PickMem ignores that README — it has no frontmatter — so it's just a map for you.)

It also seeds **one fill-in-the-blank note per group** (tagged `starter`), so the vault starts as a form to complete instead of an empty tree:

```
Monthly income: ____
Other sources: ____
```

Open them in Obsidian or with `pickmem edit <id>` and replace the `____` blanks; delete the ones you don't care about with `pickmem rm <id> --yes`. Unfilled blanks are harmless — you just won't pick those notes. Re-running `init --force` restores missing skeletons but **never duplicates or overwrites a note you've been filling in** (it skips any group+label that already exists).

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
pickmem init ~/PickMemVault --bare     # empty vault, no taxonomy
pickmem init ~/PickMemVault --force    # re-apply the starter taxonomy to an existing vault
```

You can point `init` at a subfolder of an existing Obsidian vault (`pickmem init ~/ObsidianVault/memory`) or at the vault root — PickMem only creates and manages files it owns and never touches your other notes.

**After `init`, the vault path is remembered** in `~/.config/pickmem/config.json` (or `$XDG_CONFIG_HOME/pickmem/config.json`), so later commands don't need `--vault`. The resolution order for every command except `init` is:

1. `--vault <path>` flag
2. `$PICKMEM_VAULT` environment variable
3. the recorded user config

---

## 5. Add memories

The one-at-a-time command is `pickmem add`. `--label` is always required; `--group` is required unless you're staging to the inbox.

```bash
# inline body
pickmem add --label "salary" --group finance/income --body "monthly base \$8k plus quarterly bonus"

# with tags
pickmem add --label "editor preference" --group about/preferences --tags tools,strong \
  --body "I use vim. Don't suggest vscode extensions for vim problems."

# body piped from stdin
echo "chronic anemia, on iron supplements" | pickmem add --label "anemia" --group about/health

# body from a file
pickmem add --label "kickoff notes" --group work/projects --file notes.txt

# no body source given + a terminal → opens $EDITOR
pickmem add --label "meeting notes" --group work/projects
```

Body source precedence: `--body`, then `--file` (`-` means stdin), then piped stdin, then `$EDITOR`.

Groups nest with `/` (e.g. `work/projects`). `pickmem add` generates the note's stable id for you — this is why you normally add through the CLI rather than hand-creating files (see §12).

**Note types.** A note isn't only "a fact about me." Pass `--type` to say what kind it is — `fact` (default), `idea`, `thought`, or `reference` — so you can later pick "my *ideas* about this project" separately from stable facts. The type is independent of the group (where it lives): `pickmem add -l "solo sail" -g projects -t idea -b "try a solo overnight sail"`. The default `fact` is left off disk, so plain memories stay clean; `pickmem list --type idea` filters by kind, and typing a kind name in the picker's filter narrows to it.

**Stage to the inbox instead of adding directly** with `--inbox`; the `--group` you pass becomes a *suggested* group the note carries until you accept it in review (§10):

```bash
echo "solar quotes to compare" | pickmem add --label "solar research" --group home --inbox
```

### Inspecting

```bash
pickmem list                     # active notes, grouped
pickmem list --pending           # only inbox (pending) notes
pickmem list --all               # active + pending
pickmem list --group finance     # only groups matching this prefix
pickmem show <id-or-suffix>      # print one note (accepts the last 3+ chars of the id)
pickmem show <id-or-suffix> --raw   # print the raw file, frontmatter included
```

### Editing / deleting

```bash
pickmem edit <id-or-suffix>      # opens the note in $EDITOR ($VISUAL, then vi as fallbacks)
pickmem rm <id-or-suffix> --yes  # delete (the --yes is required)
```

`pickmem edit` launches your editor on the file (`$VISUAL`, then `$EDITOR`, then `vi`) — PickMem doesn't rewrite the bytes itself. Deleting also removes the note's id from any lens or the active selection that referenced it.

---

## 6. Pick what the model sees

```bash
pickmem pick
```

This opens a full-screen picker showing your active notes as a **tree**: group headers nest by path, with notes indented underneath.

```
[ ] about
    [ ] health
        [ ] chronic anemia
    [ ] preferences
        [ ] editor preference
[ ] finance
    [ ] income
        [ ] salary
```

Every row — including group headers — has a checkbox.

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | Move the cursor (lands on headers and notes) |
| `space` | Toggle the row at the cursor. On a **note**, toggles that note. On a **group header**, selects every note in that group's subtree — or clears them all if they're already all selected. |
| `/` | Filter — matches note label + body + tags (fuzzy); ancestor group headers stay visible when a nested note matches |
| `l` | Lens overlay (opens only if you have saved lenses) |
| `s` | Save the current selection as a new lens (prompts for a name) |
| `enter` | Confirm — writes `pickmem/active.json` and exits |
| `q` / `esc` | Cancel — `active.json` is left unchanged |

Group-header checkboxes are tri-state: `[ ]` none of its notes selected, `[~]` some, `[x]` all.

The footer shows `Active: <lens|custom|none> · N selected · ~T tokens` (a rough token estimate over the selected bodies).

**Confirming with nothing selected clears `active.json`** — this is intentional; it matches the "default is nothing" rule and is how you reset. If you didn't mean to, press `q` to cancel instead.

Only **active** notes appear in the picker. Pending inbox items must be accepted in `pickmem review` (§10) first.

**Checking what's active without opening the picker:**

```bash
pickmem status           # vault, note counts, current selection, ~tokens
pickmem context          # print the exact block the model receives
pickmem context --copy   # …or copy it, to paste into any chat UI
```

`pickmem context --copy` is a delivery channel of its own: it assembles the same block the MCP server and extension produce, so you can paste your picked memory into any site — no extension needed.

---

## 7. Use PickMem with Claude Desktop (MCP)

Install the MCP server entry into a client's config (non-destructive — it merges, preserving other servers):

```bash
pickmem install claude-desktop            # ~/Library/Application Support/Claude/claude_desktop_config.json (macOS)
pickmem install claude-desktop --dry-run  # preview the entry without writing
pickmem install cursor                    # ~/.cursor/mcp.json
pickmem uninstall claude-desktop          # remove the entry
```

Supported clients for `install`: **`claude-desktop`** and **`cursor`**. For **Cline**, add it by hand in Cline's MCP settings — command: your `pickmem` binary, args: `serve`.

Then fully quit and reopen the client. The server it launches is `pickmem serve` (stdio).

**What the server exposes:**

| Resource / Tool | Purpose |
|-----------------|---------|
| resource `pickmem://active` | The assembled context block for your current pick |
| `get_active_memory` | Returns the same block via a tool call |
| `list_lenses` | Lists your saved lenses (`name`, item count) |
| `use_lens(name)` | Activates a lens — rewrites `active.json` and returns the new block |
| `list_groups` | Lists your vault's groups — **your folder tree** (plus note groups + routing-rule targets) — so the model classifies into your real, self-defined taxonomy. Folders prefixed with `_` are private and never sent. |
| `stage_memories(items)` | **The main save path.** Claude extracts the memory-worthy facts itself — one label + body + `suggested_group` per item — and stages them to the inbox as **pending**. Invalid groups are rejected (staging can't invent taxonomy), duplicates of existing vault content are skipped. Never activates; `pickmem review` is still the gate. |
| `propose_memories(chat_text)` | Fallback bulk-stage for raw text: splits on paragraphs, rules-based only. Prefer `stage_memories`. |

With `stage_memories`, saving stops being manual sentence-picking: say *"remember that"* or just share something durable, and (steered by the server's instructions) Claude extracts each fact, picks a group from your existing taxonomy, and stages everything to the inbox. Your part shrinks to `pickmem review` — press `A` to sweep routed items in.

**Testing it:** run `pickmem pick`, select a few items, confirm. In a new Claude Desktop conversation, ask something that depends on your context. The server reloads the vault on every call, so a fresh pick (or an Obsidian edit) is visible without restarting.

**The assembled block** looks like this (plain text, same on the extension side):

```
--- pickmem: selected memory ---
salary (finance/income): monthly base $8k plus quarterly bonus

anemia (about/health): chronic anemia, on iron supplements
--- end pickmem memory ---
```

With no selection it's `--- pickmem: no memory selected ---`.

### Recommended Claude Desktop settings

Two client-side settings make this reliable. Neither adds anything beyond what you picked — they just remove friction:

1. **Tool permissions → Always allow.** Settings → Connectors → pickmem → Tool access. Set all four tools to *Always allow* so Claude doesn't stop to confirm each read. (One tool may default to "Ask" on first connect — that's Claude Desktop's own behavior, not a PickMem setting; flip it too.)
2. **A custom instruction. (Optional)** Settings → Profile → personal preferences, add something like:
   > *Before answering anything that might depend on my personal context or preferences, check my PickMem active memory first (`get_active_memory` or the `pickmem://active` resource). If it's empty or unrelated, say so rather than guessing.*

---

## 8. Use PickMem in the browser (extension)

For ChatGPT, Claude.ai, and Gemini, with a clipboard fallback everywhere else.

### Build and load

```bash
cd extension/
npm install
npm run build          # writes extension/dist/
```

In Chrome: open `chrome://extensions`, enable **Developer mode**, click **Load unpacked**, and select `extension/dist/`.

### Connect your vault

Click the PickMem toolbar icon. First run shows **"Choose vault folder…"** — click it and grant access to the same folder you `init`-ed (via the browser's File System Access API). Chrome remembers the grant across sessions.

Once connected, the popup shows the vault's folder name with a **switch** button to point at a different vault, and an **inbox N** badge in the header whenever captures are waiting (accept them with `pickmem review`).

### Pick and deliver

The popup shows your groups and notes as the same **tree with group checkboxes** as the TUI (click a group to select its whole subtree; tri-state `[ ]`/`[~]`/`[x]`). Saved lenses appear as chips at the top; the bottom field saves the current selection as a new lens.

- **Insert** — on ChatGPT / Claude.ai / Gemini, prepends the assembled block into the chat's input box (your existing draft is preserved). The header shows the detected site and whether the input was found.
- **Copy** — puts the assembled block on your clipboard. Works on **any** site, and even with no vault connected.

### Add a memory from the popup

Click **+ Add memory** to open a small form: a **label** (short title), a **group** (type a new path or pick an existing one from the dropdown), optional **tags**, and the **memory text**. Saving writes a new active note into that group folder — the same result as `pickmem add` — and selects it for you. The new note appears immediately in the tree above.

The extension only ever **creates** notes (and writes `pickmem/active.json` / `pickmem/lenses.json`). It never rewrites an existing note — editing stays in Obsidian or the CLI, which hold the safeguard that prevents clobbering a note you changed elsewhere.

If a site changes its markup and the input can't be found, the popup says so and **Copy** still works.

---

## 9. Import a batch of memories

To bring in many items at once — a memory export from another assistant, or your own list — write them to a file and import it. Each item is staged to the inbox as **pending**; nothing goes active until you review.

```bash
pickmem import memories.txt
```

The parser auto-detects the file shape and takes each chunk as one item:

- JSON array of strings: `["memory 1", "memory 2"]`
- JSON array of objects with a `memory`/`text`/`content`/`body` field
- `{"memories": [...]}` wrapper
- a Markdown bullet/numbered list
- blank-line-separated paragraphs

Override detection with `--format json|bullets|paragraphs|auto`. Each item is routed with your vault's keyword rules (`pickmem/config.json`, substring → group, first match wins), and de-duplicated on a content hash against everything already in the vault.

```
Parsed:    47      # chunks the parser recognized
Staged:    47      # staged to the inbox as pending
Routed:    18      # of those, how many got a suggested_group from the rules
Duplicate: 0       # skipped: content already in the vault
```

Import works best when your file is already roughly one memory per line/paragraph. Each chunk is staged as-is, then you clean up in review.

> AI-assisted extraction (splitting messy text into clean, atomic facts) is planned for a future release. This version routes with keyword rules only.

Then review what landed:

```bash
pickmem list --pending
pickmem review
```

### Undoing an import

An import only ever stages to the inbox — nothing is active yet — so undoing one just clears the pending items:

```bash
pickmem inbox clear --source import --yes   # delete only import-staged items
pickmem inbox clear --yes                   # …or everything pending
```

Run it without `--yes` first to see what would be deleted. Only pending items are eligible — your active notes can never be touched by this command.

---

## 10. Review the inbox

```bash
pickmem review
```

Opens a TUI over the pending items. Each row shows its label and its `suggested_group` (from routing, if any).

| Key | Action |
|-----|--------|
| `space` | Select the row at the cursor |
| `a` | Accept the selected rows (or the cursor row) — moves each note into its group folder and flips it to active |
| `A` | Accept every remaining row that has a `suggested_group` |
| `r` | Reject the selected rows (or the cursor row) — deletes the inbox file |
| `g` | Reassign group — an overlay where you type a new group or pick an existing one |
| `/` | Filter |
| `enter` | Apply all decisions and exit |
| `q` / `esc` | Cancel — the inbox is left unchanged |

A row with no `suggested_group` can't be accepted with `a`/`A` until you give it a group with `g` — this prevents silently misfiling. Typical flow: press `A` to sweep everything already routed, use `g` on the stragglers, then `enter`.

Accepted notes are now active and will appear in `pickmem pick`.

---

## 11. Lenses

A **lens** is a named, saved selection — for a recurring task you don't want to re-pick every time. Lenses live in `pickmem/lenses.json`, so they sync with whatever syncs your vault.

**Save a lens:**
- In the CLI picker (`pickmem pick`): select items, press `s`, name it.
- In the extension popup: select items, type a name in the bottom field, save.

**Activate a lens** (replaces the current selection):
- CLI: `pickmem lens use <name>` — scriptable, e.g. `alias workmode='pickmem lens use Work'`.
- In the picker: `pickmem pick` → `l` → choose one.
- Extension: click its chip.
- From inside Claude Desktop: the model calls `use_lens("<name>")`.

**Manage lenses from the CLI:** `pickmem lens list` (the `*` marks the active one) and `pickmem lens rm <name>` (deletes the lens; its notes are untouched).

Ideas: `Job-Hunt`, `Client-Acme`, `Doctor-Visit`, `Gift-Sister` — anything that repeatedly pulls the same slice of your memory.

---

## 12. Editing and organizing in Obsidian

Every memory note is a normal Markdown file with a YAML frontmatter block. Open the vault in Obsidian and edit freely — with two things to know:

**Create-only.** PickMem only ever creates notes and moves inbox notes into group folders. It never rewrites a note you authored; before updating any file it owns, it checks that the on-disk content still matches what it last wrote, and refuses if you've changed it since. So your Obsidian edits are safe.

**Ids are assigned by PickMem.** Each note needs a stable `id` in its frontmatter, and `pickmem add`, `import`, and the extension's **+ Add memory** form all generate it for you. Because of the create-only rule, PickMem won't backfill an id into a file you hand-create. Practical consequences:

- A note with **no `---` frontmatter block** is ignored by PickMem (a normal Obsidian note — fine, just not a memory item).
- A note with a **complete, valid frontmatter** (including a real id) is picked up — but generating a valid id by hand is impractical.
- A note with a **frontmatter block missing its `id`** will cause vault loads to error until you fix or delete it. So: create memory items with `pickmem add`, not by typing a half-frontmatter file in Obsidian.

**Regrouping** is just changing a note's `group:` field — the frontmatter `group` is what PickMem reads, not the folder the file sits in. You can move files around in Obsidian too; keep the `group:` field in sync if you want the picker's tree to match.

**Your taxonomy is your folder tree.** You aren't limited to the starter groups — **make a folder in Obsidian and it becomes a category.** Every directory under the vault (except `pickmem/`) is a group PickMem knows about: it's what `list_groups` reports to Claude for classification, and what the review overlay suggests when you reassign. Create `work/client-acme/` or `hobbies/sailing/` and Claude can file new memories straight into them — even before any note lives there.

**Keeping a category private.** Because folder names go to the model via `list_groups`, prefix a folder with `_` to keep it *out* of the shared list: `_medical/`, `_finance/`. You can still file notes there by hand in Obsidian, but the name never leaves your disk and Claude never sees it. (Dot-folders like `.obsidian` are excluded too.)

**Don't hand-edit** files under `pickmem/` (`config.json`, `lenses.json`, `active.json`, `inbox/`) — let the tools manage those.

---

## 13. Command reference

The vault path for every command except `init` resolves as `--vault` → `$PICKMEM_VAULT` → recorded user config.

```
pickmem init <path> [--bare] [--force]         # create a vault; applies the starter taxonomy unless --bare

pickmem add --label "…" [--group …] [--type fact|idea|thought|reference] [--tags a,b]
            [--body "…" | --file <path|-> | (stdin) | ($EDITOR)] [--inbox]
pickmem list [--group <prefix>] [--type <kind>] [--pending] [--all]     # alias: ls
pickmem show <id-or-suffix> [--raw]
pickmem edit <id-or-suffix>                    # opens $EDITOR
pickmem rm <id-or-suffix> --yes

pickmem pick                                   # TUI picker → writes active.json
pickmem status                                 # vault summary + current selection
pickmem context [--copy]                       # print (or copy) the assembled block
pickmem lens list | use <name> | rm <name>     # manage saved lenses
pickmem inbox clear [--source import|extract|manual] --yes   # bulk-delete pending items

pickmem serve                                  # MCP stdio server (clients launch this)
pickmem install <claude-desktop|cursor> [--dry-run] [--name <n>] [--bin <path>]
pickmem uninstall <claude-desktop|cursor> [--name <n>]

pickmem import <file> [--format auto|json|bullets|paragraphs]
              # stages parsed items to the inbox as pending
pickmem review                                 # TUI to accept/reject/reassign inbox items

--vault <path>                                 # global override on any command
```

---

## 14. Troubleshooting

**"no vault path set"** — you haven't `init`-ed, or you're running a command before init. Run `pickmem init <path>`, set `$PICKMEM_VAULT`, or pass `--vault`.

**"warning: skipped <file>: …"** — a `.md` file in the vault has a `---` block with a missing or malformed `id` (usually a half-typed note in Obsidian), or a duplicate id. The file is skipped, everything else works; fix or delete it to clear the warning. (Files with no frontmatter at all are ignored silently — they're your normal notes.)

**"refusing to overwrite … create-only"** — you edited a file (probably in Obsidian) between PickMem reading it and trying to write it. That's the guard protecting your edit; re-run the command and it'll re-read your version.

**Claude Desktop doesn't call PickMem tools** — confirm the connector is listed and connected (Settings → Connectors), set the four tools to *Always allow*, and add the custom instruction from §7. Fully quit and relaunch the app after `pickmem install`, not just close the window.

**`pickmem import` staged 0 items** — the parser didn't recognize the file shape. Try an explicit `--format bullets` or `--format paragraphs`, and check the file isn't empty.

**Extension shows "input not found"** — the target site changed its markup. Insert is disabled but **Copy** still works; the adapter selector is a one-line fix in `extension/src/adapters/index.ts`.

**Pointing the extension at a different vault** — click **switch** next to the vault name at the top of the popup and pick the new folder.

---

*PickMem in one line: the model knows exactly what you chose to tell it, this time.*
