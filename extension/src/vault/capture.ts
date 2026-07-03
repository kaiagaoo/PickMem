// Stages captured text into pickmem/inbox/ as a pending note — the
// browser-side counterpart of the CLI's Store.AddInbox. This is the one
// place the extension creates a memory note, and it only ever creates
// pending inbox files with fresh ids; the create-only invariant on
// existing notes still holds (we never rewrite anything).
//
// Everything here must stay byte-compatible with the Go side: the CLI's
// loader rejects notes with invalid ULIDs, and `pickmem review` is how
// captured notes get accepted, so the frontmatter shape mirrors
// vault.Note.Serialize exactly (same field order, same delimiters).

import { newULID } from "../lib/ulid.ts";
import { slugify } from "../lib/slug.ts";
import { INBOX_DIR, PICKMEM_DIR, CONFIG_FILE } from "./types.ts";

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

/** Serialize a pending inbox note the way vault.Note.Serialize does:
 *  `---\n<yaml>---\n\n<body>\n`, fixed field order. Exported for tests. */
export function serializeInboxNote(
  id: string,
  label: string,
  createdAt: string,
  body: string,
  suggestedGroup?: string
): string {
  const lines = [
    "---",
    `id: ${id}`,
    `label: ${yamlQuote(label)}`,
    `group: ""`,
    "source: extract",
    "status: pending",
    `created_at: ${createdAt}`,
  ];
  if (suggestedGroup) {
    lines.push(`suggested_group: ${yamlQuote(suggestedGroup)}`);
  }
  lines.push("---", "");
  return lines.join("\n") + "\n" + body.replace(/\n+$/, "") + "\n";
}

/** Write one captured note into pickmem/inbox/. Slug collisions get the
 *  same short-id-suffix treatment as the Go side. */
export async function writeInboxNote(
  root: FileSystemDirectoryHandle,
  params: CaptureParams
): Promise<CaptureResult> {
  const id = newULID();
  const createdAt = new Date().toISOString();
  const data = serializeInboxNote(
    id,
    params.label,
    createdAt,
    params.body,
    params.suggestedGroup
  );

  const pickmem = await root.getDirectoryHandle(PICKMEM_DIR, { create: true });
  const inbox = await pickmem.getDirectoryHandle("inbox", { create: true });

  const stem = slugify(params.label);
  let name = `${stem}.md`;
  if (await fileExists(inbox, name)) {
    name = `${stem}-${id.slice(-6)}.md`;
  }

  const fh = await inbox.getFileHandle(name, { create: true });
  const stream = await fh.createWritable();
  await stream.write(data);
  await stream.close();

  return { id, relPath: `${INBOX_DIR}/${name}` };
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

// ---------- internal ----------

/** YAML scalar emitter: JSON double-quoting is valid YAML and handles
 *  every character class we can receive; the Go parser unescapes it
 *  correctly. Values that survive quoting unchanged after the extension's
 *  own minimal parser (lib/frontmatter.ts unquote) are the common case —
 *  labels with embedded quotes/backslashes render escaped there, which is
 *  a pre-existing display limitation, not a data-loss one. */
function yamlQuote(s: string): string {
  return JSON.stringify(s);
}

async function fileExists(
  dir: FileSystemDirectoryHandle,
  name: string
): Promise<boolean> {
  try {
    await dir.getFileHandle(name);
    return true;
  } catch {
    return false;
  }
}
