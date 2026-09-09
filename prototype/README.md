# PickMem — memory-extraction prototype

A standalone harness for tuning AI-driven memory extraction **before** wiring it
into the extension. Given a chat transcript and the user's vault, it asks Claude
Sonnet 5 to propose memory items, filed into existing groups in the vault's own
house style. Output is StageItem-shaped JSON — the same contract as PickMem's
`stage_memories` MCP tool — so a good prototype run integrates by "call
`stage_memories` with these items."

## Why this exists

The current deterministic `propose_memories` chunks the transcript on blank lines
and labels each chunk by truncation — it isn't extraction. This harness replaces
that with a model that (a) reads the vault, (b) infers the user's organizing
intent from it, and (c) decides what's memory-worthy, how to label it, and how to
consolidate related facts into one note.

## Layout

```
prompt.md              the extraction system prompt — tune this
fixtures/vault/        a small realistic vault (groups + notes)
transcripts/*.txt      sample chat transcripts to extract from
extract.ts             the runner: vault + transcript -> Sonnet 5 -> JSON
```

## Run

```bash
cd prototype
npm install
npm run extract -- transcripts/session1-coding.txt
```

More options:

```bash
# personal-chat sample, save the JSON
npm run extract -- transcripts/session2-personal.txt --out out.json

# point at a real vault instead of the fixture
npm run extract -- transcripts/session1-coding.txt --vault ~/PickMem-vault

# try a different model
npm run extract -- transcripts/session1-coding.txt --model claude-opus-5
```

## Auth

No key is hardcoded. The Anthropic SDK reads, in order: `ANTHROPIC_API_KEY`, then
an `ant auth login` profile. Set one:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

## The loop

1. Run against the sample transcripts.
2. Read the proposed items **and** the `skipped` list — the skipped list shows
   the model's judgment about what is *not* memory-worthy, which is half the game.
3. Tune `prompt.md`; add your own transcripts and vault fixtures for hard cases.
4. When the JSON is consistently good, the extension calls the same contract:
   scrape the transcript in the content script → run this extraction →
   `stage_memories` → items land as `pending` in the inbox → user tap-accepts in
   the picker. Nothing auto-activates.

## Notes

- `confidence`, `why`, and `skipped` are eval-only fields. Integration uses just
  `label`, `body`, and `suggested_group` (the StageItem fields).
- The runner flags any `suggested_group` the model invents that isn't in the
  vault — `stage_memories` drops those, so they should be rare.
