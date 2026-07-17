// Wire types mirroring internal/web/state.go. Keep in sync with that file.

export interface Note {
  id: string;
  label: string;
  group: string;
  type: string;
  tags: string[];
  body: string;
  source: string;
  status: string;
  created_at: string;
  rel_path: string;
  tokens: number;
}

export interface Lens {
  name: string;
  item_ids: string[];
  count: number;
}

export interface Active {
  active_lens: string;
  item_ids: string[];
}

export interface VaultRef {
  path: string;
  name: string;
  exists: boolean;
  current: boolean;
}

export interface State {
  vault_path: string;
  vault_name: string;
  vaults: VaultRef[];
  notes: Note[];
  pending: Note[];
  groups: string[];
  lenses: Lens[];
  active: Active;
  context: string;
  tokens: number;
  note_types: string[];
  warnings: string[];
}

export interface NoteInput {
  label: string;
  group: string;
  body: string;
  type: string;
  tags: string[];
  to_inbox?: boolean;
}

// A view is a center-pane screen selected from the left sidebar.
export type View = "vault" | "inbox" | "lenses" | "settings" | "suggestions";

// Nav is the current location within the vault browser: a group (folder) or
// a single note. Root group is path "".
export type Nav =
  | { kind: "group"; path: string }
  | { kind: "note"; id: string };

// Portable whole-vault export/import shape (mirrors internal/web/portable.go).
export interface PortableVault {
  format_version: number;
  vault_name?: string;
  items: unknown[];
  lenses: unknown[];
  active_item_ids: string[];
  active_lens?: string;
}
