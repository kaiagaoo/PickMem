import { useMemo, useState } from "react";
import { useVault } from "../store";
import type { Note } from "../types";
import { MemoryCard } from "./MemoryCard";
import { EmptyState } from "./ui";

// VaultView is the center zone: browse & pick. Cards are grouped under their
// group heading. A group filter (from the left tree) and a text search
// narrow what's shown.
export function VaultView({
  groupFilter,
  onAdd,
  onEdit,
  onDelete,
}: {
  groupFilter: string | null;
  onAdd: (defaultGroup?: string) => void;
  onEdit: (n: Note) => void;
  onDelete: (n: Note) => void;
}) {
  const { state, selected, actions } = useVault();
  const [q, setQ] = useState("");

  const filtered = useMemo(() => {
    const query = q.trim().toLowerCase();
    return state.notes.filter((n) => {
      if (groupFilter) {
        if (n.group !== groupFilter && !n.group.startsWith(groupFilter + "/"))
          return false;
      }
      if (query) {
        return (
          n.label.toLowerCase().includes(query) ||
          n.body.toLowerCase().includes(query) ||
          n.group.toLowerCase().includes(query)
        );
      }
      return true;
    });
  }, [state.notes, groupFilter, q]);

  const grouped = useMemo(() => {
    const m = new Map<string, Note[]>();
    for (const n of filtered) {
      const g = n.group || "(ungrouped)";
      const arr = m.get(g) ?? [];
      arr.push(n);
      m.set(g, arr);
    }
    return [...m.entries()].sort((a, b) => a[0].localeCompare(b[0]));
  }, [filtered]);

  const shownIds = filtered.map((n) => n.id);
  const shownPickedCount = shownIds.filter((id) => selected.has(id)).length;

  return (
    <section className="vault-view">
      <div className="vv-head">
        <h1 className="vv-title mono">{groupFilter ?? "All memories"}</h1>
        <div className="vv-tools">
          <input
            className="search"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search memories…"
          />
          <button className="primary" onClick={() => onAdd(groupFilter ?? undefined)}>
            + Add
          </button>
        </div>
      </div>

      {filtered.length > 0 && (
        <div className="vv-bulk">
          <span className="muted">
            {filtered.length} shown · {shownPickedCount} picked
          </span>
          <button
            className="link"
            onClick={() => actions.toggleGroup(shownIds, true)}
            disabled={shownPickedCount === filtered.length}
          >
            Pick all shown
          </button>
          <span className="sep">·</span>
          <button
            className="link"
            onClick={() => actions.toggleGroup(shownIds, false)}
            disabled={shownPickedCount === 0}
          >
            Clear shown
          </button>
        </div>
      )}

      {state.notes.length === 0 ? (
        <EmptyState
          title="Your vault is ready — add your first memory"
          hint="A memory is one small fact, idea, thought, or reference. You pick which ones your assistant sees."
          action={
            <button className="primary" onClick={() => onAdd()}>
              + Add your first memory
            </button>
          }
        />
      ) : grouped.length === 0 ? (
        <EmptyState title="No memories match your filter." />
      ) : (
        grouped.map(([group, notes]) => {
          const gIds = notes.map((n) => n.id);
          const gPicked = gIds.filter((id) => selected.has(id)).length;
          return (
          <div className="vv-group" key={group}>
            <div className="vv-group-head">
              <input
                type="checkbox"
                className="grp-pick"
                title="Pick / unpick this group"
                checked={gPicked === gIds.length}
                ref={(el) => {
                  if (el) el.indeterminate = gPicked > 0 && gPicked < gIds.length;
                }}
                onChange={() => actions.toggleGroup(gIds, gPicked !== gIds.length)}
              />
              <span className="mono">{group}</span>
              <span className="vv-group-count">
                {gPicked}/{gIds.length}
              </span>
            </div>
            <div className="vv-cards">
              {notes.map((n) => (
                <MemoryCard
                  key={n.id}
                  note={n}
                  picked={selected.has(n.id)}
                  onToggle={() => actions.toggleNote(n.id)}
                  onEdit={() => onEdit(n)}
                  onDelete={() => onDelete(n)}
                />
              ))}
            </div>
          </div>
          );
        })
      )}
    </section>
  );
}
