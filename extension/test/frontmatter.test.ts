import { test } from "node:test";
import assert from "node:assert/strict";
import { parseNote, parseYAML } from "../src/lib/frontmatter.ts";

test("parseNote round-trips the Go-emitted format", () => {
  const raw = `---
id: 01JAX5D9KX3M8VYZ8T5EK5JY7C
label: income — freelance + salary
group: financial
tags:
  - money
  - recurring
source: manual
status: active
created_at: 2026-07-01T12:00:00Z
---

Freelance ~= $5k/mo, salary $8k/mo.
Bonuses land in March.
`;
  const parsed = parseNote(raw);
  assert.ok(parsed, "expected a parsed note");
  assert.equal(parsed.frontmatter.id, "01JAX5D9KX3M8VYZ8T5EK5JY7C");
  assert.equal(parsed.frontmatter.label, "income — freelance + salary");
  assert.equal(parsed.frontmatter.group, "financial");
  assert.deepEqual(parsed.frontmatter.tags, ["money", "recurring"]);
  assert.equal(parsed.frontmatter.source, "manual");
  assert.equal(parsed.frontmatter.status, "active");
  assert.equal(parsed.frontmatter.created_at, "2026-07-01T12:00:00Z");
  assert.match(parsed.body, /Freelance/);
});

test("parseNote returns null when the file has no frontmatter block", () => {
  // User's own Obsidian note — extension should ignore.
  assert.equal(parseNote("# just a diary entry\n\nhello\n"), null);
});

test("parseNote returns null when required fields are missing", () => {
  const raw = `---
label: no id here
group: x
source: manual
status: active
created_at: 2026-07-01T12:00:00Z
---

body
`;
  assert.equal(parseNote(raw), null);
});

test("parseYAML handles flow-style tag arrays", () => {
  const y = parseYAML(`id: abc\ntags: [money, recurring, "with space"]\n`);
  assert.deepEqual(y.tags, ["money", "recurring", "with space"]);
});

test("parseYAML preserves inbox-only suggested_group", () => {
  const raw = `---
id: 01JBY2Q3
label: gift ideas
group: personal
source: extract
status: pending
suggested_group: relationships
created_at: 2026-07-01T12:00:00Z
---

plants, enamel pins
`;
  const parsed = parseNote(raw);
  assert.ok(parsed);
  assert.equal(parsed.frontmatter.status, "pending");
  assert.equal(parsed.frontmatter.suggested_group, "relationships");
});

test("parseNote strips a leading BOM", () => {
  const raw = `﻿---\nid: X\nlabel: y\ngroup: g\nsource: manual\nstatus: active\ncreated_at: 2026-07-01T12:00:00Z\n---\n\nbody\n`;
  const parsed = parseNote(raw);
  assert.ok(parsed, "BOM should not break the parser");
  assert.equal(parsed.frontmatter.id, "X");
});

test("unrecognized source/status fall back to safe defaults", () => {
  // Forward-compat: a future frontmatter shape shouldn't drop the whole
  // note; we degrade to `manual` / `active` and let the user resolve.
  const raw = `---
id: X
label: y
group: g
source: automation
status: archived
created_at: 2026-07-01T12:00:00Z
---

body
`;
  const parsed = parseNote(raw);
  assert.ok(parsed);
  assert.equal(parsed.frontmatter.source, "manual");
  assert.equal(parsed.frontmatter.status, "active");
});
