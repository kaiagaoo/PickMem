// Mirror of the vault data contract (EXECUTION.md §3). Kept as a plain
// module of type-only declarations so tree-shaking drops it entirely from
// runtime bundles. If a field is added to the Go frontmatter struct, add
// it here too — the Go binary and this extension both read the same
// files.

export type NoteSource = "manual" | "import" | "extract";
export type NoteStatus = "active" | "pending";

export interface Frontmatter {
  id: string;
  label: string;
  group: string;
  tags?: string[];
  source: NoteSource;
  status: NoteStatus;
  created_at: string; // ISO 8601
  suggested_group?: string;
}

export interface Note extends Frontmatter {
  body: string;
  /** Path relative to the vault root, forward-slash separated. */
  relPath: string;
}

export interface Lens {
  name: string;
  item_ids: string[];
}

export interface Active {
  active_lens?: string;
  item_ids: string[];
}

/**
 * PICKMEM_DIR / INBOX_DIR / and the three well-known filenames match the
 * Go side exactly. If any of these change, EXECUTION.md §3 changes too
 * and both codebases have to move together.
 */
export const PICKMEM_DIR = "pickmem";
export const INBOX_DIR = "pickmem/inbox";
export const LENSES_FILE = "lenses.json";
export const ACTIVE_FILE = "active.json";
export const CONFIG_FILE = "config.json";
