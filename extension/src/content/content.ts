// Runs on every adapter-matched page. Listens for two messages from the
// popup: ping (health check + show/hide the "Inject" button) and inject
// (prepend the assembled block into the chat input).

import {
  currentAdapter,
  prependIntoInput,
} from "../adapters/index.ts";
import {
  MSG_INJECT,
  MSG_PING,
  type InjectRequest,
  type InjectResponse,
  type PingResponse,
  type Request,
} from "../lib/messages.ts";

chrome.runtime.onMessage.addListener(
  (msg: Request, _sender, sendResponse: (r: unknown) => void) => {
    if (msg.type === MSG_PING) {
      const a = currentAdapter();
      const res: PingResponse = a
        ? { ok: true, adapter: a.name, inputFound: !!a.findInput() }
        : { ok: false };
      sendResponse(res);
      return;
    }

    if (msg.type === MSG_INJECT) {
      const res = doInject(msg);
      sendResponse(res);
      return;
    }
  }
);

function doInject(req: InjectRequest): InjectResponse {
  const a = currentAdapter();
  if (!a) {
    return { ok: false, reason: "no adapter for this site" };
  }
  const el = a.findInput();
  if (!el) {
    // Sites redesign; when a selector goes stale the popup should tell
    // the user to fall back to clipboard rather than fail silently.
    return {
      ok: false,
      reason: `couldn't find ${a.name}'s input (selector may have changed) — use "Copy" instead`,
    };
  }
  const ok = prependIntoInput(el, a.kind, req.block);
  if (!ok) {
    return {
      ok: false,
      reason: `couldn't write into ${a.name}'s input — use "Copy" instead`,
    };
  }
  return { ok: true };
}
