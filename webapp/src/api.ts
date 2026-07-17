import type { NoteInput, State } from "./types";

// Every mutation endpoint returns the full, freshly-reloaded State, so the
// client is a pure function of the last response — no local cache to keep in
// sync. All calls go through request(), which throws on a non-2xx with the
// server's { error } message.

async function request(path: string, init?: RequestInit): Promise<State> {
  const res = await fetch(`/api${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : {};
  if (!res.ok) {
    throw new Error(data?.error || `${res.status} ${res.statusText}`);
  }
  return data as State;
}

export const api = {
  getState: () => request("/state"),

  addNote: (n: NoteInput) =>
    request("/notes", { method: "POST", body: JSON.stringify(n) }),

  editNote: (id: string, n: NoteInput) =>
    request(`/notes/${id}`, { method: "PATCH", body: JSON.stringify(n) }),

  deleteNote: (id: string) => request(`/notes/${id}`, { method: "DELETE" }),

  setActive: (itemIds: string[], activeLens = "") =>
    request("/active", {
      method: "PUT",
      body: JSON.stringify({ item_ids: itemIds, active_lens: activeLens }),
    }),

  acceptInbox: (id: string, group: string) =>
    request(`/inbox/${id}/accept`, {
      method: "POST",
      body: JSON.stringify({ group }),
    }),

  rejectInbox: (id: string) =>
    request(`/inbox/${id}/reject`, { method: "POST" }),

  saveLens: (name: string, itemIds?: string[]) =>
    request(`/lenses/${encodeURIComponent(name)}`, {
      method: "PUT",
      body: JSON.stringify(itemIds ? { item_ids: itemIds } : {}),
    }),

  useLens: (name: string) =>
    request(`/lenses/${encodeURIComponent(name)}/use`, { method: "POST" }),

  deleteLens: (name: string) =>
    request(`/lenses/${encodeURIComponent(name)}`, { method: "DELETE" }),

  createGroup: (path: string) =>
    request("/groups", { method: "POST", body: JSON.stringify({ path }) }),

  renameGroup: (from: string, to: string) =>
    request("/groups/rename", { method: "POST", body: JSON.stringify({ from, to }) }),

  deleteGroup: (path: string) =>
    request("/groups/delete", { method: "POST", body: JSON.stringify({ path }) }),

  setVaultName: (name: string) =>
    request("/vault/name", { method: "PUT", body: JSON.stringify({ name }) }),

  clearVault: () => request("/vault/clear", { method: "POST" }),

  importVault: (blob: unknown) =>
    request("/import", { method: "POST", body: JSON.stringify(blob) }),

  // exportVault bypasses request() because it returns a file download, not
  // the State envelope.
  exportVault: async (): Promise<unknown> => {
    const res = await fetch("/api/export");
    if (!res.ok) throw new Error(`export failed: ${res.status}`);
    return res.json();
  },
};
