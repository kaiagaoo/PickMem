# CLAUDE.md

Context for future Claude Code sessions working in this repo.

## Source of truth

- **`PROPOSAL.md`** — the *why/what*. Read before scope decisions.
- **`EXECUTION.md`** — the *how*, milestones, locked decisions, vault data contract. Do not re-litigate locked items in §1; the vault data contract in §3 is a shared API (Go binary + Chrome extension both depend on it) — if you change it, call it out.
- **This file** — build status + how to test what's landed.

## Build status

| Milestone | Status | Notes |
|-----------|--------|-------|
| M1 — Vault Store + CLI | ✅ Done | `init`, `add`, `list`, `show`, `edit`, `rm`; single starter template auto-applied (`--bare` to skip); vault package + tests. |
| M2 — TUI picker | ✅ Done | `pickmem pick` — grouped multi-select, lens overlay, fuzzy filter, save-as-lens, Nord/plain themes. |
| M3 — MCP server | ✅ Done | `pickmem serve` (stdio) exposing `pickmem://active` + 4 tools; `install`/`uninstall` for Claude Desktop and Cursor. |
| M4 — Ingestion + inbox review | ✅ Done | `pickmem import <file>` (JSON/bullets/paragraphs auto-detect); `pickmem review` (bulk-select TUI); rules + optional Anthropic AI assistant (split into atomic claims, propose new groups, suggest merges into existing notes) behind `--allow-ai` or an interactive prompt. |
| M5 — Chrome extension | ✅ Done | MV3 TypeScript, load-unpacked. Adapters for ChatGPT/Claude.ai/Gemini + clipboard fallback. Byte-parity assemble with the MCP block. |
| M6 — Case study + polish | ⬜ Next | 4–6 scenarios, 3 conditions each. |

Work milestone by milestone, in order. At the start of each, propose a plan + file list, then implement.

## Repo layout

```
cmd/pickmem/          # main (cobra entry point)
internal/
  vault/              # THE Store: notes, groups, inbox, lenses, active. All CRUD goes through here.
  picker/             # Bubble Tea TUI (Model/Update/View, filter, lens overlay, theme)
  mcp/                # MCP server: assemble.go (active → block), propose.go (chat → inbox), server.go (SDK wiring)
  install/            # Client config writers (Claude Desktop, Cursor) — merges, doesn't clobber
  ingest/             # text.go (shared split/hash/label), parse.go (JSON/bullets/paragraphs), import.go (pipeline)
  routing/            # Router + Classifier interface, RulesClassifier, AIClassifier (Anthropic Messages API)
  cli/                # cobra subcommands + vault-path discovery
templates/            # single "starter" taxonomy (embedded via go:embed); auto-applied by init unless --bare
demo/                 # VHS tapes (pick.tape → pick.gif)
extension/            # M5: MV3 TypeScript Chrome extension
  src/
    manifest.json     # MV3 manifest (minimal permissions, 3 host permissions)
    popup/            # picker panel (HTML/CSS/TS)
    content/          # runs on adapter-matched pages, receives inject msgs
    background.ts     # minimal service worker
    adapters/         # declarative registry (ChatGPT, Claude.ai, Gemini)
    vault/            # handle persistence (IDB), reader, writer, assemble (byte-parity with Go)
    lib/              # frontmatter parser, clipboard, message types
  test/               # node --test (frontmatter, assemble, adapters)
  esbuild.config.mjs  # bundles 3 entrypoints → dist/
```

Module: `github.com/qwgao/pickmem`. Go 1.26.

## Vault path discovery

Every subcommand except `init` resolves the vault path in this order:
1. `--vault <path>` flag
2. `$PICKMEM_VAULT` env var
3. `~/.config/pickmem/config.json` (or `$XDG_CONFIG_HOME/pickmem/config.json`) — recorded by `init`

## How to test M5 (Chrome extension)

Build + load-unpacked:
```bash
cd extension/
npm install
npm run build           # → extension/dist/
# Chrome → chrome://extensions → Developer mode → Load unpacked → extension/dist/
```

Run tests + typecheck (no browser needed):
```bash
cd extension/
npm test                # 19 tests: frontmatter parser, assemble byte-parity, adapter registry
npm run typecheck       # tsc --noEmit
```

Full manual test checklist lives in [extension/README.md](extension/README.md#manual-test-checklist) — run it before any release.

**Design constraint to remember:** the extension's `src/vault/assemble.ts` must produce output byte-identical to `internal/mcp/assemble.go`. If you change the block format on either side, change both — and update the two test suites that lock the format (`assemble.test.ts` and `internal/mcp/assemble_test.go`). Users switch between the MCP path (Claude Desktop) and the extension path (browser); the same selection must produce the same context or the "same brain, two channels" thesis breaks.

**Block format:** plain markdown, deliberately boring — `--- pickmem: selected memory ---`, one `label (group): body` line per item, `--- end pickmem memory ---` footer. Single blank line between items, nowhere else. An earlier version wrapped items in `<pickmem_memory><item>` XML tags on the theory that models follow tag boundaries better; reverted because that's Claude-specific prompt-engineering guidance with no evidence it holds for the extension's other two targets (ChatGPT, Gemini), and there was no A/B data behind the claim either way — see conversation history if resurrecting this. The closing `--- end pickmem memory ---` line doubles as the boundary against whatever the user types next when Insert/Copy glue the block into a chat input, so no separate divider is needed on the extension side.

**Extension write scope:** the popup writes only `pickmem/lenses.json` and `pickmem/active.json`. It never creates or edits memory notes. That's a hard boundary — the create-only invariant lives in Go's `Store.Update` (sha256 check against last-written bytes) and can't be enforced from the browser, so we keep the extension's writes strictly to metadata files where clobbering is a non-issue.

**Adapters:**
- Registry: [extension/src/adapters/index.ts](extension/src/adapters/index.ts) — one entry per site.
- Adding a site is a single declarative entry (URL regex + input selector + insert kind). No per-site code paths.
- When a selector breaks, the popup shows a specific error and clipboard fallback still works — never silently fail.

**Distribution note:** load-unpacked only. Chrome Web Store submission is deferred (§Phase 4 in EXECUTION.md) — do not add store-submission artifacts without discussion.

## How to test M4 (`pickmem import` + `pickmem review`)

```bash
go build -o /tmp/pickmem ./cmd/pickmem
VAULT=$(mktemp -d) && /tmp/pickmem init "$VAULT"   # starter taxonomy auto-applied

# Point it at any of these shapes — auto-detect picks the right parser:
#   ["memory 1", "memory 2"]                       bare JSON array
#   [{"memory": "..."}, {"text": "..."}]           JSON objects
#   {"memories": ["...", "..."]}                   wrapped
#   - bullet 1\n- bullet 2                         markdown list
#   paragraph 1.\n\nparagraph 2.                   blank-line separated
echo '["moved to Portland in 2024","prefers vim over vscode","runs python + docker in prod"]' > /tmp/export.json
/tmp/pickmem import /tmp/export.json --vault "$VAULT"
# ->  Parsed: 3   Staged: 3   Routed: 1 (docker → stack)   Duplicate: 0

/tmp/pickmem list --pending --vault "$VAULT"     # see what's queued
/tmp/pickmem review --vault "$VAULT"             # bulk-review TUI
```

Review TUI keys:

| Key | Action |
|-----|--------|
| `space` | select at cursor |
| `a` | accept selected (or cursor row) — moves inbox → group, flips `status: active` |
| `A` | accept every remaining row that has a `suggested_group` (never auto-merges — see below) |
| `m` | accept as a **merge** into the AI-suggested existing note (no-op if the row has no `suggested_merge_id`) |
| `r` | reject selected (or cursor) — deletes inbox file |
| `g` | reassign group (overlay: type new, `tab`/`↓` browse existing) — always overrides a prior merge decision |
| `/` | filter over label + body + suggested_group |
| `enter` | apply decisions and exit |
| `q`/`esc` | cancel — inbox unchanged |

Rows with no suggested_group can't be accepted with `a`/`A` — you have to `g` first. This prevents silent misfiling. `A` deliberately never merges, even on rows with a merge suggestion — merging is a per-row `m`, always explicit.

Row suffixes in the TUI: `→ <group>` (existing-group match), `→ NEW: <group>` (AI proposes a group not in the vault yet — accent-colored so taxonomy drift is visible before you accept it), `→ merge? "<label>"` (AI suggests folding into an existing note).

**AI-assisted import (opt-in):**
```bash
export ANTHROPIC_API_KEY=sk-ant-...
/tmp/pickmem import /tmp/export.json --allow-ai --vault "$VAULT"
# or: leave off --allow-ai and answer the one-time Y/n prompt shown when
# $ANTHROPIC_API_KEY is set and stdin is a real terminal (skipped entirely
# in piped/non-interactive contexts — no surprise network calls in scripts)
```
`internal/routing.AIImportAssistant` (`internal/routing/assist.go`) adds three capabilities beyond the plain rules `Router`, used only by `pickmem import` — never by MCP's `propose_memories`, which stays conservative on purpose:
- **`SplitClaims`** — decomposes one imported item into atomic per-fact claims (e.g. "I moved to Portland and I prefer vim" → two staged notes instead of one). Skipped for short candidates (< 60 chars) to avoid a pointless API call.
- **`ClassifyForImport`** — like the plain `AIClassifier`, but allowed to propose a group that doesn't exist yet (`new: <name>`). A new-group proposal is **staged, not created** — it lands in the inbox exactly like any other suggestion and only becomes a real folder when you `a`-accept it in review. This is a deliberate loosening of the older "AI can only pick from the existing taxonomy" rule — see the git history around this change for the decision record if you're wondering why it's not still absolute.
- **`SuggestMerge`** — for a claim routed to an *existing* group, checks whether it belongs inside one of that group's notes instead of becoming a new one. Scoped to one group's notes at a time (bounded cost, never a vault-wide scan). The merge itself is a **deterministic append** (`Store.MergeInboxInto`, blank-line separated) — never an AI-authored rewrite of an existing note's prose, and it goes through the same create-only hash-check as any other `Store.Update`, so a note you've hand-edited since import refuses the merge rather than clobbering it.

All three fail soft: an assistant error for one claim falls back to the plain rules `Router` for that claim, not an aborted import (same "an outage shouldn't fail an import" posture as the original `AIClassifier`).

**Data contract note:** `vault.Frontmatter` gained one new optional field, `suggested_merge_id` (`internal/vault/note.go`) — backward compatible, `omitempty`. Flagging it here per the shared-contract rule at the top of this file.

Automated tests:
```bash
go test ./internal/ingest/...    # 21 tests: parsers, dedupe, routing, 30-item DoD, assistant split/classify/merge/fail-soft
go test ./internal/routing/...   # 30 tests: rules, Router chain, AIClassifier, AIImportAssistant (mock HTTP)
go test ./internal/picker/...    # 23 tests: picker + review model state machines, incl. merge outcome
go test ./internal/vault/...     # 21 tests: incl. MergeInboxInto + its create-only refusal case
```

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
/tmp/pickmem init "$VAULT"

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
- **AI features are gated behind `--allow-ai` (or an interactive Y/n prompt).** M4 introduced the plain `AIClassifier` (existing-taxonomy-only, used by `propose_memories` and as import's fallback layer) and `AIImportAssistant` (import-only: splits claims, proposes new groups, suggests merges). Both off by default; both require `$ANTHROPIC_API_KEY`. `AIClassifier` still can't invent taxonomy. `AIImportAssistant` *can* propose a new group — but only ever as a **staged suggestion** in the inbox, identical in kind to any other suggestion; it never creates a folder or note directly. "Nothing lands without a tap" still holds for new taxonomy exactly as it does for new notes.
- **`pickmem edit` launches `$EDITOR`** (or `$VISUAL`, or `vi`). PickMem itself does not rewrite user-facing files.

## Key libraries

- `github.com/spf13/cobra` — CLI
- `github.com/adrg/frontmatter` — YAML frontmatter reader
- `gopkg.in/yaml.v3` — frontmatter writer (`adrg` is read-only)
- `github.com/oklog/ulid/v2` — note ids
- `golang.org/x/term` — TTY detection for stdin-vs-editor
- `github.com/modelcontextprotocol/go-sdk` — MCP server (v1.6.1, official)
- `github.com/charmbracelet/bubbletea` + `bubbles` + `lipgloss` — TUI picker + review
- `github.com/sahilm/fuzzy` — picker filter
- Anthropic Messages API — direct `net/http` call, no SDK; behind `--allow-ai`
- **Extension (TypeScript):** esbuild (bundler), no runtime deps. Frontmatter parser hand-rolled (~40 lines) to avoid pulling js-yaml. `@types/chrome` for DOM/Chrome API types only.

## Before starting a new milestone

1. Re-read the milestone spec in `EXECUTION.md` (M2 = §M2, etc.).
2. Propose a plan + file list, ask any open questions.
3. Do not re-open §1 locked decisions.
4. Keep the vault data contract (§3) stable.
