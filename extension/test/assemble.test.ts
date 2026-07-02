import { test } from "node:test";
import assert from "node:assert/strict";
import { assemble } from "../src/vault/assemble.ts";
import type { Note } from "../src/vault/types.ts";

/**
 * The assembled block must match the Go implementation byte-for-byte so
 * the MCP path and the extension path deliver identical context. These
 * fixtures are the reference — if you change either side, update both.
 */

function note(over: Partial<Note> & Pick<Note, "id" | "label" | "group" | "body">): Note {
  return {
    id: over.id,
    label: over.label,
    group: over.group,
    body: over.body,
    source: over.source ?? "manual",
    status: over.status ?? "active",
    created_at: over.created_at ?? "2026-07-01T12:00:00Z",
    relPath: over.relPath ?? `${over.group}/${over.label}.md`,
    ...(over.tags ? { tags: over.tags } : {}),
    ...(over.suggested_group ? { suggested_group: over.suggested_group } : {}),
  };
}

test("empty selection returns the no-memory marker", () => {
  const got = assemble({ item_ids: [] }, () => undefined);
  assert.equal(got, "<!-- pickmem: no memory selected -->\n");
});

test("empty selection under a named lens says which lens is empty", () => {
  const got = assemble({ active_lens: "Job-Hunt", item_ids: [] }, () => undefined);
  assert.equal(got, `<!-- pickmem: lens "Job-Hunt" is empty -->\n`);
});

test("single item gets a header + body, no separator", () => {
  const n = note({ id: "01A", label: "salary", group: "financial", body: "monthly base $8k" });
  const got = assemble({ item_ids: ["01A"] }, (id) => (id === "01A" ? n : undefined));
  assert.equal(got, "# salary  ·  financial\n\nmonthly base $8k\n");
});

test("multiple items get separators; header first, --- between", () => {
  const a = note({ id: "01A", label: "salary", group: "financial", body: "monthly base $8k" });
  const b = note({ id: "01B", label: "kickoff", group: "work", body: "Aug 1" });
  const got = assemble({ item_ids: ["01A", "01B"] }, (id) =>
    id === "01A" ? a : id === "01B" ? b : undefined
  );
  assert.equal(
    got,
    "# salary  ·  financial\n\nmonthly base $8k\n\n---\n\n# kickoff  ·  work\n\nAug 1\n"
  );
  // Byte-parity with Go's fmt.Fprintf(&b, "# %s  ·  %s\n\n", ...) followed by
  // TrimRight(body, "\n") + "\n".
});

test("active_lens header is present when lens is set", () => {
  const n = note({ id: "01A", label: "x", group: "g", body: "body" });
  const got = assemble({ active_lens: "Weekend", item_ids: ["01A"] }, () => n);
  assert.equal(
    got,
    "<!-- pickmem lens: Weekend -->\n\n# x  ·  g\n\nbody\n"
  );
});

test("stale ids are silently skipped, matching Go's behavior", () => {
  const a = note({ id: "01A", label: "x", group: "g", body: "body" });
  const got = assemble({ item_ids: ["01A", "01Z-deleted"] }, (id) =>
    id === "01A" ? a : undefined
  );
  // Only the live item shows up; no error, no gap.
  assert.equal(got, "# x  ·  g\n\nbody\n");
});

test("trailing newlines in body get normalized to exactly one", () => {
  const a = note({ id: "01A", label: "x", group: "g", body: "body\n\n\n\n" });
  const got = assemble({ item_ids: ["01A"] }, () => a);
  assert.equal(got, "# x  ·  g\n\nbody\n");
});
