// Stages captured page text into pickmem/inbox/ as a pending note — the
// browser-side counterpart of the CLI's Store.AddInbox, and the sibling
// of createNote (which writes active notes). Both share the same on-disk
// serializer and slug/id helpers from writenote.ts + ulid.ts, so a
// captured note is byte-compatible with what the CLI would write and
// passes the Go store's ParseNote / ulid.ParseStrict on load.
//
// Capture only ever CREATES fresh inbox files; it never rewrites an
// existing note. Accept/reassign/reject stays with `pickmem review`.

import type { Frontmatter } from "./types.ts";
import { CONFIG_FILE, PICKMEM_DIR } from "./types.ts";
import { ulid } from "./ulid.ts";
import { serializeNote, slugify, pathExists, writeFileAt } from "./writenote.ts";

export interface CaptureParams {
  label: string;
  body: string;
  /** Routing suggestion carried until `pickmem review` accepts the note. */
  suggestedGroup?: string;
}

export interface CaptureResult {
  id: string;
  relPath: string;
}

export interface RoutingRule {
  keyword: string;
  group: string;
}

/** Write one captured note into pickmem/inbox/ as pending/extract. Slug
 *  collisions get the same short-id suffix the Go store and createNote
 *  use. */
export async function writeInboxNote(
  root: FileSystemDirectoryHandle,
  params: CaptureParams
): Promise<CaptureResult> {
  const id = ulid();
  const fm: Frontmatter = {
    id,
    label: params.label.trim(),
    group: "",
    source: "extract",
    status: "pending",
    created_at: new Date().toISOString(),
    ...(params.suggestedGroup ? { suggested_group: params.suggestedGroup } : {}),
  };
  const data = serializeNote(fm, params.body);

  const stem = slugify(params.label);
  let parts = [PICKMEM_DIR, "inbox", `${stem}.md`];
  if (await pathExists(root, parts)) {
    parts = [PICKMEM_DIR, "inbox", `${stem}-${id.slice(-6).toLowerCase()}.md`];
  }
  const relPath = parts.join("/");
  await writeFileAt(root, relPath, data);
  return { id, relPath };
}

/** Load the vault's routing rules (pickmem/config.json). Missing or
 *  malformed config degrades to no rules, same as the CLI. */
export async function loadRoutingRules(
  root: FileSystemDirectoryHandle
): Promise<RoutingRule[]> {
  try {
    const pickmem = await root.getDirectoryHandle(PICKMEM_DIR);
    const fh = await pickmem.getFileHandle(CONFIG_FILE);
    const text = await (await fh.getFile()).text();
    const cfg = JSON.parse(text) as { routing_rules?: RoutingRule[] };
    return (cfg.routing_rules ?? []).filter(
      (r) => r.keyword?.trim() !== "" && !!r.group
    );
  } catch {
    return [];
  }
}

/** Port of routing.RulesClassifier: case-insensitive substring match over
 *  the body, first rule wins, "" means no suggestion. */
export function suggestGroup(rules: RoutingRule[], body: string): string {
  const hay = body.toLowerCase();
  for (const r of rules) {
    if (hay.includes(r.keyword.toLowerCase())) return r.group;
  }
  return "";
}
