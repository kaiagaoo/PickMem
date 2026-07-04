// Creates new memory notes from the extension — the browser-side
// counterpart to `pickmem add`. This is create-only: it writes brand-new
// files and never rewrites an existing note (editing stays in Obsidian /
// the CLI, which hold the content-hash guard the browser can't).
//
// The serialized format must be readable by the Go store's ParseNote, so
// it mirrors the Go frontmatter layout (field order and types). It does
// not need to be byte-identical to Go's yaml.v3 output — just valid YAML
// frontmatter with the required fields.

import type { Frontmatter, Note } from "./types.ts";
import { ulid } from "./ulid.ts";

export interface NewNoteInput {
  label: string;
  group: string;
  body: string;
  tags?: string[];
}

// createNote writes a new active note into its group folder and returns
// the in-memory Note. The vault handle must already have read-write
// permission.
export async function createNote(
  root: FileSystemDirectoryHandle,
  input: NewNoteInput
): Promise<Note> {
  const fm: Frontmatter = {
    id: ulid(),
    label: input.label.trim(),
    group: input.group.trim().replace(/^\/+|\/+$/g, ""),
    source: "manual",
    status: "active",
    created_at: new Date().toISOString(),
    ...(input.tags && input.tags.length ? { tags: input.tags } : {}),
  };
  const body = input.body.replace(/\n+$/g, "");
  const data = serializeNote(fm, body);

  const relPath = await uniquePath(root, fm.group, slugify(fm.label), fm.id);
  await writeFileAt(root, relPath, data);

  return { ...fm, body, relPath };
}

// serializeNote renders a note to the on-disk `---`-frontmatter format.
export function serializeNote(fm: Frontmatter, body: string): string {
  const f: string[] = [];
  f.push(`id: ${fm.id}`);
  f.push(`label: ${yamlScalar(fm.label)}`);
  f.push(`group: ${yamlScalar(fm.group)}`);
  if (fm.tags && fm.tags.length) {
    f.push("tags:");
    for (const t of fm.tags) f.push(`  - ${yamlScalar(t)}`);
  }
  f.push(`source: ${fm.source}`);
  f.push(`status: ${fm.status}`);
  f.push(`created_at: ${fm.created_at}`);
  if (fm.suggested_group) f.push(`suggested_group: ${yamlScalar(fm.suggested_group)}`);
  return "---\n" + f.join("\n") + "\n---\n\n" + body.replace(/\n+$/g, "") + "\n";
}

// yamlScalar emits a bare scalar when it's unambiguous, otherwise a
// double-quoted string (JSON string syntax is a valid YAML double-quoted
// scalar for our inputs). This keeps simple labels clean while staying
// safe for values containing ":", "#", quotes, or leading indicators.
function yamlScalar(s: string): string {
  const risky =
    s === "" ||
    s !== s.trim() ||
    /^[\s>|@`"'!&*?{}[\],%#-]/.test(s) ||
    /:(\s|$)/.test(s) ||
    s.includes("#") ||
    s.includes("\n");
  return risky ? JSON.stringify(s) : s;
}

// slugify mirrors the Go store's Slugify: keep ASCII letters (lowercased)
// and digits, turn every other run into a single dash, trim dashes, cap
// at 60 chars, and fall back to "note" when nothing survives.
export function slugify(label: string): string {
  let out = "";
  for (const ch of label) {
    if (/[a-zA-Z]/.test(ch)) out += ch.toLowerCase();
    else if (/[0-9]/.test(ch)) out += ch;
    else out += " ";
  }
  out = out.trim().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
  if (out === "") out = "note";
  if (out.length > 60) out = out.slice(0, 60).replace(/-+$/g, "");
  return out;
}

// uniquePath returns a vault-relative path for a note in the given group.
// If <group>/<slug>.md is taken, it appends a short id suffix — the same
// collision strategy the Go store uses so two same-labelled notes don't
// clobber each other.
async function uniquePath(
  root: FileSystemDirectoryHandle,
  group: string,
  stem: string,
  id: string
): Promise<string> {
  const dirParts = group.split("/").filter((s) => s !== "");
  const base = [...dirParts, `${stem}.md`];
  if (!(await pathExists(root, base))) return base.join("/");
  const suffix = id.slice(-6).toLowerCase();
  return [...dirParts, `${stem}-${suffix}.md`].join("/");
}

export async function pathExists(
  root: FileSystemDirectoryHandle,
  parts: string[]
): Promise<boolean> {
  try {
    let cur = root;
    for (let i = 0; i < parts.length - 1; i++) {
      cur = await cur.getDirectoryHandle(parts[i]!);
    }
    await cur.getFileHandle(parts[parts.length - 1]!);
    return true;
  } catch {
    return false;
  }
}

export async function writeFileAt(
  root: FileSystemDirectoryHandle,
  relPath: string,
  data: string
): Promise<void> {
  const parts = relPath.split("/");
  let cur = root;
  for (let i = 0; i < parts.length - 1; i++) {
    cur = await cur.getDirectoryHandle(parts[i]!, { create: true });
  }
  const fh = await cur.getFileHandle(parts[parts.length - 1]!, { create: true });
  const stream = await fh.createWritable();
  await stream.write(data);
  await stream.close();
}
