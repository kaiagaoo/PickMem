import { useMemo } from "react";
import { useVault } from "../store";
import type { Note } from "../types";
import { MemoryCard } from "./MemoryCard";
import { EmptyState } from "./ui";

// SearchResults is the filter view: a flat list of every memory matching the
// query (label / body / tags / group), across all groups, shown as pickable
// cards. Mirrors the extension's filter. Each hit shows its group path so you
// know where it lives.
export function SearchResults({
  query,
  onNavNote,
  onEdit,
  onDelete,
}: {
  query: string;
  onNavNote: (id: string) => void;
  onEdit: (n: Note) => void;
  onDelete: (n: Note) => void;
}) {
  const { state, selected, actions } = useVault();
  const q = query.trim().toLowerCase();

  const hits = useMemo(
    () =>
      state.notes.filter((n) => {
        if (!q) return true;
        return (
          n.label.toLowerCase().includes(q) ||
          n.body.toLowerCase().includes(q) ||
          n.group.toLowerCase().includes(q) ||
          n.tags.some((t) => t.toLowerCase().includes(q))
        );
      }),
    [state.notes, q],
  );

  const hitIds = hits.map((n) => n.id);
  const pickedCount = hitIds.filter((id) => selected.has(id)).length;

  return (
    <section className="group-view">
      <div className="gv-notes-head">
        <span className="muted">
          {hits.length} match{hits.length === 1 ? "" : "es"} for “{query.trim()}”
          {hits.length > 0 ? ` · ${pickedCount} picked` : ""}
        </span>
        {hits.length > 0 && (
          <>
            <button
              className="link"
              onClick={() => actions.toggleGroup(hitIds, true)}
              disabled={pickedCount === hits.length}
            >
              Pick all
            </button>
            <span className="sep">·</span>
            <button
              className="link"
              onClick={() => actions.toggleGroup(hitIds, false)}
              disabled={pickedCount === 0}
            >
              Clear
            </button>
          </>
        )}
      </div>

      {hits.length === 0 ? (
        <EmptyState title={`No memories match “${query.trim()}”`} />
      ) : (
        <div className="gv-cards">
          {hits.map((n) => (
            <div className="search-hit" key={n.id}>
              <div className="hit-path mono">{n.group}</div>
              <MemoryCard
                note={n}
                picked={selected.has(n.id)}
                onToggle={() => actions.toggleNote(n.id)}
                onOpen={() => onNavNote(n.id)}
                onEdit={() => onEdit(n)}
                onDelete={() => onDelete(n)}
              />
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
