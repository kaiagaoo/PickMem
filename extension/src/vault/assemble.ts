// Byte-parity port of internal/mcp/assemble.go. The extension and the
// MCP server must produce the same context block for the same active
// selection — that's the whole point of the shared data contract. If
// the Go side's format changes, this file changes too.

import type { Active, Note } from "./types.ts";

/**
 * assemble is the pure function tested against the Go side. Takes the
 * active selection and a resolver that returns a Note (or undefined) for
 * an id. Undefined-returning ids are skipped silently, matching Go's
 * behavior for stale references.
 */
export function assemble(
  active: Active,
  resolve: (id: string) => Note | undefined
): string {
  if (!active.item_ids || active.item_ids.length === 0) {
    return emptyMarker(active.active_lens);
  }
  const parts: string[] = [];
  if (active.active_lens) {
    parts.push(`<!-- pickmem lens: ${active.active_lens} -->\n\n`);
  }
  let first = true;
  for (const id of active.item_ids) {
    const n = resolve(id);
    if (!n) continue;
    if (!first) parts.push("\n---\n\n");
    parts.push(`# ${n.label}  ·  ${n.group}\n\n`);
    parts.push(n.body.replace(/\n+$/g, "") + "\n");
    first = false;
  }
  return parts.join("");
}

function emptyMarker(lens: string | undefined): string {
  if (lens) return `<!-- pickmem: lens "${lens}" is empty -->\n`;
  return "<!-- pickmem: no memory selected -->\n";
}
