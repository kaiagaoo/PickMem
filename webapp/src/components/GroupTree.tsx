import { useEffect, useRef, useState } from "react";
import type { Note } from "../types";

interface GNode {
  name: string;
  path: string;
  children: GNode[];
}

function buildGroupTree(groups: string[]): GNode {
  const root: GNode = { name: "", path: "", children: [] };
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
  const sort = (n: GNode) => {
    n.children.sort((a, b) => a.name.localeCompare(b.name));
    n.children.forEach(sort);
  };
  sort(root);
  return root;
}

export interface GroupTreeProps {
  groups: string[];
  notes: Note[];
  selected: string | null; // active filter path
  picked: Set<string>; // currently-picked note ids
  onSelect: (path: string | null) => void;
  onToggleSubtree: (ids: string[], select: boolean) => void;
  onRename: (path: string) => void;
  onDelete: (path: string) => void;
  onAddHere: (path: string) => void;
}

export function GroupTree(props: GroupTreeProps) {
  const { groups, notes, selected, picked, onSelect, onToggleSubtree } = props;
  const tree = buildGroupTree(groups);
  const idsUnder = (path: string) =>
    notes.filter((n) => n.group === path || n.group.startsWith(path + "/")).map((n) => n.id);

  const allIds = notes.map((n) => n.id);
  const allPicked = allIds.length > 0 && allIds.every((id) => picked.has(id));
  const somePicked = allIds.some((id) => picked.has(id));

  return (
    <div className="group-tree">
      <div className={`nav-item all ${selected === null ? "active" : ""}`}>
        <TriToggle
          ids={allIds}
          checked={allPicked}
          indeterminate={somePicked && !allPicked}
          onToggle={onToggleSubtree}
        />
        <button className="ni-btn" onClick={() => onSelect(null)}>
          <span className="ni-label">All memories</span>
          <span className="ni-count">{notes.length}</span>
        </button>
      </div>
      {tree.children.map((c) => (
        <GroupRow key={c.path} node={c} depth={0} idsUnder={idsUnder} {...props} />
      ))}
    </div>
  );
}

function TriToggle({
  ids,
  checked,
  indeterminate,
  onToggle,
}: {
  ids: string[];
  checked: boolean;
  indeterminate: boolean;
  onToggle: (ids: string[], select: boolean) => void;
}) {
  return (
    <input
      type="checkbox"
      className="grp-pick"
      title={ids.length ? "Pick / unpick this group" : "No memories here yet"}
      disabled={ids.length === 0}
      checked={checked}
      ref={(el) => {
        if (el) el.indeterminate = indeterminate;
      }}
      onClick={(e) => e.stopPropagation()}
      onChange={() => onToggle(ids, !checked)}
    />
  );
}

function GroupRow({
  node,
  depth,
  idsUnder,
  selected,
  picked,
  onSelect,
  onToggleSubtree,
  onRename,
  onDelete,
  onAddHere,
}: GroupTreeProps & { node: GNode; depth: number; idsUnder: (p: string) => string[] }) {
  const [open, setOpen] = useState(depth < 1);
  const hasChildren = node.children.length > 0;
  const ids = idsUnder(node.path);
  const allPicked = ids.length > 0 && ids.every((id) => picked.has(id));
  const somePicked = ids.some((id) => picked.has(id));

  return (
    <div>
      <div
        className={`nav-item group ${selected === node.path ? "active" : ""}`}
        style={{ paddingLeft: 8 + depth * 12 }}
      >
        <span
          className="tw-caret"
          onClick={(e) => {
            e.stopPropagation();
            if (hasChildren) setOpen((o) => !o);
          }}
        >
          {hasChildren ? (open ? "▾" : "▸") : ""}
        </span>
        <TriToggle
          ids={ids}
          checked={allPicked}
          indeterminate={somePicked && !allPicked}
          onToggle={onToggleSubtree}
        />
        <button className="ni-btn" onClick={() => onSelect(node.path)}>
          <span className="ni-label mono">{node.name}</span>
          <span className="ni-count">{ids.length}</span>
        </button>
        <GroupMenu
          path={node.path}
          onRename={onRename}
          onDelete={onDelete}
          onAddHere={onAddHere}
        />
      </div>
      {open &&
        node.children.map((c) => (
          <GroupRow
            key={c.path}
            node={c}
            depth={depth + 1}
            idsUnder={idsUnder}
            groups={[]}
            notes={[]}
            selected={selected}
            picked={picked}
            onSelect={onSelect}
            onToggleSubtree={onToggleSubtree}
            onRename={onRename}
            onDelete={onDelete}
            onAddHere={onAddHere}
          />
        ))}
    </div>
  );
}

function GroupMenu({
  path,
  onRename,
  onDelete,
  onAddHere,
}: {
  path: string;
  onRename: (p: string) => void;
  onDelete: (p: string) => void;
  onAddHere: (p: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  const act = (fn: () => void) => {
    setOpen(false);
    fn();
  };

  return (
    <div className="group-menu-wrap" ref={ref}>
      <button
        className="grp-menu-btn"
        title="Group actions"
        onClick={(e) => {
          e.stopPropagation();
          setOpen((o) => !o);
        }}
      >
        ⋯
      </button>
      {open && (
        <div className="grp-menu">
          <button onClick={() => act(() => onAddHere(path))}>Add memory here</button>
          <button onClick={() => act(() => onRename(path))}>Rename group…</button>
          <button className="danger" onClick={() => act(() => onDelete(path))}>
            Delete group…
          </button>
        </div>
      )}
    </div>
  );
}
