import { useState } from "react";
import type { Nav, Note } from "../types";
import { buildGroupForest, idsUnder, type GroupNode } from "../treeutil";
import { Menu } from "./ui";

export interface TreeHandlers {
  onNavGroup: (path: string) => void;
  onNavNote: (id: string) => void;
  onToggleSubtree: (ids: string[], select: boolean) => void;
  onNewSubgroup: (parent: string) => void;
  onAddNote: (group: string) => void;
  onRenameGroup: (path: string) => void;
  onDeleteGroup: (path: string) => void;
  onEditNote: (n: Note) => void;
  onDeleteNote: (n: Note) => void;
}

interface Props extends TreeHandlers {
  groups: string[];
  notes: Note[];
  picked: Set<string>;
  nav: Nav;
}

// VaultTree is the Obsidian-style folder explorer: the group hierarchy,
// collapsible, with a subtree pick checkbox and a management menu per group.
// Individual notes live in the main pane, not the tree.
export function VaultTree(props: Props) {
  const { groups } = props;
  const forest = buildGroupForest(groups);
  return (
    <div className="vault-tree">
      {forest.children.map((c) => (
        <Branch key={c.path} node={c} depth={0} {...props} />
      ))}
    </div>
  );
}

function Branch({
  node,
  depth,
  ...h
}: Props & { node: GroupNode; depth: number }) {
  const { notes, picked, nav } = h;
  const [open, setOpen] = useState(depth < 1);
  const ids = idsUnder(notes, node.path);
  const allPicked = ids.length > 0 && ids.every((id) => picked.has(id));
  const somePicked = ids.some((id) => picked.has(id));
  const isSel = nav.kind === "group" && nav.path === node.path;
  const hasKids = node.children.length > 0;

  return (
    <div>
      <div
        className={`tree-row group ${isSel ? "sel" : ""}`}
        style={{ paddingLeft: 6 + depth * 12 }}
      >
        <span
          className="tw-caret"
          onClick={(e) => {
            e.stopPropagation();
            if (hasKids) setOpen((o) => !o);
          }}
        >
          {hasKids ? (open ? "▾" : "▸") : ""}
        </span>
        <input
          type="checkbox"
          className="grp-pick"
          title="Pick / unpick this group"
          disabled={ids.length === 0}
          checked={allPicked}
          ref={(el) => {
            if (el) el.indeterminate = somePicked && !allPicked;
          }}
          onClick={(e) => e.stopPropagation()}
          onChange={() => h.onToggleSubtree(ids, !allPicked)}
        />
        <button className="tree-label" onClick={() => h.onNavGroup(node.path)}>
          <span className="folder-ico">{open ? "▾" : "▸"}</span>
          <span className="tl-name mono">{node.name}</span>
          <span className="tl-count">{ids.length}</span>
        </button>
        <Menu
          className="tree-menu"
          items={[
            { label: "New subgroup…", onClick: () => h.onNewSubgroup(node.path) },
            { label: "Add memory here", onClick: () => h.onAddNote(node.path) },
            { label: "Rename group…", onClick: () => h.onRenameGroup(node.path) },
            { label: "Delete group…", danger: true, onClick: () => h.onDeleteGroup(node.path) },
          ]}
        />
      </div>
      {open &&
        node.children.map((c) => (
          <Branch key={c.path} node={c} depth={depth + 1} {...h} />
        ))}
    </div>
  );
}
