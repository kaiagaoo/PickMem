import type { Note } from "../types";
import { PickToggle, TypeTag } from "./ui";

// MemoryCard is the core object of the app — one memory as an index card.
// Clicking the card (or its toggle) picks/unpicks it; picking is the
// high-frequency action, so the whole card is the hit target. Edit/Delete
// are explicit and stop propagation.
export function MemoryCard({
  note,
  picked,
  onToggle,
  onEdit,
  onDelete,
}: {
  note: Note;
  picked: boolean;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div
      className={`memory-card ${picked ? "picked" : ""}`}
      onClick={onToggle}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === " " || e.key === "Enter") {
          e.preventDefault();
          onToggle();
        }
      }}
    >
      <PickToggle on={picked} onClick={onToggle} label={note.label} />
      <div className="mc-main">
        <div className="mc-top">
          <span className="mc-label">{note.label}</span>
          <TypeTag type={note.type} />
        </div>
        {note.body && <div className="mc-body">{note.body}</div>}
        {note.tags.length > 0 && (
          <div className="mc-tags">
            {note.tags.map((t) => (
              <span key={t} className="mc-tag">
                #{t}
              </span>
            ))}
          </div>
        )}
      </div>
      <div className="mc-actions" onClick={(e) => e.stopPropagation()}>
        <button className="ghost" onClick={onEdit}>
          Edit
        </button>
        <button className="danger" onClick={onDelete}>
          Delete
        </button>
      </div>
    </div>
  );
}
