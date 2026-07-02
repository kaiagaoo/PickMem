// Per-site adapters. Each adapter is a *declarative* pairing of URL
// matcher + input selector + insert method, so adding a new site is one
// entry — no per-site code paths.
//
// The insert semantics are locked (per M5 recommendation): prepend the
// assembled block, preserve any draft text below with a blank line
// separator. Users typing a question first and pulling in context second
// see it land above where they're typing.

export interface Adapter {
  /** Human-readable site name for the popup status area. */
  name: string;
  /** Matches window.location.host (or full URL — see matchesLocation). */
  urlPattern: RegExp;
  /**
   * Finds the input the model will read from. Return `null` if the
   * selector can't find one — the popup surfaces a message and falls
   * back to clipboard.
   */
  findInput: () => Element | null;
  /**
   * How to place text into the found element. Different sites use
   * `<textarea>` (native input events) vs `contenteditable` (needs a
   * text-node or InputEvent). Keeping this per-adapter avoids fragile
   * detection at insert time.
   */
  kind: "textarea" | "contenteditable" | "prosemirror";
}

const chatgpt: Adapter = {
  name: "ChatGPT",
  urlPattern: /(?:^|\.)chat(?:gpt|\.openai)\.com$/i,
  // ChatGPT ships both #prompt-textarea (contenteditable) and older
  // <textarea>-based UIs depending on rollout. Try both.
  findInput: () =>
    document.querySelector("#prompt-textarea") ??
    document.querySelector('textarea[data-testid="prompt-textarea"]') ??
    document.querySelector('div[contenteditable="true"][data-id="root"]'),
  kind: "prosemirror",
};

const claude: Adapter = {
  name: "Claude.ai",
  urlPattern: /(?:^|\.)claude\.ai$/i,
  // Claude.ai's composer is a ProseMirror-backed contenteditable div.
  findInput: () =>
    document.querySelector(
      'div[contenteditable="true"][role="textbox"], div.ProseMirror[contenteditable="true"]'
    ),
  kind: "prosemirror",
};

const gemini: Adapter = {
  name: "Gemini",
  urlPattern: /(?:^|\.)gemini\.google\.com$/i,
  // Gemini uses a rich-text editor: rich-textarea > .ql-editor (Quill).
  findInput: () =>
    document.querySelector("rich-textarea .ql-editor") ??
    document.querySelector('div.ql-editor[contenteditable="true"]'),
  kind: "contenteditable",
};

export const ADAPTERS: Adapter[] = [chatgpt, claude, gemini];

/** Find the adapter for the current page, or null if none matches. */
export function currentAdapter(host: string = location.host): Adapter | null {
  for (const a of ADAPTERS) {
    if (a.urlPattern.test(host)) return a;
  }
  return null;
}

/** Exported for tests: does the given host match this adapter? */
export function matchesLocation(adapter: Adapter, host: string): boolean {
  return adapter.urlPattern.test(host);
}

/**
 * Prepend `block` into the given element, preserving existing content
 * below. Returns true on success, false if we couldn't figure out how to
 * write to the element.
 *
 * For contenteditable inputs (ChatGPT/Claude/Gemini), a naïve
 * `innerText = ...` bypasses the framework's state (React/ProseMirror/
 * Quill), so we insert via clipboard-paste-style InputEvents — the
 * frameworks intercept those correctly.
 */
export function prependIntoInput(
  el: Element,
  kind: Adapter["kind"],
  block: string
): boolean {
  if (kind === "textarea" && el instanceof HTMLTextAreaElement) {
    const existing = el.value ?? "";
    // block already ends in its own trailing newline (see popup.ts's
    // buildBlock divider), so a single extra "\n" here yields exactly one
    // blank line before the user's draft — not two.
    const combined = existing.trimStart() ? block + "\n" + existing : block;
    el.value = combined;
    el.dispatchEvent(new Event("input", { bubbles: true }));
    // Move the caret to just after the injected block so the user's
    // next keystroke lands where they expect.
    const pos = block.length + 1;
    el.setSelectionRange(pos, pos);
    el.focus();
    return true;
  }

  // Contenteditable + ProseMirror + Quill: focus first, then simulate a
  // paste of the block + newline pair at the start of the input.
  const editable = el as HTMLElement;
  if (!editable.isContentEditable) return false;

  editable.focus();

  // Move caret to the start of the input so the block prepends.
  const selection = window.getSelection();
  if (selection) {
    const range = document.createRange();
    range.selectNodeContents(editable);
    range.collapse(true);
    selection.removeAllRanges();
    selection.addRange(range);
  }

  // Try the modern InputEvent pathway first; some editors intercept it
  // and route through their internal state.
  //
  // block already ends in its own trailing newline (see popup.ts's
  // buildBlock divider), so a single extra "\n" here yields exactly one
  // blank line before whatever the editable already contained — not two.
  const dataTransfer = new DataTransfer();
  dataTransfer.setData("text/plain", block + "\n");
  const paste = new ClipboardEvent("paste", {
    bubbles: true,
    cancelable: true,
    clipboardData: dataTransfer,
  });
  const dispatched = editable.dispatchEvent(paste);
  if (dispatched && !paste.defaultPrevented) {
    // Fallback: execCommand still works in every Chromium build we
    // target for the FSA API. Deprecated, but not removed.
    document.execCommand("insertText", false, block + "\n");
  }
  return true;
}
