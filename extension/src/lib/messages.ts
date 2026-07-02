// Message types exchanged between the popup and the content script.
// Kept in one file so both sides can import and stay in sync.

export const MSG_PING = "pickmem:ping";
export const MSG_INJECT = "pickmem:inject";

/** Popup → content script: is there an adapter for this tab, and can
 *  it find its input right now? */
export interface PingRequest {
  type: typeof MSG_PING;
}
export interface PingResponse {
  ok: boolean;
  /** Adapter name if one matches. */
  adapter?: string;
  /** True if the adapter found its input on the page. */
  inputFound?: boolean;
}

/** Popup → content script: paste this block into the chat input. */
export interface InjectRequest {
  type: typeof MSG_INJECT;
  block: string;
}
export interface InjectResponse {
  ok: boolean;
  /** Populated on failure so the popup can show a specific message. */
  reason?: string;
}

export type Request = PingRequest | InjectRequest;
export type Response = PingResponse | InjectResponse;
