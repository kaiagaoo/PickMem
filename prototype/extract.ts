// Standalone memory-extraction prototype for PickMem.
//
// Reads the user's vault (groups + example notes) and a chat transcript,
// asks Claude Sonnet 5 to propose memory items, and prints/saves the result
// as StageItem-shaped JSON — the same contract as PickMem's stage_memories
// MCP tool. This is decoupled from the extension and the Go server on
// purpose: iterate on prompt.md until the JSON is consistently good, then
// wire the same contract into the extension.
//
//   npx tsx extract.ts <transcript.txt> [--vault DIR] [--out FILE] [--model ID]
//
// Auth: reads ANTHROPIC_API_KEY, or an `ant auth login` profile — no key is
// hardcoded. See README.md.

import Anthropic from "@anthropic-ai/sdk";
import { readFile, readdir, writeFile } from "node:fs/promises";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { dirname } from "node:path";

const HERE = dirname(fileURLToPath(import.meta.url));

// ---------- args ----------

interface Args {
  transcript: string;
  vault: string;
  out?: string;
  model: string;
}

function parseArgs(argv: string[]): Args {
  const a: Partial<Args> = {
    vault: join(HERE, "fixtures", "vault"),
    model: "claude-sonnet-5",
  };
  const positional: string[] = [];
  for (let i = 0; i < argv.length; i++) {
    const t = argv[i]!;
    if (t === "--vault") a.vault = argv[++i];
    else if (t === "--out") a.out = argv[++i];
    else if (t === "--model") a.model = argv[++i];
    else positional.push(t);
  }
  a.transcript = positional[0] ?? join(HERE, "transcripts", "session1-coding.txt");
  return a as Args;
}

// ---------- vault reader (frontmatter parse mirrors extension/src/lib/frontmatter.ts) ----------

interface VaultNote {
  group: string;
  label: string;
  body: string;
}

/** Parse one .md file's `---` frontmatter block into group/label/body.
 *  Returns null for files without a valid PickMem frontmatter block. */
function parseNote(raw: string): VaultNote | null {
  const text = raw.replace(/^﻿/, "").replace(/\r\n/g, "\n");
  if (!text.startsWith("---\n")) return null;
  const rest = text.slice(4);
  const end = rest.indexOf("\n---");
  if (end < 0) return null;
  const yaml = rest.slice(0, end);
  const body = rest.slice(end + 4).replace(/^\n+/, "").trimEnd();

  const fm: Record<string, string> = {};
  for (const line of yaml.split("\n")) {
    const m = /^([A-Za-z_][A-Za-z0-9_]*)\s*:\s*(.*)$/.exec(line);
    if (!m) continue;
    let v = (m[2] ?? "").trim();
    if (
      v.length >= 2 &&
      ((v[0] === '"' && v.at(-1) === '"') || (v[0] === "'" && v.at(-1) === "'"))
    ) {
      v = v.slice(1, -1);
    }
    fm[m[1]!] = v;
  }
  if (!fm.id || !fm.label) return null; // not a PickMem note
  return { group: fm.group ?? "", label: fm.label, body };
}

async function readVault(root: string): Promise<VaultNote[]> {
  const notes: VaultNote[] = [];
  async function walk(dir: string): Promise<void> {
    let entries;
    try {
      entries = await readdir(dir, { withFileTypes: true });
    } catch {
      return;
    }
    for (const e of entries) {
      const p = join(dir, e.name);
      if (e.isDirectory()) await walk(p);
      else if (e.isFile() && e.name.endsWith(".md")) {
        const n = parseNote(await readFile(p, "utf8"));
        if (n) notes.push(n);
      }
    }
  }
  await walk(root);
  return notes;
}

/** Render the vault as prompt context: the full group list + up to two
 *  example notes per group, so the model can infer taxonomy + house style. */
function renderVaultContext(notes: VaultNote[]): { groups: string[]; text: string } {
  const byGroup = new Map<string, VaultNote[]>();
  for (const n of notes) {
    if (!n.group) continue;
    (byGroup.get(n.group) ?? byGroup.set(n.group, []).get(n.group)!).push(n);
  }
  const groups = [...byGroup.keys()].sort();

  const lines: string[] = [];
  lines.push("### Existing groups");
  for (const g of groups) lines.push(`- ${g}`);
  lines.push("");
  lines.push("### Example notes (for taxonomy + house style)");
  for (const g of groups) {
    for (const n of byGroup.get(g)!.slice(0, 2)) {
      lines.push(`\n[group: ${g}] ${n.label}`);
      lines.push(n.body);
    }
  }
  return { groups, text: lines.join("\n") };
}

// ---------- extraction ----------

/** Pull the JSON object out of a text response, tolerating a stray
 *  ```json fence if the model adds one despite the instruction not to. */
function extractJson(text: string): string {
  const fence = /```(?:json)?\s*([\s\S]*?)```/.exec(text);
  if (fence) return fence[1]!.trim();
  const start = text.indexOf("{");
  const end = text.lastIndexOf("}");
  if (start >= 0 && end > start) return text.slice(start, end + 1);
  return text.trim();
}

async function main() {
  const args = parseArgs(process.argv.slice(2));

  const [systemPrompt, transcript, vaultNotes] = await Promise.all([
    readFile(join(HERE, "prompt.md"), "utf8"),
    readFile(args.transcript, "utf8"),
    readVault(args.vault),
  ]);
  const { groups, text: vaultContext } = renderVaultContext(vaultNotes);

  console.error(
    `vault: ${vaultNotes.length} notes, ${groups.length} groups (${args.vault})`
  );
  console.error(`transcript: ${relative(HERE, args.transcript)}`);
  console.error(`model: ${args.model}\n`);

  const client = new Anthropic();

  const userMessage = [
    "## The user's vault\n",
    vaultContext,
    "\n\n## The transcript to extract from\n",
    transcript.trim(),
  ].join("");

  const resp = await client.messages.create({
    model: args.model,
    max_tokens: 16000,
    thinking: { type: "adaptive" },
    system: systemPrompt,
    messages: [{ role: "user", content: userMessage }],
  });

  const textBlock = resp.content.find((b) => b.type === "text");
  if (!textBlock || textBlock.type !== "text") {
    throw new Error("no text block in response; stop_reason=" + resp.stop_reason);
  }
  const parsed = JSON.parse(extractJson(textBlock.text)) as {
    items: Array<Record<string, string>>;
    skipped: Array<Record<string, string>>;
  };

  render(parsed, groups);

  if (args.out) {
    await writeFile(args.out, JSON.stringify(parsed, null, 2) + "\n");
    console.error(`\nwrote ${args.out}`);
  }
}

function render(
  parsed: { items: Array<Record<string, string>>; skipped: Array<Record<string, string>> },
  groups: string[]
) {
  const known = new Set(groups);
  console.log(`\n=== PROPOSED MEMORIES (${parsed.items.length}) ===\n`);
  for (const it of parsed.items) {
    const g = it.suggested_group || "(unrouted)";
    const bad = it.suggested_group && !known.has(it.suggested_group) ? "  ⚠ NOT AN EXISTING GROUP" : "";
    console.log(`• ${it.label}   [${g}]${bad}   (${it.confidence})`);
    console.log(`  ${it.body.replace(/\n/g, "\n  ")}`);
    console.log(`  ↳ ${it.why}\n`);
  }
  console.log(`=== SKIPPED (${parsed.skipped.length}) ===\n`);
  for (const s of parsed.skipped) {
    console.log(`• ${s.text}\n  ↳ ${s.reason}\n`);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
