// Walks a granted vault directory and returns everything the popup
// needs: notes (id-keyed), lenses, active. Skips hidden directories
// (`.git`, `.obsidian`, `.trash`) and pickmem/inbox/ from the active
// scan — pending notes are surfaced separately.

import { parseNoteWithPath } from "../lib/frontmatter.ts";
import type { Active, Lens, Note } from "./types.ts";
import {
  ACTIVE_FILE,
  CONFIG_FILE,
  LENSES_FILE,
  PICKMEM_DIR,
} from "./types.ts";

export interface Vault {
  active: Note[];
  pending: Note[];
  lenses: Lens[];
  activeSelection: Active;
  byID: Map<string, Note>;
  /** Quick-pick tag chips offered in the add form (config.suggested_tags,
   *  falling back to the built-in defaults). */
  suggestedTags: string[];
}

// The built-in quick-pick tags a vault starts with, matching the Go side's
// DefaultSuggestedTags. They are ordinary tags, not a required vocabulary.
export const DEFAULT_SUGGESTED_TAGS = ["fact", "idea", "thought", "reference"];

/** Read the full vault contents. Cheap enough (~10-100ms for a few
 *  hundred notes on modern SSDs) to re-run every popup open — avoids the
 *  bookkeeping of a persistent cache when Obsidian may have edited files
 *  between sessions. */
export async function readVault(root: FileSystemDirectoryHandle): Promise<Vault> {
  const active: Note[] = [];
  const pending: Note[] = [];
  const byID = new Map<string, Note>();

  await walk(root, "", async (path, file) => {
    if (!path.toLowerCase().endsWith(".md")) return;
    const text = await file.text();
    if (!text.startsWith("---\n") && !text.startsWith("﻿---\n")) return;
    const note = parseNoteWithPath(text, path);
    if (!note) return;
    if (byID.has(note.id)) return; // duplicate — first wins
    byID.set(note.id, note);
    if (note.status === "pending") pending.push(note);
    else active.push(note);
  });

  const lenses = await readJSON<Lens[]>(root, [PICKMEM_DIR, LENSES_FILE], []);
  const activeSelection = await readJSON<Active>(root, [PICKMEM_DIR, ACTIVE_FILE], {
    item_ids: [],
  });

  // suggested_tags is the current key; note_types is the pre-tags legacy key.
  const cfg = await readJSON<{ suggested_tags?: string[]; note_types?: string[] }>(
    root,
    [PICKMEM_DIR, CONFIG_FILE],
    {}
  );
  const configured = cfg.suggested_tags?.length
    ? cfg.suggested_tags
    : cfg.note_types ?? [];
  const suggestedTags = configured.length ? configured : DEFAULT_SUGGESTED_TAGS;

  return { active, pending, lenses, activeSelection, byID, suggestedTags };
}

/** Enumerate every group name currently in use across active notes.
 *  Preserves order-of-first-appearance so a stable UI can show them
 *  without needing to sort every render. */
export function groupsOf(active: Note[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const n of active) {
    if (!seen.has(n.group)) {
      seen.add(n.group);
      out.push(n.group);
    }
  }
  out.sort();
  return out;
}

// ---------- internals ----------

async function walk(
  dir: FileSystemDirectoryHandle,
  prefix: string,
  visit: (path: string, file: File) => Promise<void>
): Promise<void> {
  // TS's lib.dom.d.ts declares .entries() but not .values(); prefer
  // entries() which every implementation supports.
  const entries = (dir as unknown as {
    entries: () => AsyncIterable<[string, FileSystemHandle]>;
  }).entries();

  for await (const [name, handle] of entries) {
    if (isSkipDir(name, prefix)) continue;
    const rel = prefix ? `${prefix}/${name}` : name;
    if (handle.kind === "directory") {
      await walk(handle as FileSystemDirectoryHandle, rel, visit);
      continue;
    }
    if (handle.kind === "file") {
      // pickmem/config.json / lenses.json / active.json are not markdown
      // and don't need to route through visit.
      if (!rel.endsWith(".md")) continue;
      const file = await (handle as FileSystemFileHandle).getFile();
      await visit(rel, file);
    }
  }
}

function isSkipDir(name: string, prefix: string): boolean {
  // Never descend into hidden folders (dotdirs) — Obsidian's .obsidian
  // and any .git / .trash / node_modules are the common cases.
  if (name.startsWith(".") && prefix !== "") return false;
  if (name.startsWith(".")) return true;
  if (name === "node_modules") return true;
  return false;
}

async function readJSON<T>(
  root: FileSystemDirectoryHandle,
  parts: string[],
  fallback: T
): Promise<T> {
  try {
    let cur: FileSystemDirectoryHandle = root;
    for (let i = 0; i < parts.length - 1; i++) {
      cur = await cur.getDirectoryHandle(parts[i]!);
    }
    const fh = await cur.getFileHandle(parts[parts.length - 1]!);
    const file = await fh.getFile();
    const text = await file.text();
    return JSON.parse(text) as T;
  } catch {
    return fallback;
  }
}

// Re-export so popup imports don't need to know the constants module.
export { PICKMEM_DIR, CONFIG_FILE };
