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
 *
 * Format: plain markdown, deliberately boring — see the design note in
 * internal/mcp/assemble.go for why this isn't XML-tagged. The closing
 * "--- end pickmem memory ---" line doubles as the boundary between this
 * block and whatever the user types next when Insert/Copy glue it into a
 * chat input; no separate divider is added on the extension side.
 */
export function assemble(
  active: Active,
  resolve: (id: string) => Note | undefined
): string {
  if (!active.item_ids || active.item_ids.length === 0) {
    return emptyBlock(active.active_lens);
  }
  const parts: string[] = [];
  parts.push("--- pickmem: selected memory");
  if (active.active_lens) {
    parts.push(` (lens: ${active.active_lens})`);
  }
  parts.push(" ---\n");
  let first = true;
  for (const id of active.item_ids) {
    const n = resolve(id);
    if (!n) continue;
    if (!first) parts.push("\n");
    const body = n.body.replace(/\n+$/g, "");
    parts.push(`${n.label} (${n.group}): ${body}\n`);
    first = false;
  }
  parts.push("--- end pickmem memory ---\n");
  return parts.join("");
}

function emptyBlock(lens: string | undefined): string {
  if (lens) {
    return `--- pickmem: lens "${lens}" is empty ---\n`;
  }
  return "--- pickmem: no memory selected ---\n";
}
