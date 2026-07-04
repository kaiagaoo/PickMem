// Service worker: owns the capture flow. Two entry points — the
// "Save selection to PickMem" context menu and the capture-selection
// keyboard command — both funnel into handleCapture(), which stages the
// selected text as a pending note in pickmem/inbox/ for `pickmem review`.
//
// Both entry points count as extension invocations, so Chrome grants
// activeTab: chrome.scripting works here on any site, not just the
// adapter-matched hosts. The vault handle comes from the same IndexedDB
// record the popup persists; requestPermission needs a user gesture the
// worker doesn't have, so if permission has lapsed we tell the user to
// open the popup (which re-requests on click) instead of failing silently.

import { loadVaultHandle, hasPermission } from "./vault/handle.ts";
import {
  loadRoutingRules,
  suggestGroup,
  writeInboxNote,
} from "./vault/capture.ts";

const MENU_ID = "pickmem-capture-selection";
const COMMAND_ID = "capture-selection";

chrome.runtime.onInstalled.addListener(() => {
  // removeAll first: onInstalled re-fires on extension updates and
  // create() with an existing id would error.
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: MENU_ID,
      title: "Save selection to PickMem",
      contexts: ["selection"],
    });
  });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId !== MENU_ID || !tab?.id) return;
  void handleCapture(tab, info.selectionText ?? "");
});

chrome.commands.onCommand.addListener((command, tab) => {
  if (command !== COMMAND_ID) return;
  void (async () => {
    const target = tab ?? (await activeTab());
    if (!target?.id) return;
    const selection = await readSelection(target.id);
    await handleCapture(target, selection);
  })();
});

// ---------- capture flow ----------

async function handleCapture(tab: chrome.tabs.Tab, text: string): Promise<void> {
  const tabId = tab.id!;
  const selection = text.trim();
  if (selection === "") {
    await notify(tabId, "PickMem: select some text first", true);
    return;
  }

  const root = await loadVaultHandle();
  if (!root) {
    await notify(tabId, "PickMem: no vault connected — open the popup to choose one", true);
    return;
  }
  if (!(await hasPermission(root))) {
    // requestPermission needs a user gesture; the popup's open handler
    // provides one. We can only point the user there.
    await notify(tabId, "PickMem: vault access expired — open the popup once to re-grant", true);
    return;
  }

  try {
    const body = buildBody(selection, tab);
    const rules = await loadRoutingRules(root);
    const suggested = suggestGroup(rules, body);
    const label = deriveLabel(selection);
    await writeInboxNote(root, {
      label,
      body,
      ...(suggested ? { suggestedGroup: suggested } : {}),
    });
    const where = suggested ? ` (suggested: ${suggested})` : "";
    await notify(tabId, `PickMem: saved to inbox${where} — review with \`pickmem review\``, false);
  } catch (e) {
    await notify(tabId, `PickMem: capture failed — ${String(e)}`, true);
  }
}

/** Selection + a trailing source line so the note keeps its provenance.
 *  The line is part of the body on purpose: review/edit can trim it, and
 *  until then it reads naturally in Obsidian and in the assembled block. */
function buildBody(selection: string, tab: chrome.tabs.Tab): string {
  const parts: string[] = [];
  if (tab.title) parts.push(tab.title);
  if (tab.url && /^https?:/.test(tab.url)) parts.push(tab.url);
  if (parts.length === 0) return selection;
  return `${selection}\n\nSource: ${parts.join(" — ")}`;
}

/** First ~48 chars of the selection, cut at a word boundary. */
function deriveLabel(selection: string): string {
  const collapsed = selection.replace(/\s+/g, " ").trim();
  if (collapsed.length <= 48) return collapsed;
  let cut = collapsed.slice(0, 48);
  const space = cut.lastIndexOf(" ");
  if (space > 24) cut = cut.slice(0, space);
  return cut + "…";
}

// ---------- tab helpers ----------

async function activeTab(): Promise<chrome.tabs.Tab | undefined> {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  return tabs[0];
}

async function readSelection(tabId: number): Promise<string> {
  try {
    const results = await chrome.scripting.executeScript({
      target: { tabId },
      func: () => String(window.getSelection() ?? ""),
    });
    return (results[0]?.result as string) ?? "";
  } catch {
    // Pages we can't script (chrome://, web store) — treated as empty.
    return "";
  }
}

/** In-page toast, injected with activeTab. Falls back to an action badge
 *  on pages that refuse script injection. */
async function notify(tabId: number, message: string, isError: boolean): Promise<void> {
  try {
    await chrome.scripting.executeScript({
      target: { tabId },
      func: showToast,
      args: [message, isError],
    });
  } catch {
    await badge(isError ? "!" : "✓");
  }
}

/** Runs in the page — must stay self-contained (no closures over worker
 *  scope). */
function showToast(message: string, isError: boolean): void {
  const ID = "pickmem-toast";
  document.getElementById(ID)?.remove();
  const el = document.createElement("div");
  el.id = ID;
  el.textContent = message;
  el.style.cssText = [
    "position:fixed",
    "top:16px",
    "right:16px",
    "z-index:2147483647",
    "max-width:360px",
    "padding:10px 14px",
    "border-radius:8px",
    "font:13px/1.4 -apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif",
    "color:#fff",
    `background:${isError ? "#b3261e" : "#1f6f43"}`,
    "box-shadow:0 4px 12px rgba(0,0,0,.25)",
    "transition:opacity .3s",
    "opacity:1",
  ].join(";");
  document.documentElement.appendChild(el);
  setTimeout(() => {
    el.style.opacity = "0";
    setTimeout(() => el.remove(), 350);
  }, 2600);
}

async function badge(text: string): Promise<void> {
  await chrome.action.setBadgeText({ text });
  setTimeout(() => {
    void chrome.action.setBadgeText({ text: "" });
  }, 3000);
}
