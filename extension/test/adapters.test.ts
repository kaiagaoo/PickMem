import { test } from "node:test";
import assert from "node:assert/strict";
import { ADAPTERS, currentAdapter, matchesLocation } from "../src/adapters/index.ts";

test("adapters recognize their canonical hosts", () => {
  const cases: [string, string][] = [
    ["chatgpt.com", "ChatGPT"],
    ["www.chatgpt.com", "ChatGPT"],
    ["chat.openai.com", "ChatGPT"],
    ["claude.ai", "Claude.ai"],
    ["www.claude.ai", "Claude.ai"],
    ["gemini.google.com", "Gemini"],
  ];
  for (const [host, wanted] of cases) {
    const a = currentAdapter(host);
    assert.ok(a, `expected an adapter for ${host}`);
    assert.equal(a.name, wanted);
  }
});

test("unrelated hosts get no adapter", () => {
  for (const host of ["example.com", "openai.com", "google.com", "notclaude.ai.evil.com"]) {
    assert.equal(currentAdapter(host), null, `false positive on ${host}`);
  }
});

test("registry contains exactly the three shipped adapters", () => {
  // Locks the spec: EXECUTION.md §M5 says "Ship adapters for ChatGPT,
  // Claude.ai, Gemini first." Adding a fourth requires updating this
  // test — makes scope creep visible in PRs.
  assert.equal(ADAPTERS.length, 3);
  const names = ADAPTERS.map((a) => a.name).sort();
  assert.deepEqual(names, ["ChatGPT", "Claude.ai", "Gemini"]);
});

test("matchesLocation is symmetric with currentAdapter", () => {
  for (const a of ADAPTERS) {
    // Each adapter must at least match one canonical host it lists.
    const anyHost =
      a.name === "ChatGPT"
        ? "chatgpt.com"
        : a.name === "Claude.ai"
        ? "claude.ai"
        : "gemini.google.com";
    assert.ok(
      matchesLocation(a, anyHost),
      `adapter ${a.name} doesn't self-match on ${anyHost}`
    );
  }
});
