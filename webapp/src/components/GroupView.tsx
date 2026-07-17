import { useVault } from "../store";
import type { Note } from "../types";
import {
  buildGroupForest,
  crumbs,
  findNode,
  idsUnder,
  notesInGroup,
} from "../treeutil";
import { MemoryCard } from "./MemoryCard";
import { EmptyState, Menu } from "./ui";

export interface GroupViewHandlers {
  onNavGroup: (path: string) => void;
  onNavNote: (id: string) => void;
  onNewSubgroup: (parent: string) => void;
  onAddNote: (group: string) => void;
  onRenameGroup: (path: string) => void;
  onDeleteGroup: (path: string) => void;
  onEditNote: (n: Note) => void;
  onDeleteNote: (n: Note) => void;
}

// GroupView is the drill-down main pane: a breadcrumb, the current group's
// subgroups (as folders you can open) and its own notes (as cards), plus
// group management in the header.
export function GroupView({ path, h }: { path: string; h: GroupViewHandlers }) {
  const { state, selected, actions } = useVault();
  const forest = buildGroupForest(state.groups);
  const node = findNode(forest, path);
  const subgroups = node?.children ?? [];
  const ownNotes = notesInGroup(state.notes, path);

  const shownIds = ownNotes.map((n) => n.id);
  const shownPicked = shownIds.filter((id) => selected.has(id)).length;
  const isRoot = path === "";
  const bc = crumbs(path);

  return (
    <section className="group-view">
      {/* At the root the breadcrumb would just repeat the "All memories"
          title, so it only shows once you've drilled into a group. */}
      {!isRoot && (
        <nav className="breadcrumb">
          <button className="bc-link" onClick={() => h.onNavGroup("")}>
            All memories
          </button>
          {bc.map((c, i) => (
            <span key={c.path}>
              <span className="bc-sep">/</span>
              <button
                className={i === bc.length - 1 ? "bc-cur mono" : "bc-link mono"}
                onClick={() => h.onNavGroup(c.path)}
              >
                {c.name}
              </button>
            </span>
          ))}
        </nav>
      )}

      <div className="gv-head">
        <h1 className="gv-title">{isRoot ? "All memories" : bc[bc.length - 1].name}</h1>
        <div className="gv-actions">
          <button className="ghost" onClick={() => h.onNewSubgroup(path)}>
            + Subgroup
          </button>
          <button className="primary" onClick={() => h.onAddNote(path || "")}>
            + Memory
          </button>
          {!isRoot && (
            <Menu
              items={[
                { label: "Rename group…", onClick: () => h.onRenameGroup(path) },
                { label: "Delete group…", danger: true, onClick: () => h.onDeleteGroup(path) },
              ]}
            />
          )}
        </div>
      </div>

      {subgroups.length > 0 && (
        <div className="folder-grid">
          {subgroups.map((sg) => {
            const ids = idsUnder(state.notes, sg.path);
            const p = ids.filter((id) => selected.has(id)).length;
            return (
              <div key={sg.path} className="folder-card" onClick={() => h.onNavGroup(sg.path)}>
                <input
                  type="checkbox"
                  className="grp-pick"
                  title="Pick / unpick this group"
                  disabled={ids.length === 0}
                  checked={p === ids.length && ids.length > 0}
                  ref={(el) => {
                    if (el) el.indeterminate = p > 0 && p < ids.length;
                  }}
                  onClick={(e) => e.stopPropagation()}
                  onChange={() => actions.toggleGroup(ids, p !== ids.length)}
                />
                <span className="folder-ico">▸</span>
                <span className="folder-name">{sg.name}</span>
                <span className="folder-count">
                  {p}/{ids.length}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {ownNotes.length > 0 && (
        <>
          <div className="gv-notes-head">
            <span className="muted">
              {ownNotes.length} note{ownNotes.length === 1 ? "" : "s"} here · {shownPicked} picked
            </span>
            <button
              className="link"
              onClick={() => actions.toggleGroup(shownIds, true)}
              disabled={shownPicked === ownNotes.length}
            >
              Pick all
            </button>
            <span className="sep">·</span>
            <button
              className="link"
              onClick={() => actions.toggleGroup(shownIds, false)}
              disabled={shownPicked === 0}
            >
              Clear
            </button>
          </div>
          <div className="gv-cards">
            {ownNotes.map((n) => (
              <MemoryCard
                key={n.id}
                note={n}
                picked={selected.has(n.id)}
                onToggle={() => actions.toggleNote(n.id)}
                onOpen={() => h.onNavNote(n.id)}
                onEdit={() => h.onEditNote(n)}
                onDelete={() => h.onDeleteNote(n)}
              />
            ))}
          </div>
        </>
      )}

      {subgroups.length === 0 && ownNotes.length === 0 && (
        <EmptyState
          title={isRoot ? "Your vault is empty" : "This group is empty"}
          hint="Add a memory here, or create a subgroup to organize further."
          action={
            <button className="primary" onClick={() => h.onAddNote(path || "")}>
              + Add a memory
            </button>
          }
        />
      )}
    </section>
  );
}
