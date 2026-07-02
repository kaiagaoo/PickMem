# PickMem — Chrome extension

Reads your local PickMem vault (an Obsidian directory with a `pickmem/` subfolder), lets you pick which memory items to expose, and injects the selection into the chat input on ChatGPT / Claude.ai / Gemini. Model-agnostic by construction — because it's just text going into an `<input>`, it works on any surface those three cover, and the **Copy** button covers everything else.

Distributed as **load-unpacked source**, not from the Chrome Web Store — see [Non-goals](#non-goals) below.

---

## Install (three steps)

1. **Build the extension bundle.**
   ```bash
   cd extension/
   npm install
   npm run build
   ```
   This writes the bundle to `extension/dist/`. Nothing outside `dist/` is loaded by Chrome.

2. **Enable Developer Mode in Chrome.**
   Open `chrome://extensions`, flip the **Developer mode** toggle in the top-right.

3. **Load unpacked.**
   Click **Load unpacked** and select `extension/dist/`. PickMem's icon appears in the toolbar. If not, click the puzzle-piece icon and pin PickMem.

## First-time setup

Click the toolbar icon. On first open you'll see **Choose vault folder…** — click it and grant read-write access to your PickMem vault directory (the folder that contains `pickmem/`). Chrome persists the grant across sessions via the File System Access API.

- The grant is on the folder, not on individual files. PickMem never asks for anything outside that folder.
- If Chrome ever expires the grant (typically after a browser restart in incognito, or on a new session), the popup will re-request it automatically the next time you open it.

## What you can do

- **Toggle items** — click checkboxes to select memory items. Selection breaks any active lens (matches the TUI behavior).
- **Activate a lens** — click a lens chip. Selection is replaced with the lens's items.
- **Save a lens** — type a name in the bottom field and click **Save**. Writes to `pickmem/lenses.json`.
- **Copy** — assembles the block and writes to clipboard. Also persists the selection to `pickmem/active.json`.
- **Insert** — same as Copy, plus messages the active tab to prepend the block into the chat's input box (existing draft text is preserved below).

If the current tab isn't on ChatGPT / Claude.ai / Gemini, the header shows `no adapter · clipboard only` and **Insert** is disabled. **Copy** always works.

## Sites supported

| Site | URL match | How it inserts |
|------|-----------|----------------|
| ChatGPT | `chatgpt.com`, `chat.openai.com` | ProseMirror contenteditable |
| Claude.ai | `claude.ai` | ProseMirror contenteditable |
| Gemini | `gemini.google.com` | Quill contenteditable |

Adapters are declarative (URL pattern + input selector + insert method). Adding a site is a single entry in [`src/adapters/index.ts`](src/adapters/index.ts).

**When an adapter's selector breaks** (sites redesign), the popup shows a specific message and clipboard fallback still works. You'll get `couldn't find <Site>'s input (selector may have changed) — use "Copy" instead`.

## Permissions — what each one is for

The manifest requests the minimum:

- `clipboardWrite` — the **Copy** button. Works even without vault access; this is the low-permission escape hatch.
- `storage` — reserved for future prefs (currently unused; kept declared so future migrations don't need a manifest bump that Chrome would re-review).
- `activeTab` + `scripting` — send inject messages to the current tab only. No wildcard content-script injection into unrelated tabs.
- Host permissions for `chatgpt.com` / `chat.openai.com` / `claude.ai` / `gemini.google.com` — the three sites we ship adapters for. Content scripts run only on those hosts.

**Not requested:** `tabs` (no tab metadata), `history`, `cookies`, `webRequest`, `<all_urls>` host permissions. The File System Access grant is per-directory and per-origin — it lives in browser state, not in `chrome://permissions`.

**Not sent anywhere:** the extension has no network calls. It reads your vault, holds selection in memory, and writes back to `active.json` / `lenses.json`. Assembled context leaves the extension only when it lands in the chat input (which the site's own JS then sends to its own backend — that's the user's deliberate action).

## Manual test checklist

Run through this before shipping a new build:

- [ ] `npm install && npm run build` produces `dist/` with `manifest.json`, `popup.html`, `popup.js`, `content.js`, `background.js`, `icons/`, `popup.css`.
- [ ] `chrome://extensions` → **Load unpacked** → `dist/` → extension appears with no error banner.
- [ ] `npm test` reports all tests passing (frontmatter, assemble, adapter registry).
- [ ] `npm run typecheck` is clean.
- [ ] Popup opens without console errors on a fresh Chrome profile.
- [ ] **Grant flow**: click **Choose vault folder…**, pick a `pickmem init`-ed folder → items appear grouped, lenses appear as chips.
- [ ] **Toggle**: clicking an item flips its checkbox and updates the `N selected · ~T tokens` summary.
- [ ] **Filter**: typing in the filter box narrows items by label/body/tags.
- [ ] **Save-lens**: type a name → click **Save** → chip appears with the new lens.
- [ ] **Apply lens**: click an existing lens chip → selection replaces, chip goes accent-colored.
- [ ] **Copy on any site**: click **Copy** on any web page (even `example.com`) → clipboard has the assembled block starting with `# label  ·  group\n\n…`.
- [ ] **Insert on ChatGPT**: navigate to `chatgpt.com`, open popup → header says `ChatGPT · ready`, click **Insert** → block prepends into the composer, existing draft preserved.
- [ ] **Insert on Claude.ai**: same as above at `claude.ai`.
- [ ] **Insert on Gemini**: same at `gemini.google.com`.
- [ ] **No-adapter site**: navigate to `example.com` → header says `no adapter · clipboard only`, **Insert** is disabled, **Copy** works.
- [ ] **Vault write**: after Copy/Insert, `pickmem/active.json` in the vault contains `item_ids` matching the selection, and `active_lens` matches the popup's header.
- [ ] **Byte parity with MCP**: same selection → `pickmem serve` and the extension produce identical assembled blocks. Test: `pickmem pick` a lens, run `pickmem serve` and read `pickmem://active` via a minimal client; also click **Copy** in the popup; the strings should be byte-identical.

## Non-goals

**Not on the Chrome Web Store (yet).** M5's spec pins distribution to load-unpacked. Web Store submission is deferred until the permission surface is stable and the target audience shifts to non-developers. The store's review process specifically scrutinizes the combination we ship — local file access + content scripts on the three big AI sites — so we're deferring the review overhead until we're ready to answer it in depth.

**Not a notes editor.** The extension only reads notes and only writes `lenses.json` + `active.json`. All note creation and mutation lives in the CLI (`pickmem add`, `pickmem review`), because the create-only invariant lives in the Go store and can't be enforced from the browser.

**Not a memory server.** No network calls. No sync. No "smart" auto-injection. The block only reaches a model when the user clicks Insert (or pastes what they Copied).

## Development

```bash
npm run watch       # rebuild on save
npm run typecheck   # tsc --noEmit
npm test            # node --test (frontmatter + assemble + adapter tests)
npm run clean       # rm -rf dist
```

After a rebuild in `dist/`, Chrome needs a reload: `chrome://extensions` → click the reload icon next to PickMem. The popup HTML/CSS are copied straight from `src/popup/` on each build.

## Troubleshooting

**Popup shows "No vault connected"** — normal on first open. Click **Choose vault folder…**. If it keeps showing after granting, Chrome may have declined to persist the handle (private mode, guest profile): try a normal profile.

**Header shows "input not found"** — the site changed its DOM. Insert is disabled but Copy still works. File an issue; the adapter selector is a one-line fix.

**Insert clobbers existing text** — the intended behavior is to prepend and keep the draft below. If it replaces, you likely hit a site that intercepts the paste event unusually; use Copy as a workaround and file a bug with the site name.

**No lens chips appear** — `pickmem/lenses.json` may not exist yet. Run `pickmem pick` in the terminal once to seed it, or save one from the popup with the **Save** button.

**File System Access permission expired mid-session** — Chrome sometimes downgrades the grant after long idles. Close and reopen the popup; the permission re-request runs from your button click.
