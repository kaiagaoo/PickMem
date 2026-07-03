import { test } from "node:test";
import assert from "node:assert/strict";
import { serializeNote, slugify } from "../src/vault/writenote.ts";
import { ulid } from "../src/vault/ulid.ts";
import { parseNote } from "../src/lib/frontmatter.ts";
import type { Frontmatter } from "../src/vault/types.ts";

// crypto.getRandomValues exists in Node 20+ globally; guard for older.
test("ulid produces a valid 26-char Crockford id", () => {
  const CROCKFORD = /^[0-9ABCDEFGHJKMNPQRSTVWXYZ]{26}$/;
  const seen = new Set<string>();
  for (let i = 0; i < 500; i++) {
    const id = ulid();
    assert.match(id, CROCKFORD, `bad ulid: ${id}`);
    assert.equal(seen.has(id), false, `duplicate ulid: ${id}`);
    seen.add(id);
  }
});

test("ulids are time-ordered (prefix grows over time)", async () => {
  const a = ulid();
  await new Promise((r) => setTimeout(r, 5));
  const b = ulid();
  // The 10-char time prefix should be non-decreasing.
  assert.ok(a.slice(0, 10) <= b.slice(0, 10), `${a} not <= ${b}`);
});

test("slugify matches the Go store's rules", () => {
  const cases: Record<string, string> = {
    "Client Acme — kickoff notes": "client-acme-kickoff-notes",
    "  weird   spacing!!  ": "weird-spacing",
    "": "note",
    "---": "note",
    "Salary 2026": "salary-2026",
  };
  for (const [inp, want] of Object.entries(cases)) {
    assert.equal(slugify(inp), want, `slugify(${JSON.stringify(inp)})`);
  }
});

test("serializeNote round-trips through parseNote", () => {
  const fm: Frontmatter = {
    id: ulid(),
    label: "income — freelance + salary",
    group: "finance/income",
    tags: ["money", "recurring"],
    source: "manual",
    status: "active",
    created_at: new Date().toISOString(),
  };
  const body = "Freelance ~= $5k/mo, salary $8k/mo.";
  const raw = serializeNote(fm, body);

  const parsed = parseNote(raw);
  assert.ok(parsed, "serialized note should parse");
  assert.equal(parsed.frontmatter.id, fm.id);
  assert.equal(parsed.frontmatter.label, fm.label);
  assert.equal(parsed.frontmatter.group, fm.group);
  assert.deepEqual(parsed.frontmatter.tags, fm.tags);
  assert.equal(parsed.frontmatter.source, "manual");
  assert.equal(parsed.frontmatter.status, "active");
  assert.equal(parsed.body.trim(), body);
});

test("serializeNote quotes labels that would break YAML", () => {
  const fm: Frontmatter = {
    id: ulid(),
    label: "budget: 2026 plan # draft",
    group: "finance",
    source: "manual",
    status: "active",
    created_at: new Date().toISOString(),
  };
  const raw = serializeNote(fm, "body");
  const parsed = parseNote(raw);
  assert.ok(parsed);
  assert.equal(parsed.frontmatter.label, "budget: 2026 plan # draft");
});

test("serializeNote omits tags when there are none", () => {
  const fm: Frontmatter = {
    id: ulid(),
    label: "x",
    group: "g",
    source: "manual",
    status: "active",
    created_at: new Date().toISOString(),
  };
  const raw = serializeNote(fm, "b");
  assert.equal(raw.includes("tags:"), false);
});
