// Writes lenses.json and active.json. The extension never REWRITES a
// memory note — the create-only invariant lives on the Go side
// (Store.Update checks content hashes). The one place the extension
// creates notes is capture.ts, and it only ever creates fresh pending
// files under pickmem/inbox/. This module stays JSON-only.

import type { Active, Lens } from "./types.ts";
import { ACTIVE_FILE, LENSES_FILE, PICKMEM_DIR } from "./types.ts";

export async function saveActive(
  root: FileSystemDirectoryHandle,
  active: Active
): Promise<void> {
  // Normalize null/undefined item_ids to an empty array to match the Go
  // side's JSON shape ("item_ids": []).
  const normalized: Active = {
    ...(active.active_lens ? { active_lens: active.active_lens } : {}),
    item_ids: active.item_ids ?? [],
  };
  await writeJSON(root, [PICKMEM_DIR, ACTIVE_FILE], normalized);
}

export async function saveLenses(
  root: FileSystemDirectoryHandle,
  lenses: Lens[]
): Promise<void> {
  await writeJSON(root, [PICKMEM_DIR, LENSES_FILE], lenses ?? []);
}

/** Insert-or-replace a lens by name. Returns the new list — callers use
 *  this to update in-memory state after the write. */
export function upsertLens(lenses: Lens[], lens: Lens): Lens[] {
  const out = lenses.slice();
  const idx = out.findIndex((l) => l.name === lens.name);
  if (idx >= 0) out[idx] = lens;
  else out.push(lens);
  return out;
}

// ---------- internal ----------

async function writeJSON(
  root: FileSystemDirectoryHandle,
  parts: string[],
  value: unknown
): Promise<void> {
  // MkdirP each parent, matching Go's writeFileAtomic behavior in the
  // sense that we create the pickmem/ dir if it's missing.
  let cur: FileSystemDirectoryHandle = root;
  for (let i = 0; i < parts.length - 1; i++) {
    cur = await cur.getDirectoryHandle(parts[i]!, { create: true });
  }
  const fh = await cur.getFileHandle(parts[parts.length - 1]!, {
    create: true,
  });
  // FSA doesn't give us cross-file atomic rename, but createWritable
  // buffers until close() so a mid-write crash won't leave a truncated
  // file — good enough for a config-sized JSON blob.
  const stream = await fh.createWritable();
  const json = JSON.stringify(value, null, 2) + "\n";
  await stream.write(json);
  await stream.close();
}
