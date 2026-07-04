import { test } from "node:test";
import assert from "node:assert/strict";
import { ulid } from "../src/vault/ulid.ts";
import { writeInboxNote, suggestGroup } from "../src/vault/capture.ts";
import { parseNote } from "../src/lib/frontmatter.ts";

// A minimal in-memory FileSystemDirectoryHandle good enough for capture:
// nested getDirectoryHandle + getFileHandle + createWritable. Records the
// last written file so tests can read it back.
function fakeVault() {
  const files = new Map<string, string>();
  function makeDir(prefix: string): any {
    return {
      async getDirectoryHandle(name: string, opts?: { create?: boolean }) {
        void opts;
        return makeDir(prefix ? `${prefix}/${name}` : name);
      },
      async getFileHandle(name: string, opts?: { create?: boolean }) {
        const path = prefix ? `${prefix}/${name}` : name;
        if (!opts?.create && !files.has(path)) throw new Error("not found");
        return {
          async getFile() {
            return { text: async () => files.get(path) ?? "" };
          },
          async createWritable() {
            let buf = "";
            return {
              async write(s: string) {
                buf += s;
              },
              async close() {
                files.set(path, buf);
              },
            };
          },
        };
      },
    };
  }
  return { root: makeDir("") as FileSystemDirectoryHandle, files };
}

test("writeInboxNote stages a pending/extract note the CLI can parse", async () => {
  const { root, files } = fakeVault();
  const res = await writeInboxNote(root, {
    label: 'Fable prefers "tabs" over spaces',
    body: "The user prefers tabs.\n\nSource: ChatGPT — https://chatgpt.com/c/abc",
    suggestedGroup: "about/preferences",
  });

  assert.match(res.relPath, /^pickmem\/inbox\/.+\.md$/);
  const raw = files.get(res.relPath)!;
  const parsed = parseNote(raw);
  assert.ok(parsed, "captured note must parse");
  assert.equal(parsed.frontmatter.id, res.id);
  assert.equal(parsed.frontmatter.status, "pending");
  assert.equal(parsed.frontmatter.source, "extract");
  assert.equal(parsed.frontmatter.group, "");
  assert.equal(parsed.frontmatter.type, "reference"); // captured web text
  assert.equal(parsed.frontmatter.suggested_group, "about/preferences");
  assert.match(parsed.body, /prefers tabs/);
  assert.match(parsed.body, /Source: ChatGPT/);
});

test("writeInboxNote omits suggested_group when routing found nothing", async () => {
  const { root, files } = fakeVault();
  const res = await writeInboxNote(root, { label: "plain", body: "a plain fact" });
  assert.ok(!files.get(res.relPath)!.includes("suggested_group"));
});

test("writeInboxNote disambiguates slug collisions with an id suffix", async () => {
  const { root } = fakeVault();
  const a = await writeInboxNote(root, { label: "same label", body: "first" });
  const b = await writeInboxNote(root, { label: "same label", body: "second" });
  assert.notEqual(a.relPath, b.relPath);
  assert.equal(a.relPath, "pickmem/inbox/same-label.md");
  assert.match(b.relPath, /^pickmem\/inbox\/same-label-[0-9a-z]{6}\.md$/);
});

test("captured id is a valid 26-char ULID the Go loader accepts", () => {
  const id = ulid();
  assert.match(id, /^[0-9ABCDEFGHJKMNPQRSTVWXYZ]{26}$/);
  assert.ok(id[0]! <= "7", "first char must keep the ULID within 128 bits");
});

test("suggestGroup is case-insensitive, first match wins", () => {
  const rules = [
    { keyword: "salary", group: "finance/income" },
    { keyword: "sal", group: "wrong/if-order-broken" },
  ];
  assert.equal(suggestGroup(rules, "My SALARY is 8k"), "finance/income");
  assert.equal(suggestGroup(rules, "nothing relevant"), "");
  assert.equal(suggestGroup([], "anything"), "");
});
