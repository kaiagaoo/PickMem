import type { Note, State } from "./types";

export type CopyFormat = "text" | "markdown" | "json";

// pickedNotes returns the active notes in the order they appear in the
// active selection, dropping ids that point at deleted notes.
export function pickedNotes(state: State): Note[] {
  const byId = new Map(state.notes.map((n) => [n.id, n]));
  return state.active.item_ids
    .map((id) => byId.get(id))
    .filter((n): n is Note => !!n);
}

// buildContext renders the picked selection in the requested format. The
// plain-text form reuses the server's assembled block verbatim so it stays
// byte-identical to what the MCP server hands the model. Markdown and JSON
// are convenience variants built client-side.
export function buildContext(state: State, fmt: CopyFormat): string {
  const notes = pickedNotes(state);
  if (fmt === "text") return state.context;

  if (fmt === "json") {
    return JSON.stringify(
      notes.map((n) => ({
        label: n.label,
        body: n.body,
        group: n.group,
        tags: n.tags,
      })),
      null,
      2,
    );
  }

  // markdown: grouped by group heading
  if (notes.length === 0) return "_No memory selected._";
  const byGroup = new Map<string, Note[]>();
  for (const n of notes) {
    const g = n.group || "(ungrouped)";
    (byGroup.get(g) ?? byGroup.set(g, []).get(g)!).push(n);
  }
  const lines: string[] = ["# Selected memory", ""];
  for (const [group, items] of byGroup) {
    lines.push(`## ${group}`);
    for (const n of items) {
      lines.push(`- **${n.label}** — ${n.body}`);
    }
    lines.push("");
  }
  return lines.join("\n").trimEnd() + "\n";
}
