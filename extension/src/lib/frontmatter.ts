// Minimal YAML frontmatter parser scoped to PickMem's contract. The
// frontmatter block is always the top of the file, delimited by `---`
// lines. Inside, we only handle:
//
//   key: scalar-string           -> string
//   key: [a, b, c]               -> string[]  (flow-style)
//   key:                         -> string[]  (block-style bullet list)
//     - a
//     - b
//   key: 2026-07-01T12:00:00Z    -> string    (kept as string; the Go
//                                              side round-trips through
//                                              time.Time but the extension
//                                              only ever displays)
//
// Anything more exotic (nested maps, anchors, multi-line scalars) is out
// of scope — PickMem doesn't emit it and shouldn't accept it silently.
// This keeps the extension free of a full YAML dependency (js-yaml is
// ~60KB, we don't need it).

import type { Frontmatter, Note, NoteSource, NoteStatus } from "../vault/types.ts";

export interface ParsedNote {
  frontmatter: Frontmatter;
  body: string;
}

const DELIM = "---";

/** Returns null when the file has no `---` block — the caller treats
 * that as "user's own Obsidian note, not a PickMem item." */
export function parseNote(raw: string): ParsedNote | null {
  const text = stripBOM(raw).replace(/\r\n/g, "\n");
  if (!text.startsWith(DELIM + "\n")) return null;
  const rest = text.slice(DELIM.length + 1);
  const end = rest.indexOf("\n" + DELIM);
  if (end < 0) return null;
  const yamlBlock = rest.slice(0, end);
  const body = rest.slice(end + DELIM.length + 1).replace(/^\n+/, "");

  const fm = parseYAML(yamlBlock);
  if (!fm.id || !fm.label) return null;
  return {
    frontmatter: coerceFrontmatter(fm),
    body,
  };
}

/** Convenience wrapper: parseNote + attach relPath. Used by the vault
 * reader after it has the on-disk path. */
export function parseNoteWithPath(raw: string, relPath: string): Note | null {
  const p = parseNote(raw);
  if (!p) return null;
  return { ...p.frontmatter, body: p.body, relPath };
}

// ---------- YAML subset ----------

type YAMLVal = string | string[];

/** Returns a shallow map of key -> string or string[]. Both flow-style
 *  (`[a, b]`) and block-style bullet lists are recognized. */
export function parseYAML(input: string): Record<string, YAMLVal> {
  const out: Record<string, YAMLVal> = {};
  const lines = input.split("\n");
  let i = 0;
  while (i < lines.length) {
    const line = lines[i];
    if (line == null || line.trim() === "" || line.trimStart().startsWith("#")) {
      i++;
      continue;
    }
    const m = /^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$/.exec(line);
    if (!m) {
      i++;
      continue;
    }
    const key = m[1]!;
    const rest = (m[2] ?? "").trim();
    if (rest === "") {
      // Possible block-style list follows.
      const list: string[] = [];
      let j = i + 1;
      while (j < lines.length) {
        const l = lines[j]!;
        const bm = /^\s+-\s+(.*)$/.exec(l);
        if (!bm) break;
        list.push(unquote(bm[1]!.trim()));
        j++;
      }
      out[key] = list;
      i = j;
      continue;
    }
    if (rest.startsWith("[")) {
      out[key] = parseFlowList(rest);
    } else {
      out[key] = unquote(rest);
    }
    i++;
  }
  return out;
}

function parseFlowList(s: string): string[] {
  const trimmed = s.trim();
  if (!trimmed.startsWith("[") || !trimmed.endsWith("]")) return [];
  const inner = trimmed.slice(1, -1).trim();
  if (inner === "") return [];
  // Split on top-level commas. This subset has no nested brackets so a
  // simple split is safe.
  return inner.split(",").map((p) => unquote(p.trim())).filter((p) => p !== "");
}

function unquote(s: string): string {
  if (s.length >= 2) {
    const first = s[0];
    const last = s[s.length - 1];
    if ((first === '"' && last === '"') || (first === "'" && last === "'")) {
      return s.slice(1, -1);
    }
  }
  return s;
}

function stripBOM(s: string): string {
  return s.charCodeAt(0) === 0xfeff ? s.slice(1) : s;
}

// ---------- Coerce to typed Frontmatter ----------

function coerceFrontmatter(raw: Record<string, YAMLVal>): Frontmatter {
  const s = (k: string) => (typeof raw[k] === "string" ? (raw[k] as string) : "");
  const arr = (k: string) =>
    Array.isArray(raw[k]) ? (raw[k] as string[]) : [];

  const source = normalizeSource(s("source"));
  const status = normalizeStatus(s("status"));

  const fm: Frontmatter = {
    id: s("id"),
    label: s("label"),
    group: s("group"),
    source,
    status,
    created_at: s("created_at"),
  };
  const tags = arr("tags");
  if (tags.length > 0) fm.tags = tags;
  const sg = s("suggested_group");
  if (sg) fm.suggested_group = sg;
  return fm;
}

function normalizeSource(s: string): NoteSource {
  if (s === "manual" || s === "import" || s === "extract") return s;
  // Default to manual for anything we don't recognize so a slightly-
  // future-shaped note doesn't get dropped.
  return "manual";
}

function normalizeStatus(s: string): NoteStatus {
  if (s === "active" || s === "pending") return s;
  return "active";
}
