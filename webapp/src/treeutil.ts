import type { Note } from "./types";

// Pure helpers over the group/note hierarchy, shared by the left tree and the
// drill-down main pane. Groups are "/"-separated paths; the root is "".

export interface GroupNode {
  name: string; // last path segment
  path: string; // full path
  children: GroupNode[];
}

// buildGroupForest builds the nested group tree from all known group paths
// (which include empty folders), so a freshly-created group still appears.
export function buildGroupForest(groups: string[]): GroupNode {
  const root: GroupNode = { name: "", path: "", children: [] };
  for (const g of groups) {
    if (!g) continue;
    let node = root;
    let acc = "";
    for (const seg of g.split("/")) {
      acc = acc ? `${acc}/${seg}` : seg;
      let child = node.children.find((c) => c.path === acc);
      if (!child) {
        child = { name: seg, path: acc, children: [] };
        node.children.push(child);
      }
      node = child;
    }
  }
  const sort = (n: GroupNode) => {
    n.children.sort((a, b) => a.name.localeCompare(b.name));
    n.children.forEach(sort);
  };
  sort(root);
  return root;
}

// findNode returns the group node at path, or the root for "".
export function findNode(root: GroupNode, path: string): GroupNode | null {
  if (path === "") return root;
  let node: GroupNode | undefined = root;
  let acc = "";
  for (const seg of path.split("/")) {
    acc = acc ? `${acc}/${seg}` : seg;
    node = node.children.find((c) => c.path === acc);
    if (!node) return null;
  }
  return node;
}

// notesInGroup: notes filed directly in this exact group.
export function notesInGroup(notes: Note[], path: string): Note[] {
  return notes
    .filter((n) => n.group === path)
    .sort((a, b) => a.label.localeCompare(b.label));
}

// idsUnder: every note id in a group and its descendants (for subtree pick).
export function idsUnder(notes: Note[], path: string): string[] {
  return notes
    .filter((n) => path === "" || n.group === path || n.group.startsWith(path + "/"))
    .map((n) => n.id);
}

// crumbs turns a group path into breadcrumb segments with cumulative paths.
export function crumbs(path: string): { name: string; path: string }[] {
  if (path === "") return [];
  const out: { name: string; path: string }[] = [];
  let acc = "";
  for (const seg of path.split("/")) {
    acc = acc ? `${acc}/${seg}` : seg;
    out.push({ name: seg, path: acc });
  }
  return out;
}
