# CLAUDE.md

Context for future Claude Code sessions working in this repo.

## Source of truth

- **`PROPOSAL.md`** — the *why/what*. Read before scope decisions.
- **`EXECUTION.md`** — the *how*, milestones, locked decisions, vault data contract. Do not re-litigate locked items in §1; the vault data contract in §3 is a shared API (Go binary + Chrome extension both depend on it) — if you change it, call it out.
- **This file** — build status + how to test what's landed.

## Build status

| Milestone | Status | Notes |
|-----------|--------|-------|
| M1 — Vault Store + CLI | ✅ Done | `init`, `add`, `list`, `show`, `edit`, `rm`; 3 templates; vault package + tests. |
| M2 — TUI picker | ✅ Done | `pickmem pick` — grouped multi-select, lens overlay, fuzzy filter, save-as-lens, Nord/plain themes. |
| M3 — MCP server | ✅ Done | `pickmem serve` (stdio) exposing `pickmem://active` + 4 tools; `install`/`uninstall` for Claude Desktop and Cursor. |
| M4 — Ingestion + inbox review | ⬜ Next | Import parsers, routing, bulk-review TUI. |
| M5 — Chrome extension | ⬜ | MV3, load-unpacked distribution. |
| M6 — Case study + polish | ⬜ | 4–6 scenarios, 3 conditions each. |

Work milestone by milestone, in order. At the start of each, propose a plan + file list, then implement.

## Repo layout

```
cmd/pickmem/          # main (cobra entry point)
internal/
  vault/              # THE Store: notes, groups, inbox, lenses, active. All CRUD goes through here.
  picker/             # Bubble Tea TUI (Model/Update/View, filter, lens overlay, theme)
  mcp/                # MCP server: assemble.go (active → block), propose.go (chat → inbox), server.go (SDK wiring)
  install/            # Client config writers (Claude Desktop, Cursor) — merges, doesn't clobber
  cli/                # cobra subcommands + vault-path discovery
templates/            # personal, developer, researcher (embedded via go:embed)
demo/                 # VHS tapes (pick.tape → pick.gif)
```

Module: `github.com/qwgao/pickmem`. Go 1.26.

## Vault path discovery

Every subcommand except `init` resolves the vault path in this order:
1. `--vault <path>` flag
2. `$PICKMEM_VAULT` env var
3. `~/.config/pickmem/config.json` (or `$XDG_CONFIG_HOME/pickmem/config.json`) — recorded by `init`

## How to test M3 (`pickmem serve` + `install`)

The server is stdio-only, so most useful testing goes through a real client. But you can drive it by hand for a sanity check:

```bash
go build -o /tmp/pickmem ./cmd/pickmem

# Fresh vault, add a note, put it into the active selection.
VAULT=$(mktemp -d) && /tmp/pickmem init "$VAULT"
/tmp/pickmem add --label "salary" --group financial --body "monthly base \$8k" --vault "$VAULT"
# Grab the ULID from that add output, then:
ID=<paste the ULID>
echo "{\"item_ids\":[\"$ID\"]}" > "$VAULT/pickmem/active.json"

# Send a stdio round-trip: initialize + tools/list + read the resource.
{
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"cli","version":"1"}}}'
  echo '{"jsonrpc":"2.0","method":"notifications/initialized"}'
  echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
  echo '{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"pickmem://active"}}'
  sleep 0.2
} | /tmp/pickmem serve --vault "$VAULT"
```

Wire it into Claude Desktop (macOS):

```bash
/tmp/pickmem install claude-desktop            # writes ~/Library/Application Support/Claude/claude_desktop_config.json
/tmp/pickmem install claude-desktop --dry-run  # preview the entry without writing
/tmp/pickmem uninstall claude-desktop          # remove
```

For Cursor: `/tmp/pickmem install cursor` (writes `~/.cursor/mcp.json`). Both merges are non-destructive — other MCP servers already in the config are preserved.

**Cline** doesn't have a stable per-user config path (its state lives inside VS Code workspaces), so add it by hand from Cline's Settings → MCP Servers UI. Command: the `pickmem` binary; args: `serve`.

**Tools exposed:**

| Tool / Resource | Purpose |
|-----------------|---------|
| resource `pickmem://active` | The assembled context block for the current pick |
| tool `get_active_memory` | Same block via a tool call (for clients that don't auto-wire resources) |
| tool `list_lenses` | `[{name, items}]` for every saved lens |
| tool `use_lens(name)` | Activate a lens → rewrites `active.json` → returns the new block |
| tool `propose_memories(chat_text)` | Rules-based extraction → stages to `pickmem/inbox/` as `status: pending`. **Never activates.** Only the user (via `pick`) can promote them. |

The invariant to keep intact: `propose_memories` writes to inbox only. If you touch the propose path, re-verify `TestProposeMemoriesStagesToInboxOnly` in `internal/mcp/server_test.go`.

Automated tests:
```bash
go test ./internal/mcp/...       # 9 tests: resource, all 4 tools, dedupe, routing rules
go test ./internal/install/...   # 6 tests: merge, replace, uninstall preserves siblings
```

## How to test M2 (`pickmem pick`)

Run the picker against the vault you built in the M1 walkthrough (or a fresh one):

```bash
go build -o /tmp/pickmem ./cmd/pickmem
/tmp/pickmem pick
```

Keys inside the picker:

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | move (skips group headers) |
| `space` | toggle item at cursor |
| `/` | filter mode — types match label + body + tags (fuzzy) |
| `esc` (in filter) | clear + exit filter |
| `enter` (in filter) | keep filter, back to browse |
| `l` | open lens overlay (opens only if lenses exist) |
| `s` | save current selection as a new lens (name prompt) |
| `enter` (browse) | confirm — writes `pickmem/active.json` and exits |
| `q`/`esc` (browse) | cancel — active.json unchanged |

Confirming with **no items** selected is intentional: it clears `active.json` (matches the "default is nothing" invariant). If you want to avoid that, `q` to cancel instead.

Verify the round-trip:
```bash
/tmp/pickmem pick                       # select a few, save as lens "Weekend", enter
cat "$VAULT/pickmem/active.json"        # should list the selected ids
cat "$VAULT/pickmem/lenses.json"        # should include Weekend
/tmp/pickmem pick                       # re-open — same selection + activeLens preloaded
```

Non-interactive checks:
```bash
go test ./internal/picker/...           # 12 tests: toggle, filter, tokens, lens apply, persistence
NO_COLOR=1 /tmp/pickmem pick --help     # help text renders without escape codes
```

### VHS demo

`demo/pick.tape` records a ~10s walkthrough. Requires `vhs` (`brew install charmbracelet/tap/vhs`) and `pickmem` on `PATH`:

```bash
go install ./cmd/pickmem
vhs demo/pick.tape                      # writes demo/pick.gif
```

## How to test M1

```bash
# build
go build -o /tmp/pickmem ./cmd/pickmem

# fresh vault (use a tmp dir; init records it as your default)
VAULT=$(mktemp -d)
/tmp/pickmem init "$VAULT" --template personal

# subsequent commands use the recorded vault
/tmp/pickmem add --label "salary" --group financial --tags money,recurring --body "monthly base \$8k"
echo "loves plants" | /tmp/pickmem add --label "sister gift ideas" --group relationships
/tmp/pickmem add --label "kickoff notes" --group "work/Client-Acme" --body "kickoff Aug 1"
/tmp/pickmem add --label "solar research" --group home --inbox --body "brainstorm"

/tmp/pickmem list                    # active only, grouped
/tmp/pickmem list --all              # includes pending inbox
/tmp/pickmem show <short-suffix>     # e.g. show 9TW0KK — 3+ chars enough
/tmp/pickmem show <id> --raw         # print raw frontmatter+body

# edit launches $EDITOR; PickMem never rewrites bytes itself
EDITOR=vi /tmp/pickmem edit <id>

/tmp/pickmem rm <id>                 # needs --yes to confirm
/tmp/pickmem rm <id> --yes
```

### Automated tests

```bash
go test ./...            # ~14 tests, ~1s
go vet ./...
gofmt -l .               # should print nothing
```

The load-bearing test is `TestCreateOnlyNeverRewritesUserAuthoredFile` in `internal/vault/vault_test.go` — it plants a user-authored file with a colliding slug and verifies its bytes are untouched after every store operation. That test defends invariant #1 of §4 in EXECUTION.md. Don't skip or weaken it.

### On-disk layout after a fresh init + a few adds

```
<vault>/
├── financial/salary.md              # frontmatter + body
├── relationships/sister-gift-ideas.md
├── work/Client-Acme/kickoff-notes.md
├── health/ home/ personal/          # empty group folders (.gitkeep)
└── pickmem/
    ├── inbox/                       # pending notes stage here
    │   └── solar-research.md
    ├── config.json                  # routing rules, template name, schema version
    ├── lenses.json                  # []
    └── active.json                  # { "active_lens": "", "item_ids": [] }
```

## Conventions worth remembering

- **Create-only.** PickMem only creates files and moves inbox→group. `Store.Update` checks the on-disk sha256 against the last-written hash before rewriting; if a user edited via Obsidian, Update refuses.
- **Frontmatter is grouping truth.** Folder location is derived, never authoritative.
- **Deterministic id lookup, not RAG.** Picking = fetch by id. No similarity search anywhere.
- **AI features are gated behind `--allow-ai`.** M1 has none; M4 will introduce classifiers behind that flag.
- **`pickmem edit` launches `$EDITOR`** (or `$VISUAL`, or `vi`). PickMem itself does not rewrite user-facing files.

## Key libraries

- `github.com/spf13/cobra` — CLI
- `github.com/adrg/frontmatter` — YAML frontmatter reader
- `gopkg.in/yaml.v3` — frontmatter writer (`adrg` is read-only)
- `github.com/oklog/ulid/v2` — note ids
- `golang.org/x/term` — TTY detection for stdin-vs-editor
- `github.com/modelcontextprotocol/go-sdk` — MCP server (v1.6.1, official)
- `github.com/charmbracelet/bubbletea` + `bubbles` + `lipgloss` — TUI picker
- `github.com/sahilm/fuzzy` — picker filter

## Before starting a new milestone

1. Re-read the milestone spec in `EXECUTION.md` (M2 = §M2, etc.).
2. Propose a plan + file list, ask any open questions.
3. Do not re-open §1 locked decisions.
4. Keep the vault data contract (§3) stable.
