import { test } from "node:test";
import assert from "node:assert/strict";
import { newULID } from "../src/lib/ulid.ts";
import { slugify } from "../src/lib/slug.ts";
import { serializeInboxNote, suggestGroup } from "../src/vault/capture.ts";
import { parseNote } from "../src/lib/frontmatter.ts";

// ---------- ulid ----------

const CROCKFORD = /^[0-9ABCDEFGHJKMNPQRSTVWXYZ]{26}$/;

test("newULID emits 26 Crockford chars that the Go loader would accept", () => {
  const id = newULID();
  assert.match(id, CROCKFORD);
  // ParseStrict also requires the leading char <= '7' so 128 bits fit.
  assert.ok(id[0]! <= "7", `first char ${id[0]} overflows 128 bits`);
});

test("newULID time component sorts lexicographically", () => {
  const a = newULID(1000000000000);
  const b = newULID(2000000000000);
  assert.ok(a.slice(0, 10) < b.slice(0, 10));
});

test("newULID is collision-resistant across calls in the same ms", () => {
  const now = Date.now();
  const seen = new Set<string>();
  for (let i = 0; i < 100; i++) seen.add(newULID(now));
  assert.equal(seen.size, 100);
});

// ---------- slugify (must match Go's vault.Slugify) ----------

test("slugify matches the Go implementation's behavior", () => {
  assert.equal(slugify("Editor preference"), "editor-preference");
  assert.equal(slugify("  spaces   everywhere  "), "spaces-everywhere");
  assert.equal(slugify("C++ & Go!"), "c-go");
  assert.equal(slugify("2026 budget"), "2026-budget");
  // Non-ASCII letters lowercase first, then collapse to dashes.
  assert.equal(slugify("面试准备"), "note");
  assert.equal(slugify(""), "note");
  assert.equal(slugify("!!!"), "note");
});

test("slugify caps at 60 chars without a trailing dash", () => {
  const s = slugify("a".repeat(59) + " bcd");
  assert.ok(s.length <= 60);
  assert.ok(!s.endsWith("-"));
});

// ---------- serializeInboxNote ----------

test("serialized capture note round-trips through the extension parser", () => {
  const id = newULID();
  const raw = serializeInboxNote(
    id,
    'Fable prefers "tabs" over spaces',
    "2026-07-03T10:00:00.000Z",
    "The user prefers tabs.\n\nSource: ChatGPT — https://chatgpt.com/c/abc",
    "about/preferences"
  );
  // Shape matches vault.Note.Serialize: ---\n<yaml>---\n\n<body>\n
  assert.ok(raw.startsWith("---\n"));
  assert.ok(raw.endsWith("\n"));
  assert.match(raw, /\n---\n\n/);

  const parsed = parseNote(raw);
  assert.ok(parsed, "extension parser must accept its own output");
  assert.equal(parsed.frontmatter.id, id);
  assert.equal(parsed.frontmatter.status, "pending");
  assert.equal(parsed.frontmatter.source, "extract");
  assert.equal(parsed.frontmatter.group, "");
  assert.equal(parsed.frontmatter.suggested_group, "about/preferences");
  assert.match(parsed.body, /prefers tabs/);
  assert.match(parsed.body, /Source: ChatGPT/);
});

test("serializeInboxNote omits suggested_group when routing found nothing", () => {
  const raw = serializeInboxNote(newULID(), "plain", "2026-07-03T10:00:00Z", "body");
  assert.ok(!raw.includes("suggested_group"));
});

test("serializeInboxNote keeps multi-line bodies out of the YAML block", () => {
  const raw = serializeInboxNote(
    newULID(),
    "label with\nnewline",
    "2026-07-03T10:00:00Z",
    "line1\nline2\n\n\n"
  );
  const parsed = parseNote(raw);
  assert.ok(parsed);
  // The newline in the label is escaped inside the quoted scalar, so the
  // frontmatter block stays line-oriented and the body survives intact.
  assert.equal(parsed.body, "line1\nline2\n");
});

// ---------- suggestGroup (must match routing.RulesClassifier) ----------

test("suggestGroup is case-insensitive, first match wins", () => {
  const rules = [
    { keyword: "salary", group: "finance/income" },
    { keyword: "sal", group: "wrong/if-order-broken" },
  ];
  assert.equal(suggestGroup(rules, "My SALARY is 8k"), "finance/income");
  assert.equal(suggestGroup(rules, "nothing relevant"), "");
  assert.equal(suggestGroup([], "anything"), "");
});
