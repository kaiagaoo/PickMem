// Popup controller. Reads the vault on open, renders groups + lenses,
// tracks the pending selection in memory, writes active.json on Insert
// or Copy. All I/O is thin — the interesting logic lives in the
// composed modules (vault reader/writer, adapters, assemble).

import {
  clearVaultHandle,
  grantVault,
  hasPermission,
  loadVaultHandle,
  requestPermission,
} from "../vault/handle.ts";
import { readVault, type Vault } from "../vault/reader.ts";
import { assemble } from "../vault/assemble.ts";
import { saveActive, saveLenses, upsertLens } from "../vault/writer.ts";
import { createNote } from "../vault/writenote.ts";
import { copyToClipboard } from "../lib/clipboard.ts";
import type { Lens, Note } from "../vault/types.ts";
import {
  MSG_INJECT,
  MSG_PING,
  type InjectResponse,
  type PingResponse,
} from "../lib/messages.ts";

// ---------- state ----------

interface State {
  handle?: FileSystemDirectoryHandle;
  vault?: Vault;
  selected: Set<string>;
  activeLens: string; // "" when custom
  filter: string;
  adapter?: string;
  inputFound: boolean;
}

const state: State = {
  selected: new Set(),
  activeLens: "",
  filter: "",
  inputFound: false,
};

// ---------- boot ----------

document.addEventListener("DOMContentLoaded", async () => {
  wireButtons();

  // Ping the current tab so we know whether Insert is possible on this
  // site right away. Errors here mean the tab is a chrome:// page or the
  // content script hasn't loaded — treat as "no adapter."
  await pingActiveTab();

  const handle = await loadVaultHandle();
  if (!handle) {
    showEmpty();
    return;
  }
  // queryPermission does NOT require a user gesture, so this is safe to
  // call from a page-load handler. requestPermission DOES require one —
  // that only happens inside the button click below.
  state.handle = handle;
  if (await hasPermission(handle)) {
    await refreshVault();
  } else {
    showEmpty(true);
  }
});

function wireButtons() {
  qs("#btn-grant").addEventListener("click", async () => {
    try {
      if (state.handle) {
        // We already have a handle from a previous session; Chrome just
        // needs a fresh user-gesture-backed grant to re-activate it.
        const ok = await requestPermission(state.handle);
        if (!ok) {
          toast("Permission denied", true);
          return;
        }
      } else {
        state.handle = await grantVault();
      }
      await refreshVault();
    } catch (e) {
      toast(String(e), true);
    }
  });

  qs<HTMLInputElement>("#filter").addEventListener("input", (e) => {
    state.filter = (e.target as HTMLInputElement).value;
    renderItems();
  });

  qs("#btn-select-all").addEventListener("click", selectAllFiltered);
  qs("#btn-clear-all").addEventListener("click", clearAll);

  qs("#btn-save-lens").addEventListener("click", saveCurrentAsLens);
  qs<HTMLInputElement>("#lens-name").addEventListener("keydown", (e) => {
    if (e.key === "Enter") saveCurrentAsLens();
  });

  qs("#btn-copy").addEventListener("click", onCopy);
  qs("#btn-insert").addEventListener("click", onInsert);

  qs("#btn-add-toggle").addEventListener("click", () => {
    const form = qs("#add-form");
    form.classList.toggle("hidden");
    if (!form.classList.contains("hidden")) qs<HTMLInputElement>("#add-label").focus();
  });
  qs("#btn-add-cancel").addEventListener("click", closeAddForm);
  qs("#btn-add-save").addEventListener("click", onAddNote);

  // Switch vault: grantVault() persists the newly-picked handle over the
  // old one, so a cancelled picker (AbortError) leaves the current vault
  // connected — no clear-then-restore dance needed.
  qs("#btn-switch").addEventListener("click", async () => {
    try {
      state.handle = await grantVault();
      state.selected = new Set();
      state.activeLens = "";
      await refreshVault();
      toast(`Connected "${state.handle.name}"`);
    } catch (e) {
      if (e instanceof DOMException && e.name === "AbortError") return;
      toast(String(e), true);
    }
  });
}

async function refreshVault() {
  if (!state.handle) return;
  try {
    const vault = await readVault(state.handle);
    state.vault = vault;
    // Seed selection from active.json so re-opening the popup shows
    // what's currently active.
    state.selected = new Set(vault.activeSelection.item_ids);
    state.activeLens = vault.activeSelection.active_lens ?? "";
  } catch (e) {
    toast(`load failed: ${e instanceof Error ? e.message : String(e)}`, true);
    return;
  }
  showLoaded();
  renderVaultBar();
  renderInboxBadge();
  renderLenses();
  renderItems();
  renderSummary();
}

// ---------- header extras ----------

function renderVaultBar() {
  if (!state.handle) return;
  qs("#vault-bar").classList.remove("hidden");
  qs("#vault-name").textContent = state.handle.name;
  qs("#vault-name").title = "Connected vault folder";
}

// renderInboxBadge surfaces pending captures so they don't pile up
// invisibly — the browser can stage to the inbox but (for now) only
// `pickmem review` can accept.
function renderInboxBadge() {
  const el = qs("#inbox-badge");
  const n = state.vault?.pending.length ?? 0;
  if (n === 0) {
    el.classList.add("hidden");
    return;
  }
  el.classList.remove("hidden");
  el.textContent = `inbox ${n}`;
}

// ---------- rendering ----------

function renderLenses() {
  const el = qs("#lenses-list");
  el.innerHTML = "";
  if (!state.vault) return;
  if (state.vault.lenses.length === 0) {
    el.innerHTML = `<span class="lens-chip" style="cursor:default;color:var(--dim)">no lenses yet</span>`;
    return;
  }
  for (const l of state.vault.lenses) {
    const chip = document.createElement("button");
    chip.className = "lens-chip";
    if (l.name === state.activeLens) chip.classList.add("active");
    chip.textContent = `${l.name} · ${l.item_ids.length}`;
    chip.addEventListener("click", () => applyLens(l));
    el.appendChild(chip);
  }
}

// gNode is one segment of the group tree, built by splitting each note's
// group on "/". Mirrors the Go picker's groupNode so the two front-ends
// render the same shape.
interface gNode {
  name: string;
  fullPath: string;
  children: Map<string, gNode>;
  notes: Note[];
}

function buildTree(notes: Note[]): gNode {
  const root: gNode = { name: "", fullPath: "", children: new Map(), notes: [] };
  for (const n of notes) {
    let cur = root;
    let path = "";
    for (const seg of n.group.split("/")) {
      path = path ? `${path}/${seg}` : seg;
      let child = cur.children.get(seg);
      if (!child) {
        child = { name: seg, fullPath: path, children: new Map(), notes: [] };
        cur.children.set(seg, child);
      }
      cur = child;
    }
    cur.notes.push(n);
  }
  return root;
}

// descendantIds collects the ids of every note at or below node.
function descendantIds(node: gNode): string[] {
  const ids: string[] = [];
  const visit = (n: gNode) => {
    for (const note of n.notes) ids.push(note.id);
    for (const name of [...n.children.keys()].sort()) visit(n.children.get(name)!);
  };
  visit(node);
  return ids;
}

type GroupState = "none" | "some" | "all";
function groupState(ids: string[]): GroupState {
  if (ids.length === 0) return "none";
  let n = 0;
  for (const id of ids) if (state.selected.has(id)) n++;
  return n === 0 ? "none" : n === ids.length ? "all" : "some";
}

function renderItems() {
  const el = qs("#items-list");
  el.innerHTML = "";
  if (!state.vault) return;
  const q = state.filter.trim().toLowerCase();
  const filtered = state.vault.active.filter((n) => !q || matches(n, q));
  if (filtered.length === 0) {
    el.innerHTML = `<div class="group-header" style="color:var(--dim);text-transform:none">no matches</div>`;
    return;
  }
  const root = buildTree(filtered);
  // Depth-first walk, mirroring the Go picker: a group's own notes at this
  // depth, then each child group (header at this depth, its contents one
  // level deeper).
  const walk = (node: gNode, depth: number) => {
    for (const n of node.notes) el.appendChild(renderItem(n, depth));
    for (const name of [...node.children.keys()].sort()) {
      const child = node.children.get(name)!;
      el.appendChild(renderGroupHeader(child, depth));
      walk(child, depth + 1);
    }
  };
  walk(root, 0);
}

function indentPx(depth: number): string {
  return `${depth * 14 + 8}px`;
}

function renderGroupHeader(node: gNode, depth: number): HTMLElement {
  const ids = descendantIds(node);
  const st = groupState(ids);
  const row = document.createElement("div");
  row.className = "grouprow";
  row.style.paddingLeft = indentPx(depth);
  if (st !== "none") row.classList.add("selected");
  const box = document.createElement("span");
  box.className = "box";
  box.textContent = st === "all" ? "[x]" : st === "some" ? "[~]" : "[ ]";
  const label = document.createElement("span");
  label.className = "grouplabel";
  label.textContent = node.name;
  row.appendChild(box);
  row.appendChild(label);
  row.addEventListener("click", () => toggleGroup(ids));
  return row;
}

function renderItem(n: Note, depth: number): HTMLElement {
  const row = document.createElement("div");
  row.className = "item";
  row.style.paddingLeft = indentPx(depth);
  if (state.selected.has(n.id)) row.classList.add("selected");
  const box = document.createElement("span");
  box.className = "box";
  box.textContent = state.selected.has(n.id) ? "[x]" : "[ ]";
  const label = document.createElement("span");
  label.className = "label";
  label.textContent = n.label;
  row.appendChild(box);
  row.appendChild(label);
  if (n.tags && n.tags.length > 0) {
    const tags = document.createElement("span");
    tags.className = "tags";
    tags.textContent = "#" + n.tags.join(" #");
    row.appendChild(tags);
  }
  row.addEventListener("click", () => toggle(n.id));
  return row;
}

function renderSummary() {
  const label = state.activeLens || (state.selected.size ? "custom" : "none");
  const tokens = estimateTokens(selectedBodies());
  qs("#summary").textContent = `Active: ${label} · ${state.selected.size} selected · ~${tokens} tokens`;
  const insert = qs<HTMLButtonElement>("#btn-insert");
  insert.disabled = !state.inputFound || state.selected.size === 0;
}

// ---------- actions ----------

function toggle(id: string) {
  if (state.selected.has(id)) state.selected.delete(id);
  else state.selected.add(id);
  state.activeLens = ""; // manual edit breaks the lens
  renderLenses();
  renderItems();
  renderSummary();
}

// toggleGroup selects every note under a group header, or clears them all
// if the group is already fully selected — the same behavior as the TUI's
// header toggle.
function toggleGroup(ids: string[]) {
  if (groupState(ids) === "all") {
    for (const id of ids) state.selected.delete(id);
  } else {
    for (const id of ids) state.selected.add(id);
  }
  state.activeLens = "";
  renderLenses();
  renderItems();
  renderSummary();
}

// selectAllFiltered adds every item currently shown (respecting the filter)
// to the selection. Clearing the filter first would select the whole vault.
function selectAllFiltered() {
  if (!state.vault) return;
  const q = state.filter.trim().toLowerCase();
  for (const n of state.vault.active) {
    if (!q || matches(n, q)) state.selected.add(n.id);
  }
  state.activeLens = "";
  renderLenses();
  renderItems();
  renderSummary();
}

// clearAll empties the whole selection.
function clearAll() {
  state.selected.clear();
  state.activeLens = "";
  renderLenses();
  renderItems();
  renderSummary();
}

function applyLens(l: Lens) {
  state.selected = new Set(
    l.item_ids.filter((id) => state.vault?.byID.has(id))
  );
  state.activeLens = l.name;
  renderLenses();
  renderItems();
  renderSummary();
}

async function saveCurrentAsLens() {
  if (!state.handle || !state.vault) return;
  const name = qs<HTMLInputElement>("#lens-name").value.trim();
  if (!name) {
    toast("lens name required", true);
    return;
  }
  const lens: Lens = { name, item_ids: orderedSelectedIDs() };
  const next = upsertLens(state.vault.lenses, lens);
  try {
    await saveLenses(state.handle, next);
    state.vault.lenses = next;
    state.activeLens = name;
    qs<HTMLInputElement>("#lens-name").value = "";
    renderLenses();
    renderSummary();
    toast(`Saved lens "${name}"`);
  } catch (e) {
    toast(`save failed: ${String(e)}`, true);
  }
}

// ---------- add memory ----------

// populateGroupOptions fills the group <datalist> with the vault's
// existing group paths, so the user can pick one or type a new nested
// path.
function populateGroupOptions() {
  const dl = qs("#group-options");
  dl.innerHTML = "";
  if (!state.vault) return;
  const groups = new Set<string>();
  for (const n of state.vault.active) groups.add(n.group);
  for (const g of [...groups].sort()) {
    const opt = document.createElement("option");
    opt.value = g;
    dl.appendChild(opt);
  }
}

function closeAddForm() {
  qs("#add-form").classList.add("hidden");
  qs<HTMLInputElement>("#add-label").value = "";
  qs<HTMLInputElement>("#add-group").value = "";
  qs<HTMLInputElement>("#add-tags").value = "";
  qs<HTMLTextAreaElement>("#add-body").value = "";
}

async function onAddNote() {
  if (!state.handle) return;
  const label = qs<HTMLInputElement>("#add-label").value.trim();
  const group = qs<HTMLInputElement>("#add-group").value.trim();
  const body = qs<HTMLTextAreaElement>("#add-body").value.trim();
  const tags = qs<HTMLInputElement>("#add-tags")
    .value.split(",")
    .map((t) => t.trim())
    .filter((t) => t !== "");

  if (!label || !group || !body) {
    toast("label, group, and text are all required", true);
    return;
  }
  try {
    const note = await createNote(state.handle, { label, group, body, tags });
    closeAddForm();
    await refreshVault(); // re-read so the new note shows in the tree
    // Select the just-added note so it's ready to deliver.
    state.selected.add(note.id);
    state.activeLens = "";
    renderItems();
    renderSummary();
    toast(`Added "${label}"`);
  } catch (e) {
    toast(`add failed: ${String(e)}`, true);
  }
}

async function onCopy() {
  const block = buildBlock();
  await copyToClipboard(block);
  await persistActive();
  toast("Copied to clipboard");
}

async function onInsert() {
  if (!state.inputFound) return;
  const block = buildBlock();
  await copyToClipboard(block); // belt-and-suspenders — user can always paste
  await persistActive();
  try {
    const tab = await activeTab();
    if (!tab?.id) {
      toast("no active tab", true);
      return;
    }
    const res = (await chrome.tabs.sendMessage(tab.id, {
      type: MSG_INJECT,
      block,
    })) as InjectResponse | undefined;
    if (!res) {
      toast("no response from page — try Copy", true);
      return;
    }
    if (!res.ok) {
      toast(res.reason ?? "insert failed", true);
      return;
    }
    toast("Inserted");
  } catch (e) {
    toast(`inject failed: ${String(e)}`, true);
  }
}

async function persistActive() {
  if (!state.handle) return;
  await saveActive(state.handle, {
    ...(state.activeLens ? { active_lens: state.activeLens } : {}),
    item_ids: orderedSelectedIDs(),
  });
}

async function pingActiveTab() {
  try {
    const tab = await activeTab();
    if (!tab?.id) return;
    const res = (await chrome.tabs.sendMessage(tab.id, {
      type: MSG_PING,
    })) as PingResponse | undefined;
    if (!res || !res.ok) {
      qs("#site-status").textContent = "no adapter · clipboard only";
      state.inputFound = false;
      return;
    }
    state.adapter = res.adapter;
    state.inputFound = !!res.inputFound;
    qs("#site-status").textContent = res.inputFound
      ? `${res.adapter} · ready`
      : `${res.adapter} · input not found`;
  } catch {
    // No content script on this page (chrome:// or unmatched host).
    qs("#site-status").textContent = "no adapter · clipboard only";
    state.inputFound = false;
  }
}

// ---------- helpers ----------

function selectedBodies(): string[] {
  if (!state.vault) return [];
  const out: string[] = [];
  for (const id of state.selected) {
    const n = state.vault.byID.get(id);
    if (n) out.push(n.body);
  }
  return out;
}

/** Return selected ids in the display order (group-alphabetical, then
 *  file order within a group). Matches how the Go picker persists ids so
 *  the two write paths converge. */
function orderedSelectedIDs(): string[] {
  if (!state.vault) return [];
  const out: string[] = [];
  const sorted = [...state.vault.active].sort((a, b) => {
    if (a.group === b.group) return a.id.localeCompare(b.id);
    return a.group.localeCompare(b.group);
  });
  for (const n of sorted) {
    if (state.selected.has(n.id)) out.push(n.id);
  }
  return out;
}

function buildBlock(): string {
  const ids = orderedSelectedIDs();
  // assemble()'s own closing "--- end pickmem memory ---" line already
  // gives Insert/Copy a clean boundary against whatever the user types
  // next in the same chat input — no extra divider needed here.
  return assemble(
    {
      ...(state.activeLens ? { active_lens: state.activeLens } : {}),
      item_ids: ids,
    },
    (id) => state.vault?.byID.get(id)
  );
}

function estimateTokens(bodies: string[]): number {
  let total = 0;
  for (const b of bodies) total += b.length;
  if (total === 0) return 0;
  return Math.ceil(total / 4);
}

function matches(n: Note, q: string): boolean {
  return (
    n.label.toLowerCase().includes(q) ||
    n.body.toLowerCase().includes(q) ||
    (n.tags ?? []).some((t) => t.toLowerCase().includes(q))
  );
}

async function activeTab(): Promise<chrome.tabs.Tab | undefined> {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  return tabs[0];
}

function qs<T extends HTMLElement = HTMLElement>(sel: string): T {
  const el = document.querySelector<T>(sel);
  if (!el) throw new Error(`missing element: ${sel}`);
  return el;
}

function showEmpty(needsReconnect = false) {
  qs("#vault-empty").classList.remove("hidden");
  qs("#vault-bar").classList.add("hidden");
  qs("#inbox-badge").classList.add("hidden");
  qs("#lenses").classList.add("hidden");
  qs("#items").classList.add("hidden");
  qs("#save-lens").classList.add("hidden");
  if (needsReconnect) {
    qs("#vault-empty-msg").textContent =
      "Vault permission expired. Click below to reconnect.";
    qs<HTMLButtonElement>("#btn-grant").textContent = "Reconnect vault…";
  } else {
    qs("#vault-empty-msg").textContent = "No vault connected yet.";
    qs<HTMLButtonElement>("#btn-grant").textContent = "Choose vault folder…";
  }
  qs("#summary").textContent = "No vault. Copy still works — assembles from an empty selection.";
}

function showLoaded() {
  qs("#vault-empty").classList.add("hidden");
  qs("#lenses").classList.remove("hidden");
  qs("#items").classList.remove("hidden");
  qs("#add").classList.remove("hidden");
  qs("#save-lens").classList.remove("hidden");
  populateGroupOptions();
}

let toastTimer: number | undefined;
function toast(msg: string, err = false) {
  const el = qs("#toast");
  el.textContent = msg;
  el.classList.remove("hidden");
  el.classList.toggle("err", err);
  if (toastTimer !== undefined) window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => el.classList.add("hidden"), 2400);
}

// Silence the unused-import warning — kept so a future explicit
// "disconnect vault" action doesn't have to re-add the import. (Switching
// vaults doesn't need it: grantVault overwrites the stored handle.)
void clearVaultHandle;
