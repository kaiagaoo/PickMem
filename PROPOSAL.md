# PickMem — a memory-curation layer for LLMs

> **PickMem** — *you pick what the model remembers.*
> Local-first. Your brain lives in an Obsidian vault. You choose which slice reaches the model, per session.
> *(Name locked in Phase 0. Confirm the GitHub org, PyPI/npm, and `.dev`/`.com` domain before first publish.)*

---

## 1. Why I'm building this

Every major assistant now has memory, marketed as intimacy: it "remembers you." In practice the remembered context often makes answers *worse*, not better — it imports details irrelevant to the question and bends the model toward agreement instead of usefulness. Two failure modes underneath that feeling:

- **Over-personalization noise** — past-chat context leaks into questions where it doesn't belong.
- **Sycophancy amplification** — the more the model "knows" about you, the more it mirrors you instead of staying a neutral tool.

The root issue is *who holds the controls*. Today the system decides what's relevant and injects it silently; you can only view or delete after the fact. PickMem inverts that: **the default is nothing, and you add context on purpose.**

## 2. The thesis

PickMem is built around one inversion: **the user curates context; the system does not guess it.** Three primitives:

1. **A taxonomy you own** — groups like `financial → income, bills`; `work → Client-X, stack`; `personal`.
2. **A per-session picker** — you select which items the model sees, deliberately, scoped to the task.
3. **Lenses** — saved, named selections for recurring scenarios ("Job-Hunt", "Gift-for-sister", "Client-Acme"). Activate a lens instead of re-picking.

The opposite of "the model remembers everything about you" is "the model knows exactly what you chose to tell it, this time."

## 3. Design axioms

- **Local-first, no exceptions.** Your memory lives on your disk; PickMem never phones home. Convenience is pursued *within* this constraint, not around it.
- **Obsidian is the store (vault-as-DB).** One source of truth — which deletes the hardest problem in the design: two-store sync. Notes are memory items; folders plus frontmatter are the taxonomy; lenses are a file inside the vault.
- **Deterministic lookup, not RAG.** When you pick an item, its content is fetched by id — an exact read, not a similarity guess. (RAG appears only as an optional *suggestion* helper, never as the core mechanism; using it to choose would re-automate the decision PickMem exists to hand back.)
- **Create-only ingestion.** PickMem creates and moves files; it never edits notes you authored. This sidesteps the multi-writer corruption that file-based AI tools otherwise have to engineer around.
- **The user decides relevance.** PickMem will not silently auto-inject memory — that is the exact behavior it replaces.

## 4. Architecture: one brain, two delivery channels

**The brain** is an Obsidian vault. Each note is a memory item (label = short title, body = content). Frontmatter carries `id`, `group`, `tags`, `source`, `status`, `created_at` — and **frontmatter is the source of truth for grouping**, so a user can reorganize folders or use tags and PickMem still knows each item's group. Lenses live in an in-vault file, so they sync with the user's existing Obsidian sync for free.

The picked selection feeds two delivery channels:

- **Channel A — MCP (native clients).** A local MCP server exposes *only the currently-picked notes* to clients that can launch it: Claude Desktop, Claude Code, Cursor, Cline. Structured and out-of-band.
- **Channel B — Chrome extension (web models).** The extension reads the vault and injects the assembled context into the chat box of any website — ChatGPT, Gemini, Claude.ai, Perplexity, and others. Because it's just text entering an input field, it is **model-agnostic by construction** — this is what makes PickMem usable on *all* models, including those with no MCP support. A **clipboard fallback** (`copy → paste`) covers the long tail with zero per-site code.

**Front-ends are parallel, not chained.** Terminal users open the **TUI picker**; web users open the **extension**; native apps read over **MCP**. A web user never touches a terminal. The extension reads the vault via the browser's File System Access API (grant the folder once) in v1; an optional silent companion process can serve the vault over localhost later for live-sync reliability.

## 5. How memory gets in

Three ingestion paths, all reducing to the same safe primitive — *write a new file* — and all flowing through one lifecycle:

- **Manual add** — a form (or you create a note in Obsidian yourself). A new note with `source: manual`.
- **Import** — point PickMem at a ChatGPT/Claude memory export or any list; one new note per item, de-duped on content hash.
- **Auto-extraction** — PickMem reads the current chat, extracts candidate memories, and writes them as *new files* to a `pickmem/inbox/` folder with `status: pending`.

**Everything stages before it lands.** Candidates sit in the in-vault inbox (reviewable in PickMem's UI *or* directly in Obsidian). You **accept / edit / reject**; on accept, the file moves into its group folder and flips to `status: active`. This keeps two guardrails at once: the *thesis* guardrail (nothing enters your active brain without your tap) and the *concurrency* guardrail (only new-file creates and moves, never edits to your notes).

**Imports route into real groups, not a dump.** A graduated router assigns each item a *suggested* group: rules-based keyword mapping handles the obvious majority; AI classification proposes a group (from your existing taxonomy) for the rest; you override anything in a fast **bulk-review** screen (multi-select → reassign → accept-all). Import just pre-fills a suggested `group` in the inbox; review and accept are identical to the other paths.

## 6. How it's used

The full lifecycle: **capture** (manual / import / auto-propose → new files) → **review** (tap-accept out of the inbox) → **curate** (the picker selects active notes into a lens) → **deliver** (MCP or extension exposes only the picked slice). Obsidian is both the store and the management UI; PickMem is the capture-review-and-pick layer on top.

## 7. Supported surfaces

| Surface | Channel | How context arrives |
| --- | --- | --- |
| Claude Desktop, Claude Code, Cursor, Cline | MCP (local server) | Structured, out-of-band |
| ChatGPT, Gemini, Claude.ai, Perplexity, … (web) | Chrome extension | Injected into the chat box |
| Any other chat surface | Clipboard | `copy → paste` |

"All models" via injection is a *maintenance commitment*: ship 2–3 solid site adapters first, plus the clipboard fallback, then grow adapter by adapter.

## 8. How this differs

- **Agent-memory backends** (Mem0, Zep, Letta) and **consumer memory** (ChatGPT/Claude/Gemini) bet on *automatic* retrieval; the system chooses. PickMem hands the choice back to the user.
- **claude-obsidian** (Karpathy's LLM-Wiki pattern) *builds and auto-organizes* an Obsidian brain — retrieval-first, AI writes. PickMem is *curation-first, user chooses*: it adds the per-session **pick** that no one in the Obsidian ecosystem offers. They're complementary — one keeps the vault rich, the other scopes what the AI sees this session.

The single differentiating axis remains: **system-decides-relevance vs. user-decides-relevance.**

## 9. Roadmap

- **Phase 0 — Positioning.** Name (**PickMem**) and thesis locked. Remaining: MIT license, README-first. *(In progress.)*
- **Phase 1 — MVP.** Vault-backed `Store`, the TUI picker (full-screen grouped multi-select with a lens overlay), the MCP server exposing the picked slice, taxonomy templates, and `list`/`show` CLI for inspection. Demo GIF via VHS.
- **Phase 2 — Ingestion + web reach.** The inbox lifecycle (manual / import / auto-extract, staging, tap-accept), import routing + bulk review, and the Chrome extension (File System Access, picker panel, on-demand injection, 2–3 adapters + clipboard).
- **Phase 3 — Case study + flagship writeup.** A reproducible case study across 4–6 scenarios, three conditions each (curated vs. full-dump vs. naive auto-injection), with committed vault fixtures and prompts and at least one honest failure case. The thesis is qualitative, so the side-by-side contrast is the proof — and the best launch post.
- **Phase 4 — Adoption polish.** Docs, integration guides, good-first-issues, tagged releases; launch in the Obsidian community, r/LocalLLaMA, Show HN, dev.to.
- **Phase 5 — Stretch.** Silent companion process for live sync, more site adapters, optional auto-injection, richer provenance/recency.

Phases 0–3 are what matter for the portfolio; the Phase 3 case study is the leverage — resist scope creep before it exists.

## 10. Risks & open decisions

- **Audience narrowing.** Obsidian-canonical centers Obsidian users for the native/TUI path. The extension re-broadens to everyday tasks (shopping, brainstorming) on any web model. Net: a sharper, shippable tool, with breadth restored through the browser.
- **Page injection is per-site glue.** Adapters break when sites redesign; the clipboard fallback always works. This is ongoing maintenance, not a one-time build.
- **Privacy, stated precisely.** The vault stays on disk and PickMem never phones home. But injecting into ChatGPT sends that text to OpenAI — because you chose to send it there. The guarantee is "PickMem doesn't leak your brain," not "the model you paste into won't see what you pasted."
- **Injection default:** on-demand (deliberate, fits the thesis), with an optional per-site auto toggle.

## 11. Non-goals

Not a notes app, not an auto-summarizer, not a "remember everything" agent, not a cloud service. PickMem does not auto-inject, does not decide relevance for you, and does not edit the notes you write.

---

*Reference doc. Revisit Sections 4–6 before each build phase; revisit Section 1 whenever scope drifts toward "remember everything" — that's the failure mode this exists to refuse.*
