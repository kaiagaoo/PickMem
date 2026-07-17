import { useVault } from "../store";
import type { Note } from "../types";
import { crumbs } from "../treeutil";
import { PickToggle, TypeTag, EmptyState } from "./ui";

// NoteDetail is the main pane for a selected note: a read view with its
// breadcrumb, body, tags, and pick/edit/delete actions. Editing opens the
// same modal used elsewhere.
export function NoteDetail({
  id,
  onNavGroup,
  onEdit,
  onDelete,
}: {
  id: string;
  onNavGroup: (path: string) => void;
  onEdit: (n: Note) => void;
  onDelete: (n: Note) => void;
}) {
  const { state, selected, actions } = useVault();
  const note = state.notes.find((n) => n.id === id);

  if (!note) {
    return (
      <section className="group-view">
        <EmptyState title="This note is no longer here" hint="It may have been deleted or moved." />
      </section>
    );
  }

  const bc = crumbs(note.group);
  const picked = selected.has(note.id);

  return (
    <section className="group-view note-detail">
      <nav className="breadcrumb">
        <button className="bc-link" onClick={() => onNavGroup("")}>
          All memories
        </button>
        {bc.map((c) => (
          <span key={c.path}>
            <span className="bc-sep">/</span>
            <button className="bc-link mono" onClick={() => onNavGroup(c.path)}>
              {c.name}
            </button>
          </span>
        ))}
      </nav>

      <div className="nd-head">
        <PickToggle on={picked} onClick={() => actions.toggleNote(note.id)} label={note.label} />
        <h1 className="nd-title">{note.label}</h1>
        <TypeTag type={note.type} />
      </div>

      <div className="nd-body">{note.body || <span className="muted">No content yet.</span>}</div>

      {note.tags.length > 0 && (
        <div className="nd-tags">
          {note.tags.map((t) => (
            <span key={t} className="mc-tag">
              #{t}
            </span>
          ))}
        </div>
      )}

      <div className="nd-meta mono">
        {note.type} · {note.group} · {note.source}
      </div>

      <div className="nd-actions">
        <button className="primary" onClick={() => onEdit(note)}>
          Edit
        </button>
        <button className="danger" onClick={() => onDelete(note)}>
          Delete
        </button>
      </div>
    </section>
  );
}
