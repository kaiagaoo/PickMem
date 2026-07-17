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

export interface State {
  vault_path: string;
  vault_name: string;
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
