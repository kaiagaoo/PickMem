# PickMem — Execution Guidance for Claude Code

You are building **PickMem**, an open-source, local-first memory-curation tool for LLMs. Read `PROPOSAL.md` in this repo first — it is the source of truth for *why* and *what*. This document is the *how*: the build plan, the locked decisions, and the shared data contract. Build the **full** system (all milestones), not a minimal MVP.

One-line thesis: **instead of an assistant auto-injecting remembered context, the user deliberately picks which memory items the model sees per session.** The store is an Obsidian vault. The user picks a slice; PickMem delivers only that slice to the model.

---

## 0. How to work

- Work **milestone by milestone** (M1 → M6 below), in order — later milestones depend on earlier ones. Each milestone is independently shippable.
- **At the start of each milestone**, propose a short plan + file list and ask me about anything ambiguous. Then implement in small, logical commits.
- Do **not** re-litigate the locked decisions in §1. If you think one is wrong, raise it in one sentence and wait — don't silently substitute.
- Keep the **vault data contract (§3) stable** — two codebases (Go + the extension) depend on it. Any change to it must be called out explicitly.
- Prefer boring, idiomatic code. `gofmt`/`vet` clean. Tests for all non-UI logic.

## 1. Locked decisions (do not substitute)

- **Native side = Go**, a single binary `pickmem` with subcommands (use `cobra`). Covers: vault store, TUI picker, MCP server, CLI, ingestion, import routing.
- **Extension = TypeScript**, Chrome Manifest V3.
- **No database.** The Obsidian vault *is* the store. Do not add SQLite/Postgres. (A rebuildable in-memory index over the vault is fine; a second source of truth is not.)
- **TUI = Bubble Tea** (`charmbracelet/bubbletea`) + `bubbles` (`list` with a custom delegate) + `lipgloss`.
- **MCP = the official Go MCP SDK** (`github.com/modelcontextprotocol/go-sdk`) — verify it's current and maintained before adding; if a clearly more standard choice exists, flag it and wait.
- **Markdown + frontmatter** via a maintained Go lib (e.g. `github.com/adrg/frontmatter` + a YAML lib) — verify current before adding.
- **AI is optional and off by default.** Rules-based logic is the default everywhere; any LLM call (import classification, auto-extraction) sits behind an interface and an explicit `--allow-ai` consent flag. Local-first: no network unless the user opts in.
- **Create-only file discipline** (see §4). This is non-negotiable — it's what avoids vault corruption.

## 2. Repo layout

```
pickmem/
  cmd/pickmem/            # main, cobra wiring
  internal/
    vault/               # THE Store: notes, groups, inbox, lenses, active selection
    picker/              # Bubble Tea TUI (pick + inbox review)
    mcp/                 # MCP server exposing the picked slice
    ingest/              # manual add, import parsers, extraction staging
    routing/             # import group-router (rules + optional AI classifier)
    classifier/          # pluggable classifier interface (rules default, AI optional)
  docs/casestudy/        # committed vault fixtures + prompts for the M6 case study
  extension/             # TypeScript MV3 Chrome extension (M5)
  templates/             # starter taxonomy templates
  demo/                  # VHS tapes
  PROPOSAL.md
  EXECUTION.md
  README.md
  LICENSE                # MIT
```

## 3. The vault data contract (shared — treat as an API)

Everything the Go binary and the extension exchange happens through files in the vault. Keep these formats stable and documented.

**Memory note** — one file per item, at `<group-path>/<slug>.md` (groups may nest, e.g. `work/Client-Acme/`):

```markdown
---
id: 01JAX...            # ULID, stable, never reused
label: income — freelance + salary   # short text shown in the picker
group: financial        # AUTHORITATIVE grouping (folder is derived, not truth)
tags: [money, recurring]
source: manual          # manual | import | extract
status: active          # active | pending
created_at: 2026-07-01T12:00:00Z
---

Full memory content goes in the body. Everything below frontmatter
is the content delivered to the model when this item is picked.
```

- `group` in frontmatter is the source of truth. If the folder and `group` disagree, `group` wins. This lets users reorganize folders or use tags freely.
- `label` is what the picker lists; `body` is what gets delivered.

**Inbox (staging)** — `pickmem/inbox/<slug>.md`, same frontmatter with `status: pending` plus `suggested_group: <name>` (for imports/extraction). Reviewable in PickMem's UI *or* directly in Obsidian.

**Lenses** — `pickmem/lenses.json`:
```json
[{ "name": "Job-Hunt", "item_ids": ["01JAX...", "01JBY..."] }]
```

**Active selection** — `pickmem/active.json`:
```json
{ "active_lens": "Job-Hunt", "item_ids": ["01JAX...", "01JBY..."] }
```

**Config** — `pickmem/config.json`: routing rules (keyword→group), injection prefs, classifier settings.

The MCP server and the extension both: read `active.json` → resolve `item_ids` to note bodies → deliver. This resolution is a **deterministic lookup by id**, never a similarity search.

## 4. Core invariants (enforce everywhere)

1. **Create-only.** PickMem creates new files and moves files (inbox → group folder). It edits **only** files it created and that the user hasn't modified since. It never edits user-authored notes. This is what makes ingestion conflict-safe without file locking.
2. **Frontmatter is grouping truth** (see §3).
3. **Deterministic lookup, not RAG.** Picked items are fetched by id. RAG/embeddings appear only as an optional *suggestion* helper for pre-selecting or classifying — never as the retrieval path.
4. **Default to nothing.** Active selection starts empty. Memory is opt-in per session.
5. **The user decides relevance.** No silent auto-injection. Auto-extraction only ever *proposes* (into the inbox); nothing goes active without a human accept.
6. **Local-first.** No network by default. AI features are opt-in behind `--allow-ai`.

---

## Milestones

### M1 — Vault Store + CLI foundation
**Goal:** a working `vault` package that reads/writes the data contract, plus inspection CLI.
- `internal/vault`: load all notes (parse frontmatter), group them, CRUD by id, manage `inbox/`, `lenses.json`, `active.json`, `config.json`. In-memory index built on load.
- CLI: `pickmem init` (scaffold vault structure + copy a taxonomy template + write config), `pickmem add` (create a note), `pickmem list`, `pickmem show <id>`, `pickmem edit <id>`, `pickmem rm <id>`.
- Ship 3–4 taxonomy templates in `templates/` (e.g. "personal", "developer", "multi-client freelancer").
- **DoD:** point it at a folder, `init`, `add` a few items, `list`/`show` them; unit tests for parse/serialize round-trip, group resolution (folder vs frontmatter conflict → frontmatter wins), and id stability.

### M2 — TUI picker (`pickmem pick`)
**Goal:** the full-screen grouped multi-select with a lens overlay. Feel target: "fzf with groups and memory."
- Bubble Tea + `bubbles/list` with a custom delegate rendering: group headers (non-selectable), indented items, a checkbox glyph.
- **Three distinct row states:** unselected (dim), cursor (highlighted bar), selected (accent + filled checkbox).
- Keys: `↑/↓` `j/k` move · `space` toggle · `/` fuzzy filter (label+body+tags) · `l` lens overlay (`gum choose`-style; selecting replaces selection + sets active lens) · `s` save current selection as a new lens (inline name prompt) · `enter` confirm → write `active.json`, print one-line summary, exit 0 · `q`/`esc` cancel.
- Footer: `Active: <lens|custom> · N selected · ~T tokens` (token ≈ ceil(chars/4) over selected bodies) + dim key hints.
- One theme file (Lip Gloss), Nord/Catppuccin-ish, swappable; respect `NO_COLOR`.
- Must work at 80×24 and on resize.
- **DoD:** usable against an M1 vault; unit tests for toggle, lens-apply, token estimate, active-selection persistence; a `demo/pick.tape` VHS script producing a ~10s GIF (open → filter → toggle → lens overlay → pick → enter).

### M3 — MCP server (`pickmem serve`)
**Goal:** expose *only the picked slice* to native clients.
- stdio MCP server. Expose the active selection as: a **resource** (`pickmem://active` returning the assembled context block) and **tools**: `get_active_memory`, `list_lenses`, `use_lens(name)`, and `propose_memories(chat_text)` (auto-extraction → writes candidates to `inbox/` as `pending`; returns a summary, does NOT activate).
- Assembled block = concatenated bodies of picked items, with light provenance headers. Deterministic id lookup.
- Provide `pickmem install <client>` helpers that write the MCP server entry into Claude Desktop / Cursor / Cline configs.
- **DoD:** connect from Claude Desktop; the model sees only picked items; `use_lens` switches context; `propose_memories` stages to inbox without activating. Document the client config in README.

### M4 — Ingestion + inbox review + import routing
**Goal:** the three-path capture lifecycle, all create-only, all staged.
- `pickmem import <file>`: parse ChatGPT/Claude memory exports (and a generic list format) → one `pending` note per item in `inbox/`, de-duped on content hash.
- **Import router** (`internal/routing`): assign each item a `suggested_group` — rules first (keyword→group from config), then optional AI classifier (`--allow-ai`) proposing from the *existing* taxonomy for unmatched items.
- **Inbox review** (TUI screen, reuse picker chrome): list pending items with `suggested_group` inline; **bulk** multi-select → reassign group → accept; accept-all-remaining. Accept = move file inbox→group folder, flip `status: active`. Reject = archive/delete.
- `classifier` interface: `RulesClassifier` default, `AIClassifier` optional (Anthropic API or local endpoint) behind consent.
- **DoD:** import a 30+ item export → routed with suggestions → bulk-review → accepted into group folders as active; tests for parser, dedupe, rules router, and the create-only/accept file moves.

### M5 — Chrome extension (web models, all models)
**Goal:** a self-contained control panel + injector so any website chat works. Parallel to the TUI, not downstream of it.
- MV3 TypeScript extension. **Vault connection:** File System Access API (`showDirectoryPicker`) granted once; persist the directory handle in IndexedDB; re-request on expiry. Read notes, `lenses.json`, `active.json`, `config.json` from the vault (same contract as §3).
- **Picker panel** (popup): show groups + lenses; toggle items / activate a lens; writes `active.json` back to the vault.
- **Injection** via per-site **adapters** (each: URL match + input-box selector + insert method). Ship adapters for **ChatGPT, Claude.ai, Gemini** first. **On-demand** by default (user clicks "insert" → assembled block goes into the input, prepended), with an optional per-site auto toggle.
- **Clipboard fallback** (`copy`) that works on every site with zero adapter code.
- **Distribution: unpacked / load-your-own, NOT the Chrome Web Store.** Ship the extension as source in `extension/` with a documented three-step "Load unpacked" install (enable Developer Mode → Load unpacked → grant the vault folder). Do not build for or assume a Web Store submission. Chrome Web Store is deferred to Phase 4 (adoption), to be reconsidered once the permission surface is stable and aimed at non-developer users.
- **Permission note:** the File System Access grant is the review-sensitive permission (local file access + content scripts on ChatGPT/Gemini/Claude.ai is exactly what store review scrutinizes). Keep permissions minimal and justified in `extension/README.md`. Treat the **clipboard-only mode** as the low-permission fallback path — it must work fully even if a user never grants file access — so there's a viable escape hatch if store review (later) proves difficult.
- **DoD:** installs via Load unpacked; grant vault access, pick a lens, insert into ChatGPT/Claude.ai/Gemini; clipboard fallback works anywhere (including with no file access granted); graceful message when an adapter's selector breaks. Load-unpacked install steps + permission justification + manual test checklist in `extension/README.md`.

### M6 — Case study + polish + launch assets
**Goal:** the credibility artifact and a launchable repo. The thesis (curated context beats dumped/auto-injected context) is qualitative, so prove it with a **reproducible case study**, not a synthetic benchmark.
- **Case study** (`docs/case-study.md` + `docs/casestudy/` fixtures): the primary proof artifact. Requirements:
  - **4–6 scenarios** spanning the use cases: a technical question where you want neutrality, a shopping recommendation, a multi-client work task, a brainstorm (add others as useful). Breadth is what makes it a study, not a cherry-picked demo.
  - **Three conditions per scenario, same prompt**, shown side by side: full-dump (all memory) vs. auto-memory-style (naive top-k injection) vs. **PickMem curated**. The contrast is the evidence.
  - **Include at least one honest failure/limitation case** — a scenario where curation doesn't help, or where picking was friction. A study that only wins reads as a testimonial; showing the boundary is what makes it credible.
  - **Reproducible:** commit the exact vault fixtures and prompts under `docs/casestudy/` so a reader can rerun it. This reproducibility replaces the numeric harness as the rigor signal — it must not be hand-waved.
  - Summarize findings in plain prose (what changed, why it mattered) and link it from the README. No fabricated metrics; if you quantify anything, it must come from the committed, rerunnable fixtures.
- Docs: per-component READMEs, integration guides, `demo/` GIFs for pick + inbox + extension injection.
- Repo hygiene: good-first-issues, CONTRIBUTING, semantic-versioned releases, MIT license header where appropriate.
- **Deferred (do not do now):** Chrome Web Store submission for the extension. Revisit only after the permission surface is stable and you're targeting non-developer users; it requires a $5 registration, review, and a privacy justification for the File System Access + AI-site host permissions. Until then the extension stays load-unpacked.
- **Note (audience-dependent, optional):** if later targeting research / applied-scientist roles, a small honest quantitative benchmark can be added on top — but it is not required, and it must not replace the case study or invent numbers.
- **DoD:** case study written with committed rerunnable fixtures and a failure case; GIFs recorded; docs complete.

---

## 5. Quality bar

- Go: `gofmt`/`vet` clean, no dead code, unit tests for every non-UI logic path (vault CRUD, group resolution, lens ops, active-selection round-trip, import parse/dedupe, router, accept file-moves).
- The create-only invariant must be covered by a test that proves PickMem never rewrites a user-authored note.
- TUI + extension: manual test checklists in their READMEs; VHS tapes for the TUI.
- Everything runs offline by default; AI paths are gated and tested with the gate off.

## 6. Out of scope / non-goals

- No cloud service, account, or telemetry. No second datastore. No two-way "sync engine" — there's one store (the vault), so there's nothing to sync.
- PickMem does not auto-inject, does not decide relevance for the user, and does not edit user-authored notes.
- Not a notes app, not an auto-organizer (that's claude-obsidian's job — PickMem is complementary), not a "remember everything" agent.

## 7. First action

Confirm the current status of the Go MCP SDK and frontmatter libraries, propose the M1 plan + file list, ask any open questions, then start M1. Do not begin M2 until M1's DoD is met.
